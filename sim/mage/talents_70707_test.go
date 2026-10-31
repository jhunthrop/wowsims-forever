package mage

import (
	"testing"
	"time"
)

// Client 1.60.1.70707 hotfixed four Frost and Arcane numbers in the talent
// text (the Wowhead overlay) without touching the shipped SpellEffect rows,
// which still read the 70685 values. These pin the text's numbers.

// Ice Barrier: "increasing your Frost damage by 15% for 1 min. While
// shielded, your Frost damage is increased by an additional 5%".
func TestIceBarrierCarriesThe70707Numbers(t *testing.T) {
	if iceBarrierFrostDamagePct != 15 || iceBarrierShieldedFrostDamagePct != 5 {
		t.Errorf("Ice Barrier Frost damage = %d%% + %d%% shielded, want 15%% + 5%%",
			iceBarrierFrostDamagePct, iceBarrierShieldedFrostDamagePct)
	}
}

// Winter's Chill: "increasing the target's damage taken from your next Cone
// of Cold by 20% and it gains an additional 7% of your spell damage,
// stacking up to 5 times".
func TestWintersChillCarriesThe70707Numbers(t *testing.T) {
	if wintersChillConeOfColdDamagePerStack != 0.20 {
		t.Errorf("Winter's Chill Cone of Cold damage taken a stack = %v, want 0.20", wintersChillConeOfColdDamagePerStack)
	}
	if wintersChillConeOfColdCoefficientPerStack != 0.07 {
		t.Errorf("Winter's Chill spell damage a stack = %v, want 0.07", wintersChillConeOfColdCoefficientPerStack)
	}
}

// Arcane Blast: "you gain 12% of your Intellect as Spell Damage and the mana
// cost of Arcane Blast is increased by 100%. Effect stacks up to 4 times and
// lasts 18 sec from time of initial application."
func TestArcaneBlastCarriesThe70707Numbers(t *testing.T) {
	if arcaneBlastIntellectShare != 0.12 {
		t.Errorf("Arcane Blast Intellect share a stack = %v, want 0.12", arcaneBlastIntellectShare)
	}
	if arcaneBlastCostIncreasePctPerStack != 100 {
		t.Errorf("Arcane Blast cost increase a stack = %d%%, want 100%%", arcaneBlastCostIncreasePctPerStack)
	}
	if arcaneBlastMaxStacks != 4 {
		t.Errorf("Arcane Blast stacks = %d, want 4", arcaneBlastMaxStacks)
	}
	if arcaneBlastDuration != 18*time.Second {
		t.Errorf("Arcane Blast buff duration = %v, want 18s", arcaneBlastDuration)
	}
}

// The buff's clock runs from the first cast: a cast at 17 s still adds a
// stack, and the buff is gone at 18 s.
func TestArcaneBlastBuffLastsEighteenSecondsFromTheFirstCast(t *testing.T) {
	sim, built := newArcaneBlastTestMage(t)
	arcaneBlast := built.ArcaneBlast[ArcaneBlastRanks]
	target := sim.Encounter.TargetUnits[0]
	arcaneBlast.ApplyEffects(sim, target, arcaneBlast)
	advanceBy(sim, 17*time.Second)
	if !built.ArcaneBlastAura.IsActive() {
		t.Fatal("the Arcane Blast buff dropped before 17 s")
	}
	arcaneBlast.ApplyEffects(sim, target, arcaneBlast)
	if got := built.ArcaneBlastAura.GetStacks(); got != 2 {
		t.Errorf("stacks after a second cast at 17 s = %d, want 2", got)
	}
	advanceBy(sim, 1500*time.Millisecond)
	if built.ArcaneBlastAura.IsActive() {
		t.Error("the Arcane Blast buff outlived 18 s from its first cast")
	}
}
