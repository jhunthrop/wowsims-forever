package elemental

import (
	"math"
	"testing"
)

// The guide's 30/3/18 build with the Restoration points moved: the live
// tree's Tidal Focus (16179) is +5 spell hit at five ranks, the client's
// aura 55 on every spell. guideBuildWithTidalFocus keeps 18 Restoration
// points (Improved Healing Wave 5, Mindfulness 3, Natural Grace 3, Tidal
// Focus 5, Improved Reincarnation 2).
const (
	guideBuildWithTidalFocus    = "553031130010305-03-500352"
	guideBuildWithoutTidalFocus = "553031130010305-03-500302"
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
