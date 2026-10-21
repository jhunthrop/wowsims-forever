package feral

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

const (
	catFormSpellID        = 768
	powershiftStartEnergy = 30.0
)

// powershiftEnergy returns the energy right after a powershift from
// startEnergy, with the given talents.
func powershiftEnergy(t *testing.T, talents map[string]int, startEnergy float64) float64 {
	t.Helper()
	cat, sim, _ := newFeralDruidSimWithTalents(t, 60, feralTalentsString(t, talents))
	metrics := cat.NewEnergyMetrics(core.ActionID{OtherID: proto.OtherAction_OtherActionPotion})
	cat.SpendEnergy(sim, cat.CurrentEnergy()-startEnergy, metrics)
	if got := cat.CurrentEnergy(); got != startEnergy {
		t.Fatalf("setup: energy = %v, want %v", got, startEnergy)
	}
	cat.tryPowershift(sim)
	if !cat.CatFormAura.IsActive() {
		t.Fatal("not in cat form after the powershift")
	}
	return cat.CurrentEnergy()
}

// TestFurorReachesThePowershift: the cat rotation's powershift (cancel the
// form, cast Cat Form) hands back the talent's share of the energy the druid
// had, per the live Furor text, not the Era flat 40.
func TestFurorReachesThePowershift(t *testing.T) {
	cases := []struct {
		rank int
		want float64
	}{{0, 0}, {1, 6}, {3, 18}, {5, 30}}
	for _, c := range cases {
		got := powershiftEnergy(t, map[string]int{"furor": c.rank}, powershiftStartEnergy)
		if got != c.want {
			t.Errorf("Furor %d: energy after a powershift from %v = %v, want %v", c.rank, powershiftStartEnergy, got, c.want)
		}
	}
}

// TestNaturalShapeshifterCutsTheShiftManaCost: "Reduces the mana cost of all
// shapeshifting by 10/20/30%." It must reach Cat Form (the powershift's
// cost) and Shifting Power.
func TestNaturalShapeshifterCutsTheShiftManaCost(t *testing.T) {
	base, _, _ := newFeralDruidSimWithTalents(t, 60, feralTalentsString(t, map[string]int{"shifting_power": 1}))
	talented, _, _ := newFeralDruidSimWithTalents(t, 60, feralTalentsString(t, map[string]int{"shifting_power": 1, "natural_shapeshifter": 3}))

	costs := map[string][2]float64{
		"Cat Form":       {base.CatForm.DefaultCast.Cost, talented.CatForm.DefaultCast.Cost},
		"Shifting Power": {base.ShiftingPower.DefaultCast.Cost, talented.ShiftingPower.DefaultCast.Cost},
	}
	for name, pair := range costs {
		untalented, got := pair[0], pair[1]
		if untalented <= 0 {
			t.Fatalf("%s: untalented cost = %v, want > 0", name, untalented)
		}
		if want := untalented * 0.7; got < want-0.01 || got > want+0.01 {
			t.Errorf("%s: 3/3 Natural Shapeshifter cost = %v, want 70%% of %v = %v", name, got, untalented, want)
		}
	}
}

// TestPowershiftingNeedsFuror: the hard-coded cat rotation only powershifts
// with Furor (rotation.go's maxShiftsPossible gate); with the talent it casts
// Cat Form again after leaving it.
func TestPowershiftingNeedsFuror(t *testing.T) {
	without := castsOfCatForm(t, 0)
	with := castsOfCatForm(t, 5)
	if without != 0 {
		t.Errorf("Cat Form casts without Furor = %d, want 0", without)
	}
	if with == 0 {
		t.Error("the hard-coded cat rotation never powershifted with 5/5 Furor")
	}
}

func castsOfCatForm(t *testing.T, furorRank int) int32 {
	t.Helper()
	talents := feralTalentsString(t, map[string]int{"furor": furorRank})
	player := core.WithSpec(&proto.Player{
		Class:              proto.Class_ClassDruid,
		Race:               proto.Race_RaceTauren,
		Level:              60,
		Equipment:          &proto.EquipmentSpec{},
		Buffs:              core.FullBuffs.Player,
		TalentsString:      talents,
		DistanceFromTarget: 5,
		Rotation:           &proto.APLRotation{PriorityList: []*proto.APLListItem{optimalCatRotation()}},
	}, PlayerOptionsMonoCat)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)
	result := core.RunRaidSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 120, Targets: []*proto.Target{core.NewDefaultTarget()}},
		SimOptions: &proto.SimOptions{Iterations: 1, RandomSeed: 1, IsTest: true},
	})
	if result.Error != nil {
		t.Fatalf("sim error: %s", result.Error.Message)
	}
	return castCountsBySpellID(result)[catFormSpellID]
}
