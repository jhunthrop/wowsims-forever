package core

import (
	"math"

	"github.com/wowsims/classic/sim/core/stats"
)

// Level-aware values for the three class buffs wowsims models as raid
// buffs rather than castable spells (Arcane Intellect, Blessing of Might,
// Mark of the Wild). Each takes the highest rank the character's level
// can learn. Below level 60 a rank's value is the client's rank amount
// (SpellEffect, data/builds/1.60.1.70009/raw) scaled so the top rank
// equals the BuffSpellValues entry exactly: the level-60 value is the
// engine's established one, lower ranks keep the client's progression.

// buffRank is one learnable rank: the level it is learned at and its
// client effect amount.
type buffRank struct {
	level  int
	amount float64
}

// buffRanks lists ranks in ascending level order.
type buffRanks []buffRank

// fraction is the amount of the highest rank learnable at level as a
// share of the top rank's amount; 0 before the first rank is learned.
func (ranks buffRanks) fraction(level int) float64 {
	current := 0.0
	for _, rank := range ranks {
		if rank.level > level {
			break
		}
		current = rank.amount
	}
	return current / ranks[len(ranks)-1].amount
}

// scaleAmount scales a level-60 value by the rank fraction, whole points.
func (ranks buffRanks) scaleAmount(topValue float64, level int) float64 {
	return math.Round(topValue * ranks.fraction(level))
}

// Client rank tables, spells 1459..10157 (Arcane Intellect), 19740..25291
// (Blessing of Might), 1126..9885 (Mark of the Wild); Mark of the Wild's
// stat and resistance effects only exist from the ranks shown.
var (
	arcaneIntellectRanks = buffRanks{{1, 2}, {14, 7}, {28, 15}, {42, 22}, {56, 31}}
	blessingOfMightRanks = buffRanks{{4, 14}, {12, 25}, {22, 40}, {32, 61}, {42, 83}, {52, 112}, {60, 133}}

	markOfTheWildArmorRanks  = buffRanks{{1, 34}, {10, 88}, {20, 142}, {30, 203}, {40, 263}, {50, 324}, {60, 385}}
	markOfTheWildStatRanks   = buffRanks{{10, 3}, {20, 5}, {30, 8}, {40, 11}, {50, 14}, {60, 16}}
	markOfTheWildResistRanks = buffRanks{{30, 7}, {40, 14}, {50, 20}, {60, 27}}
)

var markOfTheWildAttributes = []stats.Stat{
	stats.Stamina, stats.Agility, stats.Strength, stats.Intellect, stats.Spirit,
}

var markOfTheWildResistances = []stats.Stat{
	stats.ArcaneResistance, stats.ShadowResistance, stats.NatureResistance,
	stats.FireResistance, stats.FrostResistance,
}

// ArcaneIntellectStats is the Arcane Intellect bonus at a character level.
func ArcaneIntellectStats(level int) stats.Stats {
	top := BuffSpellValues[ArcaneIntellect]
	return stats.Stats{stats.Intellect: arcaneIntellectRanks.scaleAmount(top[stats.Intellect], level)}
}

// BlessingOfMightAttackPower is the Blessing of Might attack power at a
// character level, before Improved Blessing of Might.
func BlessingOfMightAttackPower(level int) float64 {
	return blessingOfMightRanks.scaleAmount(BuffSpellValues[BlessingOfMight][stats.AttackPower], level)
}

// MarkOfTheWildStats is the Mark of the Wild bonus at a character level,
// before Improved Mark of the Wild.
func MarkOfTheWildStats(level int) stats.Stats {
	top := BuffSpellValues[MarkOfTheWild]
	result := stats.Stats{}
	result[stats.BonusArmor] = markOfTheWildArmorRanks.scaleAmount(top[stats.BonusArmor], level)
	for _, stat := range markOfTheWildAttributes {
		result[stat] = markOfTheWildStatRanks.scaleAmount(top[stat], level)
	}
	for _, stat := range markOfTheWildResistances {
		result[stat] = markOfTheWildResistRanks.scaleAmount(top[stat], level)
	}
	return result
}
