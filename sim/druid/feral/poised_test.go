package feral

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/druid"
)

// Poised (Forever 1.60.1.70753; SkillLineAbility 59889, every druid, level
// 32): "While at maximum Energy, and out of combat, reduce the Energy cost
// of your next ability by 10." The buff it applies is 1328827 (aura 107,
// SpellMod 14 cost, -10; one charge, no duration). A fight starts out of
// combat at full Energy, so the opener is the ability it reaches.
func TestPoisedTakesTenEnergyOffTheCatsFirstAbility(t *testing.T) {
	cat, sim, target := newFeralDruidSimWithTalents(t, 60, feralTalentsString(t, nil))

	if cat.PoisedAura == nil || !cat.PoisedAura.IsActive() {
		t.Fatal("a level 60 cat starts the fight without Poised")
	}
	if got, want := cat.PoisedAura.ActionID.SpellID, int32(1328827); got != want {
		t.Errorf("Poised buff is spell %d, want the client's %d", got, want)
	}
	full := cat.Shred.Cost.BaseCost
	if got, want := cat.Shred.Cost.GetCurrentCost(), full-10; got != want {
		t.Errorf("Shred costs %v under Poised, want %v", got, want)
	}

	cat.Shred.Cast(sim, target)

	if cat.PoisedAura.IsActive() {
		t.Error("Poised is still up after the first ability")
	}
	if got := cat.Shred.Cost.GetCurrentCost(); got != full {
		t.Errorf("the second Shred costs %v, want the full %v", got, full)
	}
}

// The druid trains Poised at 32 (SpellLevels 154257 / SkillLineAbility 59889).
func TestCatPoisedIsNotKnownBelowThirtyTwo(t *testing.T) {
	low, _, _ := newFeralDruidSimWithTalents(t, 31, feralTalentsString(t, nil))
	if low.PoisedAura != nil {
		t.Error("a level 31 druid has Poised")
	}
	at, _, _ := newFeralDruidSimWithTalents(t, 32, feralTalentsString(t, nil))
	if at.PoisedAura == nil || !at.PoisedAura.IsActive() {
		t.Error("a level 32 druid starts the fight without Poised")
	}
}

// The buff's EffectSpellClassMask (SpellEffect 1368040: 8622080, 536870912,
// 0, 0) names Claw, Rake, Shred, Ravage, Pounce, Rip and Ferocious Bite (and
// Cower, which the engine does not model): every Cat Form ability that
// costs Energy, and nothing that costs rage or mana (Maul, Swipe, Lacerate,
// Primal Bite, Shifting Power).
func TestCatPoisedReachesEveryEnergyAbility(t *testing.T) {
	cat, _, _ := newFeralDruidSimWithTalents(t, 60, feralTalentsString(t, map[string]int{"shifting_power": 1}))
	mask := druid.PoisedEngineMask()

	reached := 0
	for _, spell := range cat.Spellbook {
		costsEnergy := spell.Cost != nil && spell.Cost.CostType() == core.CostTypeEnergy && spell.Cost.BaseCost > 0
		switch {
		case costsEnergy && !spell.Matches(mask):
			t.Errorf("%v costs Energy but Poised does not reach it", spell.ActionID)
		case !costsEnergy && spell.Matches(mask):
			t.Errorf("%v costs no Energy but is in Poised's mask", spell.ActionID)
		case costsEnergy:
			reached++
		}
	}
	if reached < 7 {
		t.Errorf("Poised reaches %d Energy abilities, want Claw, Rake, Shred, Ravage, Pounce, Rip and Ferocious Bite", reached)
	}
	for _, name := range []struct {
		mask  uint64
		label string
	}{{druid.DruidSpellMaskMaul, "Maul"}, {druid.DruidSpellMaskSwipe, "Swipe"}, {druid.DruidSpellMaskShiftingPower, "Shifting Power"}} {
		if mask&name.mask != 0 {
			t.Errorf("Poised reaches %s, which costs no Energy", name.label)
		}
	}
}
