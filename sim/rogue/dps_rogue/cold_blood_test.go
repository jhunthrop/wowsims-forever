package dpsrogue

import "testing"

// Cold Blood's client text names Mutilate: "...your next Sinister Strike,
// Backstab, Ambush, Eviscerate, or Mutilate by 100%." Mutilate's damage is
// dealt by its two hand sub-spells, so Cold Blood must reach both hits and
// then be spent (not fade after the main-hand hit and leave the off hand
// without it, and not linger past the cast).
func TestColdBloodMakesBothMutilateHitsCritAndIsConsumed(t *testing.T) {
	sim, built := buildRogueForTest(t, rogueTalentStringWith(t, "mutilate", "cold_blood"))
	target := sim.Encounter.TargetUnits[0]
	coldBlood := built.GetAura("Cold Blood")
	if coldBlood == nil {
		t.Fatal("talented rogue has no Cold Blood aura")
	}

	landedHands := 0
	for i := 0; i < 60; i++ {
		built.ColdBlood.ApplyEffects(sim, target, built.ColdBlood)
		mhBefore, ohBefore := built.MutilateMH.SpellMetrics[target.UnitIndex], built.MutilateOH.SpellMetrics[target.UnitIndex]
		built.Mutilate.ApplyEffects(sim, target, built.Mutilate)
		mh, oh := built.MutilateMH.SpellMetrics[target.UnitIndex], built.MutilateOH.SpellMetrics[target.UnitIndex]

		if mh.Hits != mhBefore.Hits || oh.Hits != ohBefore.Hits {
			t.Fatalf("cast %d: a Cold Blood Mutilate landed a non-crit hit (MH hits %d->%d, OH hits %d->%d)", i, mhBefore.Hits, mh.Hits, ohBefore.Hits, oh.Hits)
		}
		landedHands += int(mh.Crits-mhBefore.Crits) + int(oh.Crits-ohBefore.Crits)
		if coldBlood.IsActive() {
			t.Fatalf("cast %d: Cold Blood is still up after Mutilate", i)
		}
	}
	if landedHands == 0 {
		t.Fatal("no Mutilate hand ever landed in 60 casts")
	}
}
