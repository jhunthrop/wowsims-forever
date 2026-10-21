package druid

import (
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
)

// The ranks below are spellconst/druid.json's (the client build
// 1.60.1.70009), keyed by the ids the class trainer teaches: the ones with a
// SkillLineAbility learn row. The client carries a second, unlearnable id
// for every Regrowth and Rejuvenation rank (436937-436946 and 417057-417068,
// the Season of Discovery reissues of the vanilla numbers, which no
// SkillLineAbility row teaches) and the generated tables in
// constants_auto_gen.go keep those because they have the higher id; they
// are not the spells a Forever druid can learn, so nothing here reads them.
// Tranquility's rank-0 rows (1253568, 21791, 25817) are the NPC copies.
//
// Each rank states the base roll of its heal and of its periodic effect (per
// tick) with the level scaling the client gives it, and the spell-power
// share (coefficient) of each effect. sim/druid/healing_client_test.go checks
// every row against the vendored client file.

// fx abbreviates the client roll type the tables are written in.
type fx = clientdamage.Effect

// healingTouchTable is Healing Touch ranks 1-11; rank 11 is the Ahn'Qiraj book rank.
var healingTouchTable = []healingRank{
	{
		spellID:  5185,
		manaCost: 25,
		castTime: 1500 * time.Millisecond,
		heal:     healingEffect{fx{Amount: 44, Variance: 0.318182, PerLevel: 0.8, SpellLevel: 1, MaxLevel: 5}, 0.429},
	},
	{
		spellID:  5186,
		manaCost: 55,
		castTime: 2000 * time.Millisecond,
		heal:     healingEffect{fx{Amount: 100, Variance: 0.24, PerLevel: 1.3, SpellLevel: 8, MaxLevel: 13}, 0.571},
	},
	{
		spellID:  5187,
		manaCost: 110,
		castTime: 2500 * time.Millisecond,
		heal:     healingEffect{fx{Amount: 201, Variance: 0.219178, PerLevel: 1.9, SpellLevel: 14, MaxLevel: 19}, 0.714},
	},
	{
		spellID:  5188,
		manaCost: 190,
		castTime: 3000 * time.Millisecond,
		heal:     healingEffect{fx{Amount: 362, Variance: 0.20297, PerLevel: 2.7, SpellLevel: 20, MaxLevel: 25}, 0.857},
	},
	{
		spellID:  5189,
		manaCost: 280,
		castTime: 3500 * time.Millisecond,
		heal:     healingEffect{fx{Amount: 546, Variance: 0.192733, PerLevel: 3.5, SpellLevel: 26, MaxLevel: 31}, 1},
	},
	{
		spellID:  6778,
		manaCost: 350,
		castTime: 3500 * time.Millisecond,
		heal:     healingEffect{fx{Amount: 707, Variance: 0.185819, PerLevel: 4, SpellLevel: 32, MaxLevel: 37}, 1},
	},
	{
		spellID:  8903,
		manaCost: 425,
		castTime: 3500 * time.Millisecond,
		heal:     healingEffect{fx{Amount: 898, Variance: 0.178988, PerLevel: 4.5, SpellLevel: 38, MaxLevel: 43}, 1},
	},
	{
		spellID:  9758,
		manaCost: 520,
		castTime: 3500 * time.Millisecond,
		heal:     healingEffect{fx{Amount: 1173, Variance: 0.173648, PerLevel: 5.2, SpellLevel: 44, MaxLevel: 49}, 1},
	},
	{
		spellID:  9888,
		manaCost: 630,
		castTime: 3500 * time.Millisecond,
		heal:     healingEffect{fx{Amount: 1516, Variance: 0.169082, PerLevel: 5.9, SpellLevel: 50, MaxLevel: 55}, 1},
	},
	{
		spellID:  9889,
		manaCost: 755,
		castTime: 3500 * time.Millisecond,
		heal:     healingEffect{fx{Amount: 1920, Variance: 0.165049, PerLevel: 6.6, SpellLevel: 56, MaxLevel: 61}, 1},
	},
	{
		spellID:  25297,
		manaCost: 840,
		castTime: 3500 * time.Millisecond,
		heal:     healingEffect{fx{Amount: 2332, Variance: 0.165858, PerLevel: 7.3, SpellLevel: 60, MaxLevel: 65}, 1},
	},
}

