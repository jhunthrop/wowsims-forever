package core

import (
	"math"

	"github.com/wowsims/classic/sim/core/stats"
)

// Level-aware values for the class buffs wowsims models as raid buffs
// rather than castable spells (Arcane Intellect, Blessing of Might, Mark
// of the Wild) and for Battle Shout. Each takes the highest rank the
// character's level can learn, and each rank states the client's amount
// (SpellEffect base points and points-per-level, SpellLevels, in
// data/builds/<build>/raw): Forever rebalanced these buffs, so the engine's
// old vanilla AQ-era fixed values are gone. The site's
// sim/leveling/buff_ranks_client_test.go pins every row below against the
// client CSVs, so a new build moves that test instead of silently moving a
// number here.

// BuffRank is one learnable rank of a buff spell.
type BuffRank struct {
	SpellID int32
	// Level is the level the rank is learned at.
	Level int
	// Amount is the client's base points for the effect.
	Amount float64
	// PerLevel is added to Amount for every caster level above Level, up to
	// MaxLevel (0 means the client states no cap).
	PerLevel float64
	MaxLevel int
	// AhnQiraj marks a book rank a launch character cannot learn: it is
	// only reachable while IncludeAQ is set.
	AhnQiraj bool
	// Inferior marks a rank the client makes weaker than the rank before it
	// (data/curated/inferior-ranks.json): a player keeps the stronger rank,
	// so it is never the learned rank.
	Inferior bool
}

// At is the effect amount for a caster of the given level, whole points.
func (rank BuffRank) At(level int) float64 {
	if rank.MaxLevel > 0 && level > rank.MaxLevel {
		level = rank.MaxLevel
	}
	above := max(level-rank.Level, 0)
	return math.Floor(rank.Amount + rank.PerLevel*float64(above))
}

// BuffRanks lists ranks in ascending level order.
type BuffRanks []BuffRank

// Learned is the highest rank learnable at level; false before the first.
// Ahn'Qiraj book ranks count only while IncludeAQ is set.
func (ranks BuffRanks) Learned(level int) (BuffRank, bool) {
	return ranks.learned(level, IncludeAQ)
}

func (ranks BuffRanks) learned(level int, includeAQ bool) (BuffRank, bool) {
	var learned BuffRank
	found := false
	for _, rank := range ranks {
		if rank.Level > level {
			break
		}
		if rank.AhnQiraj && !includeAQ {
			continue
		}
		if rank.Inferior {
			continue
		}
		learned, found = rank, true
	}
	return learned, found
}

// At is the amount of the highest rank learnable at level; 0 before the
// first rank is learned.
func (ranks BuffRanks) At(level int) float64 {
	return ranks.at(level, IncludeAQ)
}

func (ranks BuffRanks) at(level int, includeAQ bool) float64 {
	rank, ok := ranks.learned(level, includeAQ)
	if !ok {
		return 0
	}
	return rank.At(level)
}

