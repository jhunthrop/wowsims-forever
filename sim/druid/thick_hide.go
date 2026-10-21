package druid

import "github.com/wowsims/classic/sim/core/stats"

// Thick Hide (node 104942, build 1.60.1.70009), rank 3 text: "While in
// Bear Form, Cat Form, Dire Bear Form, or Moonkin Form, you gain 3
// additional base Armor per level and another 2.00 base Armor for each
// point of defense skill beyond five times your level. This amount can be
// further increased by multipliers from those forms." Ranks 1 and 2 give 1
// and 2 per level and 0.67 and 1.33 per defense point, which is a third of
// the top rank's 2.00 a rank.
//
// stats.Defense is the skill beyond five times the level (a character has
// no base Defense stat, and the miss, dodge and parry tables key on that
// excess), so it is read as it stands when the form is entered.
//
// Only the forms this package models armor for are covered: Cat and Bear.
// Moonkin Form's own armor multiplier is not modelled either, so Thick Hide
// is left out of it rather than half applied.
const (
	thickHideMaxRank                = 3
	thickHideArmorPerLevelPerRank   = 1.0
	thickHideArmorPerDefensePerRank = 2.0 / 3.0
	catFormArmorMultiplier          = 1.0
)

// thickHideArmor is the base armor Thick Hide adds in a form that
// multiplies the armor items carry by armorMultiplier.
func (druid *Druid) thickHideArmor(armorMultiplier float64) stats.Stats {
	rank := float64(clampRank(druid.Talents.ThickHide, thickHideMaxRank))
	if rank == 0 {
		return stats.Stats{}
	}
	perLevel := thickHideArmorPerLevelPerRank * rank * float64(druid.Level)
	perDefense := thickHideArmorPerDefensePerRank * rank * max(druid.GetStat(stats.Defense), 0)
	return stats.Stats{stats.Armor: (perLevel + perDefense) * armorMultiplier}
}
