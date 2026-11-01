package hunter

import (
	"math"
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// publishedSurvivalTalents is the band-60 Survival build the site publishes
// (data/builds/1.60.1.70707/bis/hunter-survival.json), Resourcefulness 2/2.
const publishedSurvivalTalents = "0000000000000000-3250050010000000-500230231051222151"

func buildPublishedSurvivalHunter(t *testing.T) (*core.Simulation, *Hunter) {
	t.Helper()
	return buildLevel60Hunter(t, publishedSurvivalTalents)
}

// buildLevel60Hunter is a level-60, no-gear, unbuffed orc hunter in melee range
// with the given talents.
func buildLevel60Hunter(t *testing.T, talents string) (*core.Simulation, *Hunter) {
	t.Helper()
	player := core.WithSpec(&proto.Player{
		Class:              proto.Class_ClassHunter,
		Race:               proto.Race_RaceOrc,
		Level:              60,
		Equipment:          &proto.EquipmentSpec{},
		TalentsString:      talents,
		DistanceFromTarget: 5,
	}, P1PlayerOptions)
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter:  &proto.Encounter{Duration: 60, Targets: []*proto.Target{core.DefaultTargetProtoLvl60}},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	agent, ok := sim.Raid.Parties[0].Players[0].(HunterAgent)
	if !ok {
		t.Fatal("the raid's first player is not a hunter agent")
	}
	return sim, agent.GetHunter()
}

// castingTickGain is the mana one regen tick restores inside the five-second
// rule, from a pool well under its cap.
func castingTickGain(sim *core.Simulation, hunter *Hunter, metrics *core.ResourceMetrics) float64 {
	hunter.SpendMana(sim, hunter.CurrentMana()/2, metrics)
	hunter.PseudoStats.FiveSecondRuleRefreshTime = sim.CurrentTime + 5*time.Second
	before := hunter.CurrentMana()
	hunter.ManaTick(sim)
	return hunter.CurrentMana() - before
}

// TestResourcefulnessRegenContinuesWhileCasting: Resourcefulness 440529
// "grants your melee and ranged ability Critical Strikes a 100% chance to
// allow 50% of your Mana Regeneration to continue while casting for 30 sec".
// The aura raised PseudoStats.SpiritRegenRateCasting, but the unit's regen
// per tick is cached (core UpdateManaRegenRates) and nothing refreshed it, so
// the casting tick never grew.
func TestResourcefulnessRegenContinuesWhileCasting(t *testing.T) {
	sim, hunter := buildPublishedSurvivalHunter(t)
	aura := hunter.GetAura("Resourcefulness")
	if aura == nil {
		t.Fatal("the published Survival build registers no Resourcefulness aura")
	}
	metrics := hunter.NewManaMetrics(core.ActionID{SpellID: 440529, Tag: 99})

	without := castingTickGain(sim, hunter, metrics)
	aura.Activate(sim)
	with := castingTickGain(sim, hunter, metrics)

	spirit := hunter.SpiritManaRegenPerSecondDefault()
	if hunter.SpiritManaRegenPerSecond != nil {
		spirit = hunter.SpiritManaRegenPerSecond()
	}
	want := 0.5 * spirit * hunter.PseudoStats.SpiritRegenMultiplier * 2 // half the spirit regen, one 2 s tick
	if got := with - without; want <= 0 || math.Abs(got-want) > 0.01 {
		t.Errorf("Resourcefulness adds %.2f mana a casting tick, want %.2f (50%% of %.2f spirit regen a second)", got, want, spirit)
	}
}

// TestResourcefulnessProcsOnlyFromAbilityCrits: SpellAuraOptions 242512 gives
// 440529 ProcTypeMask 272 (melee and ranged ability damage, 16 | 256), as its
// text says ("your melee and ranged ability Critical Strikes"). A white swing's
// crit or a bleed tick's is not an ability crit.
func TestResourcefulnessProcsOnlyFromAbilityCrits(t *testing.T) {
	sim, hunter := buildPublishedSurvivalHunter(t)
	trigger := hunter.GetAura("Resourcefulness Trigger")
	regen := hunter.GetAura("Resourcefulness")
	if trigger == nil || regen == nil || hunter.RaptorStrikeHit == nil {
		t.Fatal("the published Survival build is missing Resourcefulness or Raptor Strike")
	}
	crit := &core.SpellResult{Outcome: core.OutcomeCrit, Target: sim.Encounter.TargetUnits[0]}

	trigger.OnSpellHitDealt(trigger, sim, hunter.AutoAttacks.MHAuto(), crit)
	if regen.IsActive() {
		t.Error("a white melee crit started Resourcefulness's regen")
	}
	trigger.OnSpellHitDealt(trigger, sim, hunter.RaptorStrikeHit, crit)
	if !regen.IsActive() {
		t.Error("a Raptor Strike crit did not start Resourcefulness's regen")
	}
}

// TestBestialDisciplineLetsHalfTheManaRegenContinueWhileCasting: 19590 (rank 2)
// carries aura 134 (SPELL_AURA_MOD_MANA_REGEN_INTERRUPT) at 50 beside the pet's
// focus regen; its text: "allows 50% of your Mana Regeneration to continue
// while casting". Only the focus half was modelled.
func TestBestialDisciplineLetsHalfTheManaRegenContinueWhileCasting(t *testing.T) {
	_, hunter := buildLevel60Hunter(t, "5420001505001251-0052451001200000-000000000000000000")
	if hunter.Talents.BestialDiscipline != 2 {
		t.Fatalf("the published Beast Mastery build has Bestial Discipline %d, want 2", hunter.Talents.BestialDiscipline)
	}
	if got := hunter.PseudoStats.SpiritRegenRateCasting; math.Abs(got-0.5) > 1e-9 {
		t.Errorf("Bestial Discipline 2/2 lets %.2f of spirit regen continue while casting, want 0.50", got)
	}
}