// Client rank tables. Mark of the Wild's stat and resistance effects exist
// only from the ranks that state them (earlier ranks carry Amount 0).
var (
	ArcaneIntellectRanks = BuffRanks{
		{SpellID: 1459, Level: 1, Amount: 2},
		{SpellID: 1460, Level: 14, Amount: 7},
		{SpellID: 1461, Level: 28, Amount: 15},
		{SpellID: 10156, Level: 42, Amount: 22},
		{SpellID: 10157, Level: 56, Amount: 31},
	}

	BlessingOfMightRanks = BuffRanks{
		{SpellID: 19740, Level: 4, Amount: 14},
		{SpellID: 19834, Level: 12, Amount: 25},
		{SpellID: 19835, Level: 22, Amount: 40},
		{SpellID: 19836, Level: 32, Amount: 61},
		{SpellID: 19837, Level: 42, Amount: 83},
		{SpellID: 19838, Level: 52, Amount: 112},
		{SpellID: 25291, Level: 60, Amount: 133, AhnQiraj: true},
	}

	// Trueshot Aura: the Marksmanship talent's aura, ranked by trainer.
	// Rank 5 (20906, level 60) is 50 ranged attack power against rank 4's
	// 75 in the client, so a level-60 hunter keeps rank 4.
	TrueshotAuraRanks = BuffRanks{
		{SpellID: 1299346, Level: 25, Amount: 30},
		{SpellID: 1299348, Level: 32, Amount: 40},
		{SpellID: 19506, Level: 40, Amount: 50},
		{SpellID: 20905, Level: 50, Amount: 75},
		{SpellID: 20906, Level: 60, Amount: 50, Inferior: true},
	}

	MarkOfTheWildArmorRanks = BuffRanks{
		{SpellID: 1126, Level: 1, Amount: 34},
		{SpellID: 5232, Level: 10, Amount: 88},
		{SpellID: 6756, Level: 20, Amount: 142},
		{SpellID: 5234, Level: 30, Amount: 203},
		{SpellID: 8907, Level: 40, Amount: 263},
		{SpellID: 9884, Level: 50, Amount: 324},
		{SpellID: 9885, Level: 60, Amount: 385},
	}
	MarkOfTheWildStatRanks = BuffRanks{
		{SpellID: 1126, Level: 1, Amount: 0},
		{SpellID: 5232, Level: 10, Amount: 3},
		{SpellID: 6756, Level: 20, Amount: 5},
		{SpellID: 5234, Level: 30, Amount: 8},
		{SpellID: 8907, Level: 40, Amount: 11},
		{SpellID: 9884, Level: 50, Amount: 14},
		{SpellID: 9885, Level: 60, Amount: 16},
	}
	MarkOfTheWildResistRanks = BuffRanks{
		{SpellID: 1126, Level: 1, Amount: 0},
		{SpellID: 5232, Level: 10, Amount: 0},
		{SpellID: 6756, Level: 20, Amount: 0},
		{SpellID: 5234, Level: 30, Amount: 7},
		{SpellID: 8907, Level: 40, Amount: 14},
		{SpellID: 9884, Level: 50, Amount: 20},
		{SpellID: 9885, Level: 60, Amount: 27},
	}

	// Devotion Aura: armor to the paladin and its party (aura 22, misc 1).
	DevotionAuraRanks = BuffRanks{
		{SpellID: 465, Level: 1, Amount: 55},
		{SpellID: 10290, Level: 10, Amount: 160},
		{SpellID: 643, Level: 20, Amount: 275},
		{SpellID: 10291, Level: 30, Amount: 390},
		{SpellID: 1032, Level: 40, Amount: 505},
		{SpellID: 10292, Level: 50, Amount: 620},
		{SpellID: 10293, Level: 60, Amount: 735},
	}

	BattleShoutRankTable = BuffRanks{
		{SpellID: 6673, Level: 1, Amount: 9, PerLevel: 0.3, MaxLevel: 11},
		{SpellID: 5242, Level: 12, Amount: 21, PerLevel: 0.3, MaxLevel: 21},
		{SpellID: 6192, Level: 22, Amount: 33, PerLevel: 0.3, MaxLevel: 31},
		{SpellID: 11549, Level: 32, Amount: 51, PerLevel: 0.6, MaxLevel: 41},
		{SpellID: 11550, Level: 42, Amount: 78, PerLevel: 0.6, MaxLevel: 51},
		{SpellID: 11551, Level: 52, Amount: 111, PerLevel: 0.6, MaxLevel: 61},
		{SpellID: 25289, Level: 60, Amount: 139, PerLevel: 0.6, MaxLevel: 61, AhnQiraj: true},
	}
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
	return stats.Stats{stats.Intellect: ArcaneIntellectRanks.At(level)}
}

// BlessingOfMightAttackPower is the Blessing of Might attack power at a
// character level, before Improved Blessing of Might.
func BlessingOfMightAttackPower(level int) float64 {
	return BlessingOfMightRanks.At(level)
}

// BattleShoutAttackPower is the Battle Shout attack power at a character
// level, before Improved Battle Shout.
func BattleShoutAttackPower(level int) float64 {
	return BattleShoutRankTable.At(level)
}

// DevotionAuraArmor is the Devotion Aura armor at a character level, before
// any improved form.
func DevotionAuraArmor(level int) float64 {
	return DevotionAuraRanks.At(level)
}

// MarkOfTheWildStats is the Mark of the Wild bonus at a character level,
// before Improved Mark of the Wild.
func MarkOfTheWildStats(level int) stats.Stats {
	result := stats.Stats{}
	result[stats.BonusArmor] = MarkOfTheWildArmorRanks.At(level)
	for _, stat := range markOfTheWildAttributes {
		result[stat] = MarkOfTheWildStatRanks.At(level)
	}
	for _, stat := range markOfTheWildResistances {
		result[stat] = MarkOfTheWildResistRanks.At(level)
	}
	return result
}
