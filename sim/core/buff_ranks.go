package core

import (
	"math"

	"github.com/wowsims/classic/sim/core/proto"
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

	// Blessing of Wisdom: mana per five seconds (aura 24, a periodic
	// energize every five seconds). Rank 6 (25290, level 60) is the
	// Ahn'Qiraj book rank.
	BlessingOfWisdomRanks = BuffRanks{
		{SpellID: 19742, Level: 14, Amount: 12},
		{SpellID: 19850, Level: 24, Amount: 18},
		{SpellID: 19852, Level: 34, Amount: 24},
		{SpellID: 19853, Level: 44, Amount: 30},
		{SpellID: 19854, Level: 54, Amount: 36},
		{SpellID: 25290, Level: 60, Amount: 40, AhnQiraj: true},
	}

	// Trueshot Aura: the Marksmanship talent's aura, ranked by trainer.
	// Client 1.60.1.70707 cut ranks 1-4 to 15/22/30/40 (from 30/40/50/75);
	// the server's 70753 hotfix push (2026-11-03) raised all five to
	// 20/35/50/75/100, so rank 5 (20906, level 60) stays the strongest.
	TrueshotAuraRanks = BuffRanks{
		{SpellID: 1299346, Level: 25, Amount: 20},
		{SpellID: 1299348, Level: 32, Amount: 35},
		{SpellID: 19506, Level: 40, Amount: 50},
		{SpellID: 20905, Level: 50, Amount: 75},
		{SpellID: 20906, Level: 60, Amount: 100},
	}

	// Heart of the Lion's area buff (409583): melee and ranged attack power
	// are 40 base points plus 4 per caster level above the spell's
	// SpellLevels BaseLevel 0 (its SpellLevel is 1), capped at MaxLevel 60:
	// 280 there, as Wowhead Forever's buff tooltip reads. The hunter's own
	// spell (409580) carries the stat effect only.
	HeartOfTheLionRanks = BuffRanks{
		{SpellID: 409583, Level: 0, Amount: 40, PerLevel: 4, MaxLevel: 60},
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

	// Devotion Aura: armor to the paladin and its party (aura 22 until build 70291, 674 after; misc 1).
	DevotionAuraRanks = BuffRanks{
		{SpellID: 465, Level: 1, Amount: 55},
		{SpellID: 10290, Level: 10, Amount: 160},
		{SpellID: 643, Level: 20, Amount: 275},
		{SpellID: 10291, Level: 30, Amount: 390},
		{SpellID: 1032, Level: 40, Amount: 505},
		{SpellID: 10292, Level: 50, Amount: 620},
		{SpellID: 10293, Level: 60, Amount: 735},
	}

	// Strength of Earth Totem (8075 to 25361; aura spells 8076, 8162,
	// 8163, 10441, 25362, aura 29 Strength). Forever cut the Era 61/77 to
	// 42/53; rank 5 is the Ahn'Qiraj book.
	StrengthOfEarthRanks = BuffRanks{
		{SpellID: 8075, Level: 10, Amount: 7},
		{SpellID: 8160, Level: 24, Amount: 14},
		{SpellID: 8161, Level: 38, Amount: 25},
		{SpellID: 10442, Level: 52, Amount: 42},
		{SpellID: 25361, Level: 60, Amount: 53, AhnQiraj: true},
	}

	// Grace of Air Totem (8835, 10627, 25359; aura spells 8836, 10626,
	// 25360, aura 29 Agility). Forever raised Era's 43/67/77 to 49/77/89;
	// rank 3 is the Ahn'Qiraj book.
	GraceOfAirRanks = BuffRanks{
		{SpellID: 8835, Level: 42, Amount: 49},
		{SpellID: 10627, Level: 56, Amount: 77},
		{SpellID: 25359, Level: 60, Amount: 89, AhnQiraj: true},
	}

	// Mana Spring Totem (5675, 10495, 10496, 10497; aura spells 5677,
	// 10491, 10493, 10494, aura 24 every 2 s): the mana each pulse
	// restores. No Ahn'Qiraj rank exists.
	ManaSpringRanks = BuffRanks{
		{SpellID: 5675, Level: 26, Amount: 4},
		{SpellID: 10495, Level: 36, Amount: 6},
		{SpellID: 10496, Level: 46, Amount: 8},
		{SpellID: 10497, Level: 56, Amount: 10},
	}

	// Power Word: Fortitude (1243, 1244, 1245, 2791, 10937, 10938; aura
	// 29 Stamina): Forever folded Improved Power Word: Fortitude's 30%
	// into the ranks (Era's 54 is 70). Prayer of Fortitude (21562, 21564)
	// carries ranks 5 and 6's amounts. No Ahn'Qiraj rank.
	PowerWordFortitudeRanks = BuffRanks{
		{SpellID: 1243, Level: 1, Amount: 4},
		{SpellID: 1244, Level: 12, Amount: 10},
		{SpellID: 1245, Level: 24, Amount: 26},
		{SpellID: 2791, Level: 36, Amount: 42},
		{SpellID: 10937, Level: 48, Amount: 56},
		{SpellID: 10938, Level: 60, Amount: 70},
	}

	// Divine Spirit (14752, 14818, 14819, 27841; aura 29 Spirit): a
	// trainer spell from level 30 in Forever. Prayer of Spirit (27681)
	// carries rank 4's amount.
	DivineSpiritRanks = BuffRanks{
		{SpellID: 14752, Level: 30, Amount: 17},
		{SpellID: 14818, Level: 40, Amount: 23},
		{SpellID: 14819, Level: 50, Amount: 33},
		{SpellID: 27841, Level: 60, Amount: 40},
	}

	// Blood Pact, the Imp's party Stamina (6307, 7804, 7805, 11766,
	// 11767; aura 29 Stamina with points per level to SpellLevels
	// MaxLevel): rank 5 is 54 at 60.
	BloodPactRanks = BuffRanks{
		{SpellID: 6307, Level: 4, Amount: 3, PerLevel: 0.1, MaxLevel: 14},
		{SpellID: 7804, Level: 14, Amount: 9, PerLevel: 0.3, MaxLevel: 24},
		{SpellID: 7805, Level: 26, Amount: 21, PerLevel: 0.4, MaxLevel: 36},
		{SpellID: 11766, Level: 38, Amount: 35, PerLevel: 0.4, MaxLevel: 48},
		{SpellID: 11767, Level: 50, Amount: 49, PerLevel: 0.5, MaxLevel: 60},
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

// BlessingOfWisdomMP5 is the Blessing of Wisdom mana per five seconds at a
// character level, before any improved form.
func BlessingOfWisdomMP5(level int) float64 {
	return BlessingOfWisdomRanks.At(level)
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

// Talent auras a raid member can only bring from the level its talent tier
// opens (10 + 5 x tier in Forever's trees): Leader of the Pack (17007) is
// Feral tier 4, Moonkin Form (24858, whose Moonkin Aura is 24907) Balance
// tier 6.
const (
	LeaderOfThePackLevel = 30
	MoonkinAuraLevel     = 40
)

// LeaderOfThePackReaches says whether a raid of this level can carry Leader
// of the Pack.
func LeaderOfThePackReaches(level int32) bool { return level >= LeaderOfThePackLevel }

// MoonkinAuraReaches says whether a raid of this level can carry Moonkin
// Aura.
func MoonkinAuraReaches(level int32) bool { return level >= MoonkinAuraLevel }

// RaidHasDruidCritAura says whether the raid buffs give a character of this
// level a druid's 3% crit aura. Moonkin Aura (24907) and Leader of the Pack
// (24932) are both aura 290 at 3% and, by Blizzard's Forever notes of 3
// November 2026, "again exclusive" with each other: a raid with both gives
// 3% once.
func RaidHasDruidCritAura(raidBuffs *proto.RaidBuffs, level int32) bool {
	return (raidBuffs.MoonkinAura && MoonkinAuraReaches(level)) ||
		(raidBuffs.LeaderOfThePack && LeaderOfThePackReaches(level))
}

// manaSpringPulsesPerFive is Mana Spring's 2 s pulse read as mp5.
const manaSpringPulsesPerFive = 2.5

// ManaSpringMP5 is the raid's Mana Spring at a character level: the rank a
// shaman of that level has, its 2 s pulse as mana per 5 s.
func ManaSpringMP5(level int) float64 {
	return ManaSpringRanks.At(level) * manaSpringPulsesPerFive
}

// The reactive raid buffs (round 9). Thorns and Retribution Aura deal their
// amount to each melee attacker (aura 15); the Stoneskin aura a Stoneskin
// Totem rank places reduces Physical damage taken by its amount (aura 14,
// misc 1). Level is the rank's SpellLevels SpellLevel, which is the totem's
// for Stoneskin; SpellID is the aura's.
var (
	ThornsRanks = BuffRanks{
		{SpellID: 467, Level: 6, Amount: 3},
		{SpellID: 782, Level: 14, Amount: 6},
		{SpellID: 1075, Level: 24, Amount: 9},
		{SpellID: 8914, Level: 34, Amount: 12},
		{SpellID: 9756, Level: 44, Amount: 15},
		{SpellID: 9910, Level: 54, Amount: 18},
	}

	RetributionAuraRanks = BuffRanks{
		{SpellID: 7294, Level: 16, Amount: 5},
		{SpellID: 10298, Level: 26, Amount: 8},
		{SpellID: 10299, Level: 36, Amount: 12},
		{SpellID: 10300, Level: 46, Amount: 16},
		{SpellID: 10301, Level: 56, Amount: 20},
	}

	StoneskinRanks = BuffRanks{
		{SpellID: 8072, Level: 4, Amount: 4},
		{SpellID: 8156, Level: 14, Amount: 7},
		{SpellID: 8157, Level: 24, Amount: 11},
		{SpellID: 10403, Level: 34, Amount: 16},
		{SpellID: 10404, Level: 44, Amount: 22},
		{SpellID: 10405, Level: 54, Amount: 30},
	}
)

// A party's Mana Tide Totem (round 9). The talent (16190) is Restoration
// tier 3, so a raid has it from level 25; each totem rank (16190, 17354,
// 17359, 1325256 at 1, 38, 48 and 58) places a tick spell that restores its
// amount every 3 s (aura 24). SpellID is the tick's.
const ManaTideTotemTalentLevel = 25

var ManaTideTickRanks = BuffRanks{
	{SpellID: 16191, Level: 1, Amount: 88},
	{SpellID: 17355, Level: 38, Amount: 133},
	{SpellID: 17360, Level: 48, Amount: 197},
	{SpellID: 1325257, Level: 58, Amount: 290},
}

// PartyManaTidePerTick is the mana a party member of this level gets from
// each tick of a party shaman's Mana Tide Totem; 0 before the talent opens.
func PartyManaTidePerTick(level int32) float64 {
	if level < ManaTideTotemTalentLevel {
		return 0
	}
	return ManaTideTickRanks.At(int(level))
}
