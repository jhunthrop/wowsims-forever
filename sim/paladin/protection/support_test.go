package protection

import (
	"fmt"
	"strings"
	"testing"
	"time"

	_ "github.com/wowsims/classic/sim/common" // imported to get item effects included.
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/paladin"
	googleProto "google.golang.org/protobuf/proto"
)

const gearDir = "../../../ui/protection_paladin/gear_sets"

// talentsOf builds a talent string with exactly the given ranks, by the
// talent proto's own field names, so a tree layout change cannot retarget
// a test.
func talentsOf(t testing.TB, ranks map[string]int) string {
	t.Helper()
	build, err := core.TalentsStringFromRanks((&proto.PaladinTalents{}).ProtoReflect(), paladin.TalentTreeSizes, ranks)
	if err != nil {
		t.Fatalf("building the talent string: %v", err)
	}
	return build
}

// shortRun is one small, controlled simulation: a tank with a chosen
// talent set and rotation in front of a boss with a chosen swing. Each
// behaviour test varies only what it is about.
type shortRun struct {
	Talents    map[string]int
	Options    *proto.PaladinOptions
	Rotation   string // APL JSON; empty for a tank that does nothing
	Equipment  *proto.EquipmentSpec
	Boss       *proto.Target
	Targets    int
	Seconds    float64
	Iterations int32
}

func (run shortRun) withDefaults() shortRun {
	if run.Options == nil {
		run.Options = &proto.PaladinOptions{RighteousFury: true}
	}
	if run.Rotation == "" {
		run.Rotation = `{"type": "TypeAPL"}`
	}
	if run.Equipment == nil {
		run.Equipment = core.GetGearSet(gearDir, harnessGearSet).GearSet
	}
	if run.Boss == nil {
		run.Boss = harnessBoss(harnessBossMinDamage)
	}
	if run.Targets == 0 {
		run.Targets = 1
	}
	if run.Seconds == 0 {
		run.Seconds = 30
	}
	if run.Iterations == 0 {
		run.Iterations = 1
	}
	return run
}

func (run shortRun) raid(t testing.TB) *proto.Raid {
	t.Helper()
	player := core.WithSpec(&proto.Player{
		Class:           proto.Class_ClassPaladin,
		Race:            proto.Race_RaceHuman,
		Level:           harnessLevel,
		Equipment:       run.Equipment,
		Buffs:           core.FullBuffs.Player,
		TalentsString:   talentsOf(t, run.Talents),
		Rotation:        core.APLRotationFromJsonString(run.Rotation),
		InFrontOfTarget: true,
		HealingModel:    harnessHealingModel(),
		ReactionTimeMs:  150,
	}, &proto.Player_ProtectionPaladin{ProtectionPaladin: &proto.ProtectionPaladin{Options: run.Options}})
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, nil)
	raid.Tanks = append(raid.Tanks, &proto.UnitReference{Type: proto.UnitReference_Player, Index: 0})
	return raid
}

func (run shortRun) encounter() *proto.Encounter {
	targets := make([]*proto.Target, run.Targets)
	for i := range targets {
		targets[i] = run.Boss
	}
	return &proto.Encounter{Duration: run.Seconds, Targets: targets}
}

// metrics runs the simulation and returns the tank's metrics, summed over
// the iterations (the per-action and per-aura figures are totals).
func (run shortRun) metrics(t testing.TB) *proto.UnitMetrics {
	t.Helper()
	run = run.withDefaults()
	result := core.RunRaidSim(&proto.RaidSimRequest{
		Raid:       run.raid(t),
		Encounter:  run.encounter(),
		SimOptions: &proto.SimOptions{Iterations: run.Iterations, IsTest: true, RandomSeed: 101},
	})
	if result.Error != nil {
		t.Fatalf("sim failed: %s", result.Error.Message)
	}
	return result.RaidMetrics.Parties[0].Players[0]
}

// fight is a simulation built, reset and prepulled, with a boss that never
// swings of its own: a test delivers exactly the hits it is about, through
// the whole damage pipeline, and advances time when it needs to.
type fight struct {
	sim    *core.Simulation
	tank   *ProtectionPaladin
	boss   *core.Unit
	attack *core.Spell // a melee swing, the hit most of the tank's talents answer
	magic  *core.Spell // a Holy spell, which no melee talent answers
}