// regrowthTable is Regrowth ranks 1-9: a heal and a periodic heal per tick.
var regrowthTable = []healingRank{
	{
		spellID:  8936,
		manaCost: 70,
		castTime: 2000 * time.Millisecond,
		heal:     healingEffect{fx{Amount: 87, Variance: 0.153846, PerLevel: 1.8, SpellLevel: 12, MaxLevel: 17}, 0.286},
		tick:     healingEffect{fx{Amount: 13, SpellLevel: 12, MaxLevel: 17}, 0.071},
	},
	{
		spellID:  8938,
		manaCost: 125,
		castTime: 2000 * time.Millisecond,
		heal:     healingEffect{fx{Amount: 164, Variance: 0.136364, PerLevel: 2.5, SpellLevel: 18, MaxLevel: 23}, 0.286},
		tick:     healingEffect{fx{Amount: 22, SpellLevel: 18, MaxLevel: 23}, 0.071},
	},
	{
		spellID:  8939,
		manaCost: 170,
		castTime: 2000 * time.Millisecond,
		heal:     healingEffect{fx{Amount: 235, Variance: 0.132296, PerLevel: 3.1, SpellLevel: 24, MaxLevel: 29}, 0.286},
		tick:     healingEffect{fx{Amount: 32, SpellLevel: 24, MaxLevel: 29}, 0.071},
	},
	{
		spellID:  8940,
		manaCost: 210,
		castTime: 2000 * time.Millisecond,
		heal:     healingEffect{fx{Amount: 311, Variance: 0.123894, PerLevel: 3.6, SpellLevel: 30, MaxLevel: 35}, 0.286},
		tick:     healingEffect{fx{Amount: 42, SpellLevel: 30, MaxLevel: 35}, 0.071},
	},
	{
		spellID:  8941,
		manaCost: 250,
		castTime: 2000 * time.Millisecond,
		heal:     healingEffect{fx{Amount: 394, Variance: 0.12065, PerLevel: 4.1, SpellLevel: 36, MaxLevel: 41}, 0.286},
		tick:     healingEffect{fx{Amount: 52, SpellLevel: 36, MaxLevel: 41}, 0.071},
	},
	{
		spellID:  9750,
		manaCost: 305,
		castTime: 2000 * time.Millisecond,
		heal:     healingEffect{fx{Amount: 504, Variance: 0.117864, PerLevel: 4.7, SpellLevel: 42, MaxLevel: 47}, 0.286},
		tick:     healingEffect{fx{Amount: 68, SpellLevel: 42, MaxLevel: 47}, 0.071},
	},
	{
		spellID:  9856,
		manaCost: 370,
		castTime: 2000 * time.Millisecond,
		heal:     healingEffect{fx{Amount: 645, Variance: 0.113869, PerLevel: 5.3, SpellLevel: 48, MaxLevel: 53}, 0.286},
		tick:     healingEffect{fx{Amount: 88, SpellLevel: 48, MaxLevel: 53}, 0.071},
	},
	{
		spellID:  9857,
		manaCost: 445,
		castTime: 2000 * time.Millisecond,
		heal:     healingEffect{fx{Amount: 817, Variance: 0.112019, PerLevel: 6, SpellLevel: 54, MaxLevel: 59}, 0.286},
		tick:     healingEffect{fx{Amount: 113, SpellLevel: 54, MaxLevel: 59}, 0.071},
	},
	{
		spellID:  9858,
		manaCost: 525,
		castTime: 2000 * time.Millisecond,
		heal:     healingEffect{fx{Amount: 1021, Variance: 0.109331, PerLevel: 6.8, SpellLevel: 60, MaxLevel: 65}, 0.286},
		tick:     healingEffect{fx{Amount: 142, SpellLevel: 60, MaxLevel: 65}, 0.071},
	},
}

