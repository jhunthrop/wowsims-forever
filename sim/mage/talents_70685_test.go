package mage

import (
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
	_, on := newMageAtLevel(t, 60, withPrecision)
	_, off := newMageAtLevel(t, 60, without)

	want := 3 * precisionHitPerRank * core.HitRatingPerHitChance
	spells := map[string]func(m *Mage) *core.Spell{
		"Frostbolt":    func(m *Mage) *core.Spell { return m.Frostbolt[core.MaxTrainerRank(FrostboltRanks)] },
		"Fireball":     func(m *Mage) *core.Spell { return m.Fireball[core.MaxTrainerRank(FireballRanks)] },
		"Arcane Wrath": func(m *Mage) *core.Spell { return m.ArcaneWrath[ArcaneWrathRanks] },
	}
	for name, spell := range spells {
		if got := spell(on).BonusHitRating - spell(off).BonusHitRating; got != want {
			t.Errorf("%s: 3/3 Precision adds %v hit rating, want %v", name, got, want)
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

// Ice Lance's rank 1 carries the client's 8 sec cooldown since 1.60.1.70685;
// the trained ranks a level-60 mage casts carry none.
func TestIceLanceCooldownFollowsTheClientRank(t *testing.T) {
	_, low := newMageAtLevel(t, 20, ForeverFrostTalents)
	if got := low.IceLance[1].CD.Duration; got != 8*time.Second {
		t.Errorf("rank 1 Ice Lance cooldown = %v, want 8s", got)
	}
	_, high := newMageAtLevel(t, 60, ForeverFrostTalents)
	if cd := high.IceLance[IceLanceRanks].CD; cd.Timer != nil {
		t.Errorf("max-rank Ice Lance has a cooldown of %v, want none", cd.Duration)
	}
}
