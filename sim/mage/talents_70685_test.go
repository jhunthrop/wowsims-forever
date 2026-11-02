package mage

import (
	"math"
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Client 1.60.1.70685 (unchanged in 70707) renamed Improved Fireball to
// Accelerant and added Arcane Wrath to its spell list.
func TestAccelerantShortensArcaneWrath(t *testing.T) {
	talents := talentStringWithRank(t, ForeverFrostTalents, "arcane_wrath", 1)
	talents = talentStringWithRank(t, talents, "accelerant", 5)
	_, mage := newMageAtLevel(t, 60, talents)
	spell := mage.ArcaneWrath[ArcaneWrathRanks]
	if spell == nil {
		t.Fatal("a level-60 mage with the Arcane Wrath point has no max-rank Arcane Wrath")
	}
	want := time.Duration(ArcaneWrathCastTime[ArcaneWrathRanks])*time.Millisecond - 500*time.Millisecond
	if got := spell.CastTime(); got != want {
		t.Errorf("Arcane Wrath cast time with 5/5 Accelerant = %v, want %v", got, want)
	}
}

// Precision is Elemental Precision widened to every school: "Improves
// your chance to hit with all spells by 2%." Arcane spells, which Arcane
// Focus (removed in the same build) used to cover, now take it too.
func TestPrecisionAddsHitToEverySchool(t *testing.T) {
	withPrecision := talentStringWithRank(t, ForeverFrostTalents, "arcane_wrath", 1)
	without := talentStringWithRank(t, withPrecision, "precision", 0)
	onSim, on := newMageAtLevel(t, 60, withPrecision)
	offSim, off := newMageAtLevel(t, 60, without)
	onTarget, offTarget := onSim.Encounter.TargetUnits[0], offSim.Encounter.TargetUnits[0]

	want := 3 * precisionHitPerRank / 100
	spells := map[string]func(m *Mage) *core.Spell{
		"Frostbolt":    func(m *Mage) *core.Spell { return m.Frostbolt[core.MaxTrainerRank(FrostboltRanks)] },
		"Fireball":     func(m *Mage) *core.Spell { return m.Fireball[core.MaxTrainerRank(FireballRanks)] },
		"Arcane Wrath": func(m *Mage) *core.Spell { return m.ArcaneWrath[ArcaneWrathRanks] },
	}
	for name, spell := range spells {
		if got := spell(on).SpellHitChance(onTarget) - spell(off).SpellHitChance(offTarget); math.Abs(got-want) > 1e-9 {
			t.Errorf("%s: 3/3 Precision adds %v hit chance, want %v", name, got, want)
		}
	}
}

// Critical Mass has three ranks since client 1.60.1.70685. A string
// written for the old five-rank tree must read as 3/3, not 5/5.
func TestCriticalMassClampsToThreeRanks(t *testing.T) {
	three := talentStringWithRank(t, ForeverFrostTalents, "critical_mass", 3)
	five := talentStringWithRank(t, ForeverFrostTalents, "critical_mass", 5)
	_, atThree := newMageAtLevel(t, 60, three)
	_, atFive := newMageAtLevel(t, 60, five)
	fireball := func(m *Mage) *core.Spell { return m.Fireball[core.MaxTrainerRank(FireballRanks)] }
	if got, want := fireball(atFive).BonusCritRating, fireball(atThree).BonusCritRating; got != want {
		t.Errorf("Fireball crit rating with Critical Mass written as 5 = %v, want the 3/3 value %v", got, want)
	}
}