// start builds the fight. The boss is the harness boss with its swing
// turned off.
func (run shortRun) start(t testing.TB) *fight {
	t.Helper()
	run = run.withDefaults()
	run.Boss = googleProto.Clone(run.Boss).(*proto.Target)
	run.Boss.SwingSpeed = 0

	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       run.raid(t),
		Encounter:  run.encounter(),
		SimOptions: &proto.SimOptions{RandomSeed: 101, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()

	tank, ok := sim.Raid.Parties[0].Players[0].(*ProtectionPaladin)
	if !ok {
		t.Fatalf("player agent is %T, want *ProtectionPaladin", sim.Raid.Parties[0].Players[0])
	}
	bossUnit := sim.Encounter.TargetUnits[0]
	attack := bossUnit.RegisterSpell(core.SpellConfig{
		ActionID:         core.ActionID{OtherID: proto.OtherAction_OtherActionAttack},
		SpellSchool:      core.SpellSchoolPhysical,
		DefenseType:      core.DefenseTypeMelee,
		ProcMask:         core.ProcMaskMeleeMHAuto,
		Flags:            core.SpellFlagMeleeMetrics,
		DamageMultiplier: 1,
		ThreatMultiplier: 1,
	})
	magic := bossUnit.RegisterSpell(core.SpellConfig{
		ActionID:         core.ActionID{OtherID: proto.OtherAction_OtherActionAttack, Tag: 1},
		SpellSchool:      core.SpellSchoolHoly,
		DefenseType:      core.DefenseTypeMagic,
		ProcMask:         core.ProcMaskSpellDamage,
		DamageMultiplier: 1,
		ThreatMultiplier: 1,
	})
	return &fight{sim: sim, tank: tank, boss: bossUnit, attack: attack, magic: magic}
}

// hit delivers one boss attack of the given outcome and raw damage to the
// tank. The raw damage still goes through armor and the other damage
// taken modifiers, and the absorbs, before it reaches the result.
func (f *fight) hit(outcome core.HitOutcome, rawDamage float64) *core.SpellResult {
	return f.hitWith(f.attack, outcome, rawDamage)
}

func (f *fight) hitWith(attack *core.Spell, outcome core.HitOutcome, rawDamage float64) *core.SpellResult {
	return attack.CalcAndDealDamage(f.sim, &f.tank.Unit, rawDamage,
		func(_ *core.Simulation, result *core.SpellResult, _ *core.AttackTable) {
			result.Outcome = outcome
		})
}

// advance steps the simulation until the given time has passed.
func (f *fight) advance(duration time.Duration) {
	until := f.sim.CurrentTime + duration
	for f.sim.CurrentTime < until {
		if f.sim.Step() {
			return
		}
	}
}

// logLines starts capturing the simulation's log and returns what has
// been logged so far on each call.
func (f *fight) logLines() func() []string {
	var lines []string
	f.sim.Log = func(message string, vals ...interface{}) {
		lines = append(lines, fmt.Sprintf(message, vals...))
	}
	return func() []string { return lines }
}

func countContaining(lines []string, needle string) int {
	var count int
	for _, line := range lines {
		if strings.Contains(line, needle) {
			count++
		}
	}
	return count
}

// spellOf is the tank's registered spell by id, failing the test without it.
func (f *fight) spellOf(t testing.TB, spellID int32) *core.Spell {
	t.Helper()
	spell := f.tank.GetSpell(core.ActionID{SpellID: spellID})
	if spell == nil {
		t.Fatalf("spell %d is not registered", spellID)
	}
	return spell
}

func actionCasts(metrics *proto.UnitMetrics, spellID int32) float64 {
	var casts float64
	for _, action := range metrics.Actions {
		if action.Id.GetSpellId() != spellID {
			continue
		}
		for _, target := range action.Targets {
			casts += float64(target.Casts)
		}
	}
	return casts
}

func actionHits(metrics *proto.UnitMetrics, spellID int32) float64 {
	var hits float64
	for _, action := range metrics.Actions {
		if action.Id.GetSpellId() != spellID {
			continue
		}
		for _, target := range action.Targets {
			hits += float64(target.Hits)
		}
	}
	return hits
}

func actionShielding(metrics *proto.UnitMetrics, spellID int32) float64 {
	var shielding float64
	for _, action := range metrics.Actions {
		if action.Id.GetSpellId() == spellID {
			for _, target := range action.Targets {
				shielding += target.Shielding
			}
		}
	}
	return shielding
}

func auraProcs(metrics *proto.UnitMetrics, spellID int32) float64 {
	for _, aura := range metrics.Auras {
		if aura.Id.GetSpellId() == spellID {
			return aura.ProcsAvg
		}
	}
	return 0
}

func auraUptime(metrics *proto.UnitMetrics, spellID int32) float64 {
	for _, aura := range metrics.Auras {
		if aura.Id.GetSpellId() == spellID {
			return aura.UptimeSecondsAvg
		}
	}
	return 0
}

// resourceEvents counts the events a spell logged against the tank's mana.
func resourceEvents(metrics *proto.UnitMetrics, spellID int32) float64 {
	for _, resource := range metrics.Resources {
		if resource.Type == proto.ResourceType_ResourceTypeMana && resource.Id.GetSpellId() == spellID {
			return float64(resource.Events)
		}
	}
	return 0
}

func resourceGain(metrics *proto.UnitMetrics, spellID int32) float64 {
	for _, resource := range metrics.Resources {
		if resource.Type == proto.ResourceType_ResourceTypeMana && resource.Id.GetSpellId() == spellID {
			return resource.Gain
		}
	}
	return 0
}

// rapidBoss swings often enough that a short run sees many hits.
func rapidBoss(swingSeconds float64) *proto.Target {
	boss := harnessBoss(harnessBossMinDamage)
	boss.SwingSpeed = swingSeconds
	return boss
}

// prepullCast is a prepull cast of a spell, seconds before the pull.
func prepullCast(spellID int32, secondsBefore string) string {
	return fmt.Sprintf(`{"action": {"castSpell": {"spellId": {"spellId": %d}}}, "doAtValue": {"const": {"val": "-%ss"}}}`, spellID, secondsBefore)
}

// cast is a priority line casting a spell whenever it can.
func cast(spellID int32) string {
	return fmt.Sprintf(`{"action": {"castSpell": {"spellId": {"spellId": %d}}}}`, spellID)
}

func rotationOf(prepull []string, priority []string) string {
	return fmt.Sprintf(`{"type": "TypeAPL", "prepullActions": [%s], "priorityList": [%s]}`,
		strings.Join(prepull, ","), strings.Join(priority, ","))
}
