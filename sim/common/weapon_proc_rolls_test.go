package common

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
)

// The 1.60.1.70707 client's proc for each weapon: the spell ItemXItemEffect
// names with trigger 2 (chance on hit), that spell's SpellMisc school, and
// its SpellEffect Effect 2 row (EffectBasePointsF and Variance). Forever
// retuned many of these from Era's numbers and gave several new spells.
func TestWeaponProcsMatchTheClient(t *testing.T) {
	type want struct {
		spellID  int32
		school   core.SpellSchool
		amount   float64
		variance float64
	}
	for item, w := range map[int32]want{
		AlcorsSunrazor:           {18833, core.SpellSchoolFire, 105, 0.2},
		BaronCharrsSceptre:       {13442, core.SpellSchoolFire, 90, 0},
		BlackhandDoomsaw:         {16549, core.SpellSchoolPhysical, 448, 0.33333},
		Bloodfist:                {16433, core.SpellSchoolPhysical, 35, 0},
		BloodletterScalpel:       {13486, core.SpellSchoolPhysical, 77, 0.15385},
		BonechillHammer:          {18276, core.SpellSchoolFrost, 105, 0},
		ColdrageDagger:           {1293790, core.SpellSchoolFrost, 42, 0.4},
		Darrowspike:              {18276, core.SpellSchoolFrost, 105, 0},
		Deathbringer:             {1305398, core.SpellSchoolShadow, 140, 0.5},
		DrakeTalonCleaver:        {21140, core.SpellSchoolPhysical, 240, 0},
		EbonHand:                 {18211, core.SpellSchoolShadow, 231, 0.6087},
		GryphonRidersStormhammer: {18081, core.SpellSchoolNature, 91, 0.31481},
		GutRipper:                {18107, core.SpellSchoolPhysical, 105, 0.24074},
		HammerOfTheNorthernWind:  {13439, core.SpellSchoolFrost, 50, 0.4},
		JekliksCrusher:           {24257, core.SpellSchoolPhysical, 210, 0.09524},
		JoonhosMercy:             {20883, core.SpellSchoolArcane, 84, 0},
		LinkensSwordOfMastery:    {18089, core.SpellSchoolNature, 98, 0.5},
		Nightblade:               {18211, core.SpellSchoolShadow, 231, 0.6087},
		PendulumOfDoom:           {10373, core.SpellSchoolPhysical, 311, 0.33333},
		PerditionsBlade:          {23267, core.SpellSchoolFire, 48, 0.33333},
		SatyrsLash:               {1292155, core.SpellSchoolShadow, 70, 0.42857},
		SearingNeedle:            {16454, core.SpellSchoolFire, 56, 0},
		TheNeedler:               {16405, core.SpellSchoolPhysical, 75, 0},
		VilerendSlicer:           {1320808, core.SpellSchoolPhysical, 225, 0},
		ViskagTheBloodletter:     {1305394, core.SpellSchoolPhysical, 236, 0.33333},
		ZulianSlicer:             {24251, core.SpellSchoolNature, 84, 0.28571},
		ClawOfTheShadowmancer:    {16409, core.SpellSchoolShadow, 35, 0},
		Thornspike:               {1293183, core.SpellSchoolPhysical, 45, 0},
	} {
		got, ok := clientWeaponProcs[item]
		if !ok {
			t.Errorf("item %d has no client weapon proc", item)
			continue
		}
		if got.spellID != w.spellID || got.school != w.school {
			t.Errorf("%s: spell %d school %v, want %d %v", got.name, got.spellID, got.school, w.spellID, w.school)
		}
		if math.Abs(got.roll.Amount-w.amount) > 1e-6 || math.Abs(got.roll.Variance-w.variance) > 1e-4 {
			t.Errorf("%s: roll %v ± %v, want %v ± %v", got.name, got.roll.Amount, got.roll.Variance, w.amount, w.variance)
		}
	}
}

// Each proc spell's SpellCategories.DefenseType in the 1.60.1.70707 client
// (DifficultyID 0), which decides the table the proc rolls: a magic one
// rolls spell hit and spell crit, a melee or ranged one melee crit for
// double damage. Thornspike's 1293183 is ranged, so it cannot be dodged or
// parried; every other proc here is magic or melee as its school suggests.
func TestWeaponProcsRollTheClientsDefenseType(t *testing.T) {
	client := map[int32]core.DefenseType{
		10373:   core.DefenseTypeMelee,
		13439:   core.DefenseTypeMagic,
		13442:   core.DefenseTypeMagic,
		13486:   core.DefenseTypeMelee,
		16405:   core.DefenseTypeMelee,
		16409:   core.DefenseTypeMagic,
		16433:   core.DefenseTypeMelee,
		16454:   core.DefenseTypeMagic,
		16549:   core.DefenseTypeMelee,
		18081:   core.DefenseTypeMagic,
		18089:   core.DefenseTypeMagic,
		18107:   core.DefenseTypeMelee,
		18211:   core.DefenseTypeMagic,
		18276:   core.DefenseTypeMagic,
		18833:   core.DefenseTypeMagic,
		20883:   core.DefenseTypeMagic,
		21140:   core.DefenseTypeMelee,
		23267:   core.DefenseTypeMagic,
		24251:   core.DefenseTypeMelee,
		24257:   core.DefenseTypeMelee,
		1292155: core.DefenseTypeMagic,
		1293183: core.DefenseTypeRanged,
		1293407: core.DefenseTypeMagic,
		1293790: core.DefenseTypeMagic,
		1305394: core.DefenseTypeMelee,
		1305398: core.DefenseTypeMagic,
		1320808: core.DefenseTypeMelee,
		1322218: core.DefenseTypeMagic,
	}
	for _, p := range clientWeaponProcs {
		want, ok := client[p.spellID]
		if !ok {
			t.Errorf("%s: proc spell %d has no client defense type in this test", p.name, p.spellID)
			continue
		}
		if p.defense != want {
			t.Errorf("%s: proc spell %d rolls defense type %v, the client's is %v", p.name, p.spellID, p.defense, want)
		}
	}
}
