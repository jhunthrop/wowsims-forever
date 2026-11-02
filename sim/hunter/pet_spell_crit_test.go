package hunter

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// Round 10 (physical lane): Hunter Pet Scaling 415429 carries a spell-crit
// channel (effect 14, aura 57) beside its melee and ranged one (effect 13,
// aura 52). The pet inherited only the latter, so its magic abilities
// (Lightning Breath 24844 to 25012, DefenseType 1) rolled the every-table
// crit alone. Paired per hunter in the foreverlogs.gg beta logs (levels 20
// to 35, eight hunters), a pet's magic abilities crit 11.7% (224 of 1,920:
// Lightning Breath and Flametongue Attack) against its melee's 11.0% (1,342
// of 12,163); the engine gave the magic ones the every-table part only.
func TestHunterPetInheritsTheHuntersSpellCrit(t *testing.T) {
	hunter := newLevel60HunterWithFullBuffs(t)
	inherit := hunter.makeStatInheritance()
	owner := stats.Stats{stats.RangedAttackPower: 2000, stats.SpellCrit: 6 * core.CritRatingPerCritChance}
	hunter.petAttackPowerSource = higherAttackPower(owner)
	if got := inherit(owner)[stats.SpellCrit]; got != 6*core.CritRatingPerCritChance {
		t.Errorf("pet inherits %.2f spell crit, want all of the hunter's %.2f", got, float64(6*core.CritRatingPerCritChance))
	}
}

// Web (1265843 to 1265883) is DefenseType 3 in SpellCategories: it rolls the
// ranged table (melee and ranged crit, ranged hit), not the spell table its
// Nature school would otherwise pick.
func TestWebRollsTheRangedTable(t *testing.T) {
	_, built, _ := newFamilyPetHunter(t, proto.Hunter_Options_Spider)
	spell := built.pet.familyAbility
	if spell == nil {
		t.Fatal("the spider has no family ability")
	}
	if spell.DefenseType != core.DefenseTypeRanged {
		t.Errorf("Web DefenseType = %v, want ranged (client DefenseType 3)", spell.DefenseType)
	}
}

// The ranged outcome must work from a pet: the spider casts Web and its
// ticks land.
func TestSpiderCastsWebOnTheRangedTable(t *testing.T) {
	sim, built, target := newFamilyPetHunter(t, proto.Hunter_Options_Spider)
	pet := built.pet
	pet.Enable(sim, pet)
	pet.CurrentTarget = target
	web := pet.familyAbility
	for i := 0; i < 20000 && web.SpellMetrics[target.UnitIndex].Ticks+web.SpellMetrics[target.UnitIndex].CritTicks == 0; i++ {
		if sim.Step() {
			break
		}
	}
	if web.SpellMetrics[target.UnitIndex].Casts == 0 {
		t.Fatal("the spider never cast Web")
	}
	if web.SpellMetrics[target.UnitIndex].Ticks+web.SpellMetrics[target.UnitIndex].CritTicks == 0 {
		t.Error("Web was cast but never ticked")
	}
}
