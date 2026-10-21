package feral

import (
	"testing"
	"time"

	_ "github.com/wowsims/classic/sim/common"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
)

// newFeralDruidSimAtLevel builds a Feral druid at the given level against a
// level-matched dummy target (see claw_level_test.go: a fixed level-63 raid
// boss makes a low-level attacker's specials miss/dodge almost every time,
// which would make cast/damage assertions here flaky), and returns the
// built agent, a reset+pre-pulled sim, and the target unit.
func newFeralDruidSimAtLevel(t *testing.T, level int32) (*FeralDruid, *core.Simulation, *core.Unit) {
	t.Helper()

	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassDruid,
			Race:               proto.Race_RaceTauren,
			Level:              level,
			Equipment:          &proto.EquipmentSpec{}, // no gear: this fork's item database isn't generated in this test environment.
			Buffs:              core.FullBuffs.Player,
			TalentsString:      P1Talents,
			DistanceFromTarget: 5,
		},
		PlayerOptionsMonoCat,
	)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)

	target := &proto.Target{
		Level: level,
		Stats: stats.Stats{
			stats.Armor: 100,
		}.ToFloatArray(),
	}

	sim := core.NewSim(&proto.RaidSimRequest{
		Raid: raid,
		Encounter: &proto.Encounter{
			Duration: 60,
			Targets:  []*proto.Target{target},
		},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()

	built, ok := sim.Raid.Parties[0].Players[0].(*FeralDruid)
	if !ok {
		t.Fatal("the raid's first player is not a *FeralDruid")
	}

	return built, sim, sim.Encounter.TargetUnits[0]
}

// TestRavageOnlyCastableWhileProwling rebuilds the finding this lane closes:
// the engine had no Ravage at all, so a Feral rotation could never open
// with it. Ravage's own ExtraCastCondition (ravage.go) must gate the cast
// on Prowl exactly the way sim/rogue/ambush.go gates Ambush on Stealth.
func TestRavageOnlyCastableWhileProwling(t *testing.T) {
	built, sim, target := newFeralDruidSimAtLevel(t, 60)

	if built.Ravage == nil {
		t.Fatal("level-60 Feral druid has no Ravage registered")
	}
	if got, want := built.Ravage.ActionID.SpellID, int32(9867); got != want {
		t.Errorf("Ravage spell ID = %d, want max rank %d", got, want)
	}

	if built.ProwlAura.IsActive() {
		t.Fatal("ProwlAura is active before Prowl was ever cast")
	}
	if built.Ravage.CanCast(sim, target) {
		t.Fatal("Ravage is castable while not Prowling")
	}

	built.ProwlAura.Activate(sim)
	if !built.Ravage.CanCast(sim, target) {
		t.Fatal("Ravage is not castable while Prowling")
	}
}

// TestRavageDealsDamageGrantsComboPointAndBreaksProwl covers the other
// three parts of this lane's brief: a landed Ravage deals damage in the
// rank's band, grants a combo point (client data: spellconst effect
// 30/misc_value 4/amount 1, the same single-combo-point flag every other
// Cat Form builder carries -- not the 2 combo points live retail Ravage
// grants, since this fork follows its own client data over out-of-game
// knowledge), and breaks Prowl on the hit. P1Talents carries 2/2 Blood
// Frenzy (proto field PrimalFury, node 104947; talents.go's
// applyBloodFrenzy), a 100% chance for a Cat Form builder's crit to add a
// second combo point, and this fixed seed's Ravage crits, so the landed
// hit here grants 2, not 1.
func TestRavageDealsDamageGrantsComboPointAndBreaksProwl(t *testing.T) {
	built, sim, target := newFeralDruidSimAtLevel(t, 60)

	built.ProwlAura.Activate(sim)
	if built.ComboPoints() != 0 {
		t.Fatalf("combo points = %d before any builder landed, want 0", built.ComboPoints())
	}

	// Drive ApplyEffects directly (the real code a Cast would run), the
	// same way claw_level_test.go does, so the assertion isn't gated on
	// GCD/energy bookkeeping that's out of scope here.
	built.Ravage.ApplyEffects(sim, target, built.Ravage.Spell)

	metrics := built.Ravage.SpellMetrics[target.UnitIndex]
	if metrics.Hits+metrics.Crits == 0 {
		t.Fatalf("Ravage outcome was neither a hit nor a crit (misses=%d)", metrics.Misses)
	}
	if metrics.TotalDamage <= 0 {
		t.Errorf("Ravage dealt %v damage, want > 0", metrics.TotalDamage)
	}
	if got, want := built.ComboPoints(), int32(2); got != want {
		t.Errorf("combo points after a landed, critical Ravage with 2/2 Blood Frenzy = %d, want %d", got, want)
	}
	if built.ProwlAura.IsActive() {
		t.Fatal("ProwlAura is still active after a landed Ravage; Ravage must break Prowl")
	}
}

// TestRavageRankOneAtLevel32 is the level-boundary case: a level-32 Feral
// druid has just learned Ravage's rank 1 (id 6785, spellranks.json) and
// nothing higher, and casting it must still land and deal damage.
func TestRavageRankOneAtLevel32(t *testing.T) {
	const wantLevel = 32
	built, sim, target := newFeralDruidSimAtLevel(t, wantLevel)

	if built.Ravage == nil {
		t.Fatal("level-32 Feral druid has no Ravage registered")
	}
	if got, want := built.Ravage.ActionID.SpellID, int32(6785); got != want {
		t.Errorf("Ravage spell ID = %d, want rank 1 %d", got, want)
	}

	// Several casts, rather than one, so a single unlucky dodge/miss roll
	// can't make this test flaky (mirrors claw_level_test.go's
	// TestClawLevel20HasRankOne).
	const attempts = 10
	for i := 0; i < attempts; i++ {
		built.ProwlAura.Activate(sim)
		built.Ravage.ApplyEffects(sim, target, built.Ravage.Spell)
	}

	metrics := built.Ravage.SpellMetrics[target.UnitIndex]
	if metrics.Hits+metrics.Crits == 0 {
		t.Fatalf("Ravage landed 0 of %d casts (misses=%d dodges=%d parries=%d)", attempts, metrics.Misses, metrics.Dodges, metrics.Parries)
	}
	if metrics.TotalDamage <= 0 {
		t.Errorf("Ravage dealt %v damage, want > 0", metrics.TotalDamage)
	}
}

// TestProwlOnlyCastableOutOfCombatAndNotWhileAlreadyProwling covers Prowl's
// own ExtraCastCondition (prowl.go), mirroring sim/rogue/stealth.go's own
// sim.CurrentTime < 0 gate for Stealth: Prowl is a prepull-only opener,
// never available once combat has started, and cannot be re-cast while
// already Prowling (there is no Cat Form equivalent of Vanish).
func TestProwlOnlyCastableOutOfCombatAndNotWhileAlreadyProwling(t *testing.T) {
	built, sim, target := newFeralDruidSimAtLevel(t, 60)

	if built.Prowl == nil {
		t.Fatal("level-60 Feral druid has no Prowl registered")
	}

	// sim.Reset()+sim.PrePull() with no rotation-supplied prepull actions
	// leaves CurrentTime at 0 -- i.e. combat has (nominally) already
	// started, the in-combat case.
	if built.Prowl.CanCast(sim, target) {
		t.Fatal("Prowl is castable at sim.CurrentTime == 0 (in combat)")
	}

	sim.CurrentTime = -time.Second
	if !built.Prowl.CanCast(sim, target) {
		t.Fatal("Prowl is not castable before the pull (sim.CurrentTime < 0)")
	}

	built.ProwlAura.Activate(sim)
	if built.Prowl.CanCast(sim, target) {
		t.Fatal("Prowl is castable while already Prowling")
	}
}

// TestProwlBreaksOnClaw is the general half of "prowl breaks on the first
// damaging ability": every Cat Form special that can open a pull, not
// only Ravage, must break Prowl on a landed hit (mirrors how every rogue
// special in sim/rogue calls BreakStealth, not only Ambush).
func TestProwlBreaksOnClaw(t *testing.T) {
	built, sim, target := newFeralDruidSimAtLevel(t, 60)

	built.ProwlAura.Activate(sim)
	if !built.ProwlAura.IsActive() {
		t.Fatal("ProwlAura did not activate")
	}

	built.Claw.ApplyEffects(sim, target, built.Claw.Spell)

	if built.ProwlAura.IsActive() {
		t.Fatal("ProwlAura is still active after a landed Claw; every Cat Form special must break Prowl")
	}
}
