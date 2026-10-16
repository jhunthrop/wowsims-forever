package elemental

import (
	"math"
	"testing"
)

// The guide's 30/3/18 build (1.60.1.70425 layout) with Tidal Focus added
// to its Restoration points: the live tree's Tidal Focus (16179) is +5
// spell hit at five ranks, the client's aura 55 on every spell.
// guideBuildWithTidalFocus is the leveling build (Improved Healing Wave 5,
// Mindfulness 3, Natural Grace 3, Totemic Focus 5, Improved Reincarnation
// 2) plus Tidal Focus 5.
const (
	guideBuildWithTidalFocus    = "5510331300001350-030000000000000000-5533520000000000"
	guideBuildWithoutTidalFocus = "5510331300001350-030000000000000000-5033520000000000"
	tidalFocusSpellHit          = 0.05
	hitTolerance                = 1e-9
)

func TestTidalFocusAddsFiveSpellHitToLightningBolt(t *testing.T) {
	simWithout, without := newIsolatedTalentShaman(t, guideBuildWithoutTidalFocus)
	simWith, with := newIsolatedTalentShaman(t, guideBuildWithTidalFocus)

	boltWithout := without.LightningBolt[len(without.LightningBolt)-1]
	boltWith := with.LightningBolt[len(with.LightningBolt)-1]

	hitWithout := boltWithout.SpellHitChance(simWithout.Encounter.TargetUnits[0])
	hitWith := boltWith.SpellHitChance(simWith.Encounter.TargetUnits[0])

	if got := hitWith - hitWithout; math.Abs(got-tidalFocusSpellHit) > hitTolerance {
		t.Errorf("Tidal Focus 5 moved Lightning Bolt's hit by %v, want %v", got, tidalFocusSpellHit)
	}
}

func TestTidalFocusLeavesDamageSpellCostsAlone(t *testing.T) {
	_, without := newIsolatedTalentShaman(t, guideBuildWithoutTidalFocus)
	_, with := newIsolatedTalentShaman(t, guideBuildWithTidalFocus)

	boltWithout := without.LightningBolt[len(without.LightningBolt)-1]
	boltWith := with.LightningBolt[len(with.LightningBolt)-1]

	if got, want := boltWith.Cost.GetCurrentCost(), boltWithout.Cost.GetCurrentCost(); got != want {
		t.Errorf("Tidal Focus changed Lightning Bolt's cost to %v, want %v (healing spells only)", got, want)
	}
}
