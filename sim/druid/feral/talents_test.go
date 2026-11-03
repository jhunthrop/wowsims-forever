package feral

import (
	"math"
	"testing"

	_ "github.com/wowsims/classic/sim/common"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
	"github.com/wowsims/classic/sim/druid"
)

// feralTalentsString builds a full-width talent string with every talent
// zero except the ones named, keyed by the generated proto field name
// (core.TalentsStringFromRanks), so a tree layout change cannot retarget
// a test the way a typed field number did.
func feralTalentsString(t *testing.T, ranks map[string]int) string {
	t.Helper()
	str, err := core.TalentsStringFromRanks((&proto.DruidTalents{}).ProtoReflect(), druid.TalentTreeSizes, ranks)
	if err != nil {
		t.Fatal(err)
	}
	return str
}

// newFeralDruidSimWithTalents is newFeralDruidSimAtLevel (ravage_test.go)
// with an arbitrary talents string instead of the package's shared
// P1Talents fixture, so a single talent can be isolated from the rest of
// a real build.
func newFeralDruidSimWithTalents(t *testing.T, level int32, talents string) (*FeralDruid, *core.Simulation, *core.Unit) {
	t.Helper()
	return newFeralDruidSimWearing(t, level, talents, &proto.EquipmentSpec{})
}

// newFeralDruidSimWearing is newFeralDruidSimWithTalents wearing the given
// equipment.
func newFeralDruidSimWearing(t *testing.T, level int32, talents string, equipment *proto.EquipmentSpec) (*FeralDruid, *core.Simulation, *core.Unit) {
	t.Helper()

	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassDruid,
			Race:               proto.Race_RaceTauren,
			Level:              level,
			Equipment:          equipment,
			Buffs:              core.FullBuffs.Player,
			TalentsString:      talents,
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

// TestShreddingAttacksReducesShredEnergyCost covers talents.go's note in
// shred.go: "Reduces the Energy cost of your Shred ability by 6/12/18."
// (node 104945, proto field ShreddingAttacks; Feral tier 2 in the live tree).
func TestShreddingAttacksReducesShredEnergyCost(t *testing.T) {
	// Poised (poised_test.go) takes 10 off the fight's first ability; the
	// base cost is read with it spent.
	base, baseSim, _ := newFeralDruidSimWithTalents(t, 60, feralTalentsString(t, nil))
	if base.Shred == nil {
		t.Fatal("level-60 Feral druid has no Shred registered")
	}
	base.PoisedAura.Deactivate(baseSim)
	if got, want := base.Shred.Cost.GetCurrentCost(), 60.0; got != want {
		t.Fatalf("untalented Shred cost = %v, want %v", got, want)
	}

	talented, talentedSim, _ := newFeralDruidSimWithTalents(t, 60, feralTalentsString(t, map[string]int{"shredding_attacks": 3}))
	talented.PoisedAura.Deactivate(talentedSim)
	if got, want := talented.Shred.Cost.GetCurrentCost(), 60.0-18.0; got != want {
		t.Errorf("3/3 Shredding Attacks Shred cost = %v, want %v", got, want)
	}
}

// TestNaturalInstinctAddsMeleeCritDamageBonus covers: "Increases the
// critical strike damage bonus of your melee abilities by 10/20%." (node
// 104950, proto field NaturalInstinct; the healing-from-Intellect clause
// is a stat dependency this feral test does not read).
func TestNaturalInstinctAddsMeleeCritDamageBonus(t *testing.T) {
	base, _, _ := newFeralDruidSimWithTalents(t, 60, feralTalentsString(t, nil))
	baseBonus := base.Shred.CritDamageBonus

	talented, _, _ := newFeralDruidSimWithTalents(t, 60, feralTalentsString(t, map[string]int{"natural_instinct": 2}))
	if got, want := talented.Shred.CritDamageBonus, baseBonus+0.20; got != want {
		t.Errorf("2/2 Natural Instinct Shred.CritDamageBonus = %v, want %v", got, want)
	}
	// Ferocious Bite (a finisher, not a Combo-Point generator) is a
	// "melee ability" too.
	if got, want := talented.FerociousBite.CritDamageBonus, baseBonus+0.20; got != want {
		t.Errorf("2/2 Natural Instinct FerociousBite.CritDamageBonus = %v, want %v", got, want)
	}
}

// Natural Instinct (1223242) effect 0 is aura 108, misc 15 (crit damage),
// with SpellClassMask_0 0x839800 in 1.60.1.70707; its 0x800000 is Rip's
// (SpellClassOptions 9896). Rip's ticks crit (owner ruling: bleeds always
// crit), so they take the bonus.
func TestNaturalInstinctRaisesRipCrits(t *testing.T) {
	base, _, _ := newFeralDruidSimWithTalents(t, 60, feralTalentsString(t, nil))
	talented, _, _ := newFeralDruidSimWithTalents(t, 60, feralTalentsString(t, map[string]int{"natural_instinct": 2}))
	if got, want := talented.Rip.CritDamageBonus-base.Rip.CritDamageBonus, 0.20; math.Abs(got-want) > 1e-9 {
		t.Errorf("2/2 Natural Instinct raises Rip's crit damage bonus by %v, want %v", got, want)
	}
}