// rejuvenationTable is Rejuvenation ranks 1-11; rank 11 is the Ahn'Qiraj book rank.
var rejuvenationTable = []healingRank{
	{
		spellID:  774,
		manaCost: 25,
		tick:     healingEffect{fx{Amount: 8, SpellLevel: 4, MaxLevel: 9}, 0.2},
	},
	{
		spellID:  1058,
		manaCost: 40,
		tick:     healingEffect{fx{Amount: 12, SpellLevel: 10, MaxLevel: 15}, 0.2},
	},
	{
		spellID:  1430,
		manaCost: 75,
		tick:     healingEffect{fx{Amount: 23, SpellLevel: 16, MaxLevel: 21}, 0.2},
	},
	{
		spellID:  2090,
		manaCost: 105,
		tick:     healingEffect{fx{Amount: 32, SpellLevel: 22, MaxLevel: 27}, 0.2},
	},
	{
		spellID:  2091,
		manaCost: 135,
		tick:     healingEffect{fx{Amount: 42, SpellLevel: 28, MaxLevel: 33}, 0.2},
	},
	{
		spellID:  3627,
		manaCost: 160,
		tick:     healingEffect{fx{Amount: 51, SpellLevel: 34, MaxLevel: 39}, 0.2},
	},
	{
		spellID:  8910,
		manaCost: 195,
		tick:     healingEffect{fx{Amount: 71, SpellLevel: 40, MaxLevel: 45}, 0.2},
	},
	{
		spellID:  9839,
		manaCost: 235,
		tick:     healingEffect{fx{Amount: 94, SpellLevel: 46, MaxLevel: 51}, 0.2},
	},
	{
		spellID:  9840,
		manaCost: 280,
		tick:     healingEffect{fx{Amount: 124, SpellLevel: 52, MaxLevel: 57}, 0.2},
	},
	{
		spellID:  9841,
		manaCost: 335,
		tick:     healingEffect{fx{Amount: 161, SpellLevel: 58, MaxLevel: 63}, 0.2},
	},
	{
		spellID:  25299,
		manaCost: 360,
		tick:     healingEffect{fx{Amount: 194, SpellLevel: 60, MaxLevel: 65}, 0.2},
	},
}

// tranquilityTable is Tranquility ranks 1-4; the tick is each party member's.
var tranquilityTable = []healingRank{
	{
		spellID:  740,
		manaCost: 375,
		tick:     healingEffect{fx{Amount: 87, PerLevel: 0.6, SpellLevel: 30, MaxLevel: 36}, 0.067},
	},
	{
		spellID:  8918,
		manaCost: 505,
		tick:     healingEffect{fx{Amount: 129, PerLevel: 0.7, SpellLevel: 40, MaxLevel: 46}, 0.067},
	},
	{
		spellID:  9862,
		manaCost: 695,
		tick:     healingEffect{fx{Amount: 196, PerLevel: 0.9, SpellLevel: 50, MaxLevel: 56}, 0.067},
	},
	{
		spellID:  9863,
		manaCost: 925,
		tick:     healingEffect{fx{Amount: 285, PerLevel: 1, SpellLevel: 60, MaxLevel: 66}, 0.067},
	},
}

// wildGrowthTable is Wild Growth ranks 1-3; the tick is each target's average (see wildGrowthTickWeights).
var wildGrowthTable = []healingRank{
	{
		spellID:  408120,
		manaCost: 550,
		tick:     healingEffect{fx{Amount: 40, PerLevel: 1, SpellLevel: 40, MaxLevel: 48}, 0.033},
	},
	{
		spellID:  1238214,
		manaCost: 755,
		tick:     healingEffect{fx{Amount: 60, PerLevel: 1.3, SpellLevel: 50, MaxLevel: 58}, 0.033},
	},
	{
		spellID:  1238215,
		manaCost: 1050,
		tick:     healingEffect{fx{Amount: 97, PerLevel: 1.6, SpellLevel: 60, MaxLevel: 68}, 0.033},
	},
}
