package feral

import (
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

	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassDruid,
			Race:               proto.Race_RaceTauren,
			Level:              level,
			Equipment:          &proto.EquipmentSpec{},
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
	base, _, _ := newFeralDruidSimWithTalents(t, 60, feralTalentsString(t, nil))
	if base.Shred == nil {
		t.Fatal("level-60 Feral druid has no Shred registered")
	}
	if got, want := base.Shred.Cost.GetCurrentCost(), 60.0; got != want {
		t.Fatalf("untalented Shred cost = %v, want %v", got, want)
	}

	talented, _, _ := newFeralDruidSimWithTalents(t, 60, feralTalentsString(t, map[string]int{"shredding_attacks": 3}))
	if got, want := talented.Shred.Cost.GetCurrentCost(), 60.0-18.0; got != want {
		t.Errorf("3/3 Shredding Attacks Shred cost = %v, want %v", got, want)
	}
}

// TestPredatoryInstinctsAddsMeleeCritDamageBonus covers: "Increases the
// critical strike damage bonus of your melee abilities by 10/20%." (node
// 104950, proto field PredatoryInstincts).
func TestPredatoryInstinctsAddsMeleeCritDamageBonus(t *testing.T) {
	base, _, _ := newFeralDruidSimWithTalents(t, 60, feralTalentsString(t, nil))
	baseBonus := base.Shred.CritDamageBonus

	talented, _, _ := newFeralDruidSimWithTalents(t, 60, feralTalentsString(t, map[string]int{"predatory_instincts": 2}))
	if got, want := talented.Shred.CritDamageBonus, baseBonus+0.20; got != want {
		t.Errorf("2/2 Predatory Instincts Shred.CritDamageBonus = %v, want %v", got, want)
	}
	// Ferocious Bite (a finisher, not a Combo-Point generator) is a
	// "melee ability" too.
	if got, want := talented.FerociousBite.CritDamageBonus, baseBonus+0.20; got != want {
		t.Errorf("2/2 Predatory Instincts FerociousBite.CritDamageBonus = %v, want %v", got, want)
	}
}

// TestBerserkGrantsComboPointBuilderCritWhileActive covers: "... increases
// the critical strike chance of your Combo Point-generating abilities by
// 100% ... Lasts 15 sec." (node 104956, proto field Berserk). The Primal
// Bite retarget/cooldown-reset half is not modeled (talents.go's Mangle
// note: Bear Form's own damage kit is not modeled in this package).
func TestBerserkGrantsComboPointBuilderCritWhileActive(t *testing.T) {
	built, sim, target := newFeralDruidSimWithTalents(t, 60, feralTalentsString(t, map[string]int{"berserk": 1}))
	if built.Berserk == nil {
		t.Fatal("1/1 Berserk druid has no Berserk spell registered")
	}

	baseCrit := built.Shred.BonusCritRating
	wantBonus := 100.0 * core.CritRatingPerCritChance

	built.Berserk.Cast(sim, target)
	if got, want := built.Shred.BonusCritRating, baseCrit+wantBonus; got != want {
		t.Errorf("Shred.BonusCritRating during Berserk = %v, want %v", got, want)
	}
	if got, want := built.Rake.BonusCritRating, baseCrit+wantBonus; got != want {
		t.Errorf("Rake.BonusCritRating during Berserk = %v, want %v", got, want)
	}
	// Ferocious Bite is a finisher, not a Combo-Point generator: Berserk
	// must not touch it.
	if got, want := built.FerociousBite.BonusCritRating, baseCrit; got != want {
		t.Errorf("FerociousBite.BonusCritRating during Berserk = %v, want %v (Berserk must not affect it)", got, want)
	}
}
