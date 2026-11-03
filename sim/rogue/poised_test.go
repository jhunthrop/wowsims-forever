package rogue_test

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/rogue"
)

// Poised (Forever 1.60.1.70753; SkillLineAbility 59888, every rogue, level
// 18): "While at maximum Energy, and out of combat, reduce the Energy cost
// of your next ability by 10." The buff it applies is 1328826 (aura 107,
// SpellMod 14 cost, -10; one charge). A fight starts out of combat at full
// Energy, so the opener is the ability it reaches.
func TestPoisedTakesTenEnergyOffTheFirstAbility(t *testing.T) {
	sim, built := buildRogueSimForTalentTest(t, zeroRogueTalents, "combat_backstab_prebis")
	r := built.GetRogue()
	target := sim.Encounter.TargetUnits[0]

	if r.PoisedAura == nil || !r.PoisedAura.IsActive() {
		t.Fatal("a level 60 rogue starts the fight without Poised")
	}
	if got, want := r.PoisedAura.ActionID.SpellID, int32(1328826); got != want {
		t.Errorf("Poised buff is spell %d, want the client's %d", got, want)
	}
	full := r.SinisterStrike.Cost.BaseCost
	if got, want := r.SinisterStrike.Cost.GetCurrentCost(), full-10; got != want {
		t.Errorf("Sinister Strike costs %v under Poised, want %v", got, want)
	}

	r.SinisterStrike.Cast(sim, target)

	if r.PoisedAura.IsActive() {
		t.Error("Poised is still up after the first ability")
	}
	if got := r.SinisterStrike.Cost.GetCurrentCost(); got != full {
		t.Errorf("the second Sinister Strike costs %v, want the full %v", got, full)
	}
}

// Poised is trained at 18 (SpellLevels / SkillLineAbility 59888).
func TestPoisedIsNotKnownBelowEighteen(t *testing.T) {
	_, built := buildRogueSimAtLevel(t, zeroRogueTalents, "blank", 17)
	if r := built.GetRogue(); r.PoisedAura != nil {
		t.Error("a level 17 rogue has Poised")
	}
	_, built = buildRogueSimAtLevel(t, zeroRogueTalents, "blank", 18)
	if r := built.GetRogue(); r.PoisedAura == nil || !r.PoisedAura.IsActive() {
		t.Error("a level 18 rogue starts the fight without Poised")
	}
}

// The buff's EffectSpellClassMask (255727518, 2099456, 67108864, 0) holds
// every rogue ability that costs Energy. Every Energy ability the engine
// registers must be one it reaches, and nothing without an Energy cost
// (poisons, Mutilate's hand strikes, Blade Flurry's copy) may spend it.
func TestPoisedReachesEveryEnergyAbility(t *testing.T) {
	talents := talentsWithRanks(t, map[string]int{
		"mutilate": 1, "hemorrhage": 1, "ghostly_strike": 1, "blade_flurry": 1, "riposte": 1, "venom": 1,
	})
	_, built := buildRogueSimForTalentTest(t, talents, "combat_backstab_prebis")
	r := built.GetRogue()
	mask := rogue.PoisedEngineMaskForTest()

	reached := 0
	for _, spell := range r.Spellbook {
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
	if reached < 15 {
		t.Errorf("Poised reaches %d Energy abilities, want every one the engine registers (at least 15)", reached)
	}
}
