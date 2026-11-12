package druid

import "github.com/wowsims/classic/sim/core/stats"

// Thick Hide (node 104942, client 1.60.1.70790), rank text: "While in Bear
// Form, Cat Form, Dire Bear Form, or Moonkin Form, you gain 180 additional
// base Armor per level and another 0.67 / 1.33 / 2.00 base Armor for each
// point of defense skill beyond five times your level. This amount can be
// further increased by multipliers from those forms."
//
// The armor-per-level half is the talent spell's (16929) effect 1: base
// points 1 / 2 / 3 by rank (curve 111990) plus 3 a level above level 1
// (EffectRealPointsPerLevel 3, the same at every rank), 180 at level 60 on
// every rank as the text renders it; "per level" is the template's wording
// for an amount that grows with the level, not a multiplier on it. The
// defense half is effect 0: 0.67 / 1.33 / 2.00 by rank (curve 85346, in
// hundredths). Build 1.60.1.70009 scaled the per-level half by rank, 1 / 2
// / 3 a level, which is what the top rank still reads.
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
	thickHideArmorBasePerRank       = 1.0
	thickHideArmorPerLevel          = 3.0
	thickHideArmorPerDefensePerRank = 2.0 / 3.0
	catFormArmorMultiplier          = 1.0
)

// thickHideBaseArmor is the base armor Thick Hide adds at the rank and
// level with defense skill beyond five times the level, before the form's
// multiplier.
func thickHideBaseArmor(rank int32, level int32, defense float64) float64 {
	if rank <= 0 {
		return 0
	}
	r := float64(clampRank(rank, thickHideMaxRank))
	perLevel := thickHideArmorBasePerRank*r + thickHideArmorPerLevel*float64(max(level, 1)-1)
	perDefense := thickHideArmorPerDefensePerRank * r * max(defense, 0)
	return perLevel + perDefense
}

// thickHideArmor is the base armor Thick Hide adds in a form that
// multiplies the armor items carry by armorMultiplier.
func (druid *Druid) thickHideArmor(armorMultiplier float64) stats.Stats {
	armor := thickHideBaseArmor(druid.Talents.ThickHide, druid.Level, druid.GetStat(stats.Defense))
	if armor == 0 {
		return stats.Stats{}
	}
	return stats.Stats{stats.Armor: armor * armorMultiplier}
}
