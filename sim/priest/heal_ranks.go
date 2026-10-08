package priest

import "github.com/wowsims/classic/sim/common/clientdamage"

// The healing ladders below are the client's rows for build 1.60.1.70009
// (sim/core/testdata/conformance/client/priest.json), one entry per rank in
// rank order, restricted to the ranks a level-60 priest can learn.
// constants_auto_gen.go cannot serve them: it keys healing spells by name
// and rank only, so Renew resolves to the free NPC variant 28807 and the
// unlearnable 4252xx copies, Power Word: Shield and Prayer of Healing carry
// a rank 0, and Flash Heal, Greater Heal, Renew and Power Word: Shield pick
// the higher-id duplicate that the trainer does not teach first. The ids
// here are the trainer's (data/builds/<build>/trainables/priest.json); where
// the trainer lists two spells for one rank (Flash Heal 10917 and 27608,
// Renew 10929 and 27606, Power Word: Shield 10901 and 27607) the first,
// vanilla id is kept, as Shadow Word: Pain keeps 10894 over 27605.
// TestHealingLaddersMatchTheClient pins every row, at several levels, to the
// client file, so a client patch is a failing test and then a regeneration.

// healRank is one rank of a priest healing spell.
type healRank struct {
	spellID int32
	// effectSpellID is the spell that carries the amount when the cast
	// spell does not (Penance's volley, Holy Nova's heal half); zero means
	// spellID.
	effectSpellID int32
	level         int
	manaCost      float64
	castMS        int32
	// coefficient is the effect's spell-power share (per tick for a HoT).
	coefficient float64
	// effect is the amount: what a direct heal restores, a HoT restores
	// per tick, a shield absorbs, or Prayer of Mending heals per jump.
	effect clientdamage.Effect
}

var lesserHealRanks = []healRank{
	{spellID: 2050, level: 1, manaCost: 30, castMS: 1500, coefficient: 0.429,
		effect: clientdamage.Effect{Amount: 51, Variance: 0.196078, PerLevel: 0.9, SpellLevel: 1, MaxLevel: 3}},
	{spellID: 2052, level: 4, manaCost: 45, castMS: 2000, coefficient: 0.571,
		effect: clientdamage.Effect{Amount: 78, Variance: 0.179487, PerLevel: 1.1, SpellLevel: 4, MaxLevel: 9}},
	{spellID: 2053, level: 10, manaCost: 75, castMS: 2500, coefficient: 0.714,
		effect: clientdamage.Effect{Amount: 141, Variance: 0.150685, PerLevel: 1.6, SpellLevel: 10, MaxLevel: 15}},
}

var healRanks = []healRank{
	{spellID: 2054, level: 16, manaCost: 155, castMS: 3000, coefficient: 0.857,
		effect: clientdamage.Effect{Amount: 291, Variance: 0.144654, PerLevel: 2.4, SpellLevel: 16, MaxLevel: 21}},
	{spellID: 2055, level: 22, manaCost: 205, castMS: 3000, coefficient: 0.857,
		effect: clientdamage.Effect{Amount: 405, Variance: 0.134783, PerLevel: 3.2, SpellLevel: 22, MaxLevel: 27}},
	{spellID: 6063, level: 28, manaCost: 255, castMS: 3000, coefficient: 0.857,
		effect: clientdamage.Effect{Amount: 523, Variance: 0.125828, PerLevel: 4, SpellLevel: 28, MaxLevel: 33}},
	{spellID: 6064, level: 34, manaCost: 305, castMS: 3000, coefficient: 0.857,
		effect: clientdamage.Effect{Amount: 651, Variance: 0.121372, PerLevel: 4.5, SpellLevel: 34, MaxLevel: 39}},
}

var flashHealRanks = []healRank{
	{spellID: 2061, level: 20, manaCost: 125, castMS: 1500, coefficient: 0.429,
		effect: clientdamage.Effect{Amount: 194, Variance: 0.204651, PerLevel: 1.9, SpellLevel: 20, MaxLevel: 25}},
	{spellID: 9472, level: 26, manaCost: 155, castMS: 1500, coefficient: 0.429,
		effect: clientdamage.Effect{Amount: 249, Variance: 0.195804, PerLevel: 2.2, SpellLevel: 26, MaxLevel: 31}},
	{spellID: 9473, level: 32, manaCost: 185, castMS: 1500, coefficient: 0.429,
		effect: clientdamage.Effect{Amount: 312, Variance: 0.183333, PerLevel: 2.5, SpellLevel: 32, MaxLevel: 37}},
	{spellID: 9474, level: 38, manaCost: 215, castMS: 1500, coefficient: 0.429,
		effect: clientdamage.Effect{Amount: 383, Variance: 0.177677, PerLevel: 2.8, SpellLevel: 38, MaxLevel: 43}},
	{spellID: 10915, level: 44, manaCost: 265, castMS: 1500, coefficient: 0.429,
		effect: clientdamage.Effect{Amount: 507, Variance: 0.17284, PerLevel: 3.3, SpellLevel: 44, MaxLevel: 49}},
	{spellID: 10916, level: 50, manaCost: 315, castMS: 1500, coefficient: 0.429,
		effect: clientdamage.Effect{Amount: 644, Variance: 0.170455, PerLevel: 3.7, SpellLevel: 50, MaxLevel: 55}},
	{spellID: 10917, level: 56, manaCost: 380, castMS: 1500, coefficient: 0.429,
		effect: clientdamage.Effect{Amount: 825, Variance: 0.164972, PerLevel: 4.2, SpellLevel: 56, MaxLevel: 61}},
}

var greaterHealRanks = []healRank{
	{spellID: 2060, level: 40, manaCost: 370, castMS: 3000, coefficient: 0.857,
		effect: clientdamage.Effect{Amount: 844, Variance: 0.119247, PerLevel: 5.1, SpellLevel: 40, MaxLevel: 45}},
	{spellID: 10963, level: 46, manaCost: 455, castMS: 3000, coefficient: 0.857,
		effect: clientdamage.Effect{Amount: 1099, Variance: 0.114848, PerLevel: 5.8, SpellLevel: 46, MaxLevel: 51}},
	{spellID: 10964, level: 52, manaCost: 545, castMS: 3000, coefficient: 0.857,
		effect: clientdamage.Effect{Amount: 1403, Variance: 0.112935, PerLevel: 6.6, SpellLevel: 52, MaxLevel: 57}},
	{spellID: 10965, level: 58, manaCost: 655, castMS: 3000, coefficient: 0.857,
		effect: clientdamage.Effect{Amount: 1782, Variance: 0.109359, PerLevel: 7.5, SpellLevel: 58, MaxLevel: 63}},
	{spellID: 25314, level: 60, manaCost: 710, castMS: 3000, coefficient: 0.857,
		effect: clientdamage.Effect{Amount: 1960, Variance: 0.109615, PerLevel: 8.1, SpellLevel: 60, MaxLevel: 65}},
}

var bindingHealRanks = []healRank{
	{spellID: 401937, level: 25, manaCost: 155, castMS: 1500, coefficient: 0.429,
		effect: clientdamage.Effect{Amount: 247, Variance: 0.195804, PerLevel: 2.2, SpellLevel: 25, MaxLevel: 31}},
	{spellID: 1240770, level: 32, manaCost: 185, castMS: 1500, coefficient: 0.429,
		effect: clientdamage.Effect{Amount: 312, Variance: 0.183333, PerLevel: 2.5, SpellLevel: 32, MaxLevel: 37}},
	{spellID: 1240771, level: 38, manaCost: 215, castMS: 1500, coefficient: 0.429,
		effect: clientdamage.Effect{Amount: 383, Variance: 0.177677, PerLevel: 2.8, SpellLevel: 38, MaxLevel: 43}},
	{spellID: 1240772, level: 44, manaCost: 265, castMS: 1500, coefficient: 0.429,
		effect: clientdamage.Effect{Amount: 507, Variance: 0.17284, PerLevel: 3.3, SpellLevel: 44, MaxLevel: 49}},
	{spellID: 1240773, level: 50, manaCost: 315, castMS: 1500, coefficient: 0.429,
		effect: clientdamage.Effect{Amount: 644, Variance: 0.170455, PerLevel: 3.7, SpellLevel: 50, MaxLevel: 55}},
	{spellID: 1240774, level: 56, manaCost: 380, castMS: 1500, coefficient: 0.429,
		effect: clientdamage.Effect{Amount: 825, Variance: 0.164972, PerLevel: 4.2, SpellLevel: 56, MaxLevel: 61}},
}

var desperatePrayerRanks = []healRank{
	{spellID: 13908, level: 10, manaCost: 0, castMS: 0, coefficient: 0.429,
		effect: clientdamage.Effect{Amount: 149, Variance: 0.236842, PerLevel: 2.4, SpellLevel: 10, MaxLevel: 16}},
	{spellID: 19236, level: 18, manaCost: 0, castMS: 0, coefficient: 0.429,
		effect: clientdamage.Effect{Amount: 276, Variance: 0.210884, PerLevel: 3.4, SpellLevel: 18, MaxLevel: 24}},
	{spellID: 19238, level: 26, manaCost: 0, castMS: 0, coefficient: 0.429,
		effect: clientdamage.Effect{Amount: 458, Variance: 0.193939, PerLevel: 4.5, SpellLevel: 26, MaxLevel: 32}},
	{spellID: 19240, level: 34, manaCost: 0, castMS: 0, coefficient: 0.429,
		effect: clientdamage.Effect{Amount: 595, Variance: 0.185185, PerLevel: 5.3, SpellLevel: 34, MaxLevel: 40}},
	{spellID: 19241, level: 42, manaCost: 0, castMS: 0, coefficient: 0.429,
		effect: clientdamage.Effect{Amount: 856, Variance: 0.175055, PerLevel: 6.4, SpellLevel: 42, MaxLevel: 48}},
	{spellID: 19242, level: 50, manaCost: 0, castMS: 0, coefficient: 0.429,
		effect: clientdamage.Effect{Amount: 1143, Variance: 0.169576, PerLevel: 7.4, SpellLevel: 50, MaxLevel: 56}},
	{spellID: 19243, level: 58, manaCost: 0, castMS: 0, coefficient: 0.429,
		effect: clientdamage.Effect{Amount: 1383, Variance: 0.164934, PerLevel: 8.2, SpellLevel: 58, MaxLevel: 64}},
}

var prayerOfHealingRanks = []healRank{
	{spellID: 596, level: 30, manaCost: 410, castMS: 3000, coefficient: 0.286,
		effect: clientdamage.Effect{Amount: 178, Variance: 0.064309, PerLevel: 0.8, SpellLevel: 30, MaxLevel: 39}},
	{spellID: 996, level: 40, manaCost: 560, castMS: 3000, coefficient: 0.286,
		effect: clientdamage.Effect{Amount: 265, Variance: 0.061135, PerLevel: 1, SpellLevel: 40, MaxLevel: 49}},
	{spellID: 10960, level: 50, manaCost: 770, castMS: 3000, coefficient: 0.286,
		effect: clientdamage.Effect{Amount: 401, Variance: 0.056213, PerLevel: 1.3, SpellLevel: 50, MaxLevel: 59}},
	{spellID: 10961, level: 60, manaCost: 1030, castMS: 3000, coefficient: 0.286,
		effect: clientdamage.Effect{Amount: 583, Variance: 0.053886, PerLevel: 1.5, SpellLevel: 60, MaxLevel: 69}},
	{spellID: 25316, level: 60, manaCost: 1070, castMS: 3000, coefficient: 0.286,
		effect: clientdamage.Effect{Amount: 649, Variance: 0.054206, PerLevel: 1.6, SpellLevel: 60, MaxLevel: 69}},
}

var renewRanks = []healRank{
	{spellID: 139, level: 8, manaCost: 30, castMS: 0, coefficient: 0.2,
		effect: clientdamage.Effect{Amount: 9, Variance: 0, PerLevel: 0, SpellLevel: 8, MaxLevel: 13}},
	{spellID: 6074, level: 14, manaCost: 65, castMS: 0, coefficient: 0.2,
		effect: clientdamage.Effect{Amount: 15, Variance: 0, PerLevel: 0, SpellLevel: 14, MaxLevel: 19}},
	{spellID: 6075, level: 20, manaCost: 105, castMS: 0, coefficient: 0.2,
		effect: clientdamage.Effect{Amount: 25, Variance: 0, PerLevel: 0, SpellLevel: 20, MaxLevel: 25}},
	{spellID: 6076, level: 26, manaCost: 140, castMS: 0, coefficient: 0.2,
		effect: clientdamage.Effect{Amount: 32, Variance: 0, PerLevel: 0, SpellLevel: 26, MaxLevel: 31}},
	{spellID: 6077, level: 32, manaCost: 170, castMS: 0, coefficient: 0.2,
		effect: clientdamage.Effect{Amount: 41, Variance: 0, PerLevel: 0, SpellLevel: 32, MaxLevel: 37}},
	{spellID: 6078, level: 38, manaCost: 205, castMS: 0, coefficient: 0.2,
		effect: clientdamage.Effect{Amount: 54, Variance: 0, PerLevel: 0, SpellLevel: 38, MaxLevel: 43}},
	{spellID: 10927, level: 44, manaCost: 250, castMS: 0, coefficient: 0.2,
		effect: clientdamage.Effect{Amount: 74, Variance: 0, PerLevel: 0, SpellLevel: 44, MaxLevel: 49}},
	{spellID: 10928, level: 50, manaCost: 305, castMS: 0, coefficient: 0.2,
		effect: clientdamage.Effect{Amount: 102, Variance: 0, PerLevel: 0, SpellLevel: 50, MaxLevel: 55}},
	{spellID: 10929, level: 56, manaCost: 365, castMS: 0, coefficient: 0.2,
		effect: clientdamage.Effect{Amount: 134, Variance: 0, PerLevel: 0, SpellLevel: 56, MaxLevel: 61}},
	{spellID: 25315, level: 60, manaCost: 410, castMS: 0, coefficient: 0.2,
		effect: clientdamage.Effect{Amount: 166, Variance: 0, PerLevel: 0, SpellLevel: 60, MaxLevel: 65}},
}

var powerWordShieldRanks = []healRank{
	{spellID: 17, level: 6, manaCost: 45, castMS: 0, coefficient: 0.1,
		effect: clientdamage.Effect{Amount: 44, Variance: 0, PerLevel: 0.8, SpellLevel: 6, MaxLevel: 11}},
	{spellID: 592, level: 12, manaCost: 80, castMS: 0, coefficient: 0.1,
		effect: clientdamage.Effect{Amount: 86, Variance: 0, PerLevel: 1.2, SpellLevel: 12, MaxLevel: 17}},
	{spellID: 600, level: 18, manaCost: 130, castMS: 0, coefficient: 0.1,
		effect: clientdamage.Effect{Amount: 154, Variance: 0, PerLevel: 1.6, SpellLevel: 18, MaxLevel: 23}},
	{spellID: 3747, level: 24, manaCost: 175, castMS: 0, coefficient: 0.1,
		effect: clientdamage.Effect{Amount: 226, Variance: 0, PerLevel: 2, SpellLevel: 24, MaxLevel: 29}},
	{spellID: 6065, level: 30, manaCost: 210, castMS: 0, coefficient: 0.1,
		effect: clientdamage.Effect{Amount: 291, Variance: 0, PerLevel: 2.3, SpellLevel: 30, MaxLevel: 35}},
	{spellID: 6066, level: 36, manaCost: 250, castMS: 0, coefficient: 0.1,
		effect: clientdamage.Effect{Amount: 368, Variance: 0, PerLevel: 2.6, SpellLevel: 36, MaxLevel: 41}},
	{spellID: 10898, level: 42, manaCost: 300, castMS: 0, coefficient: 0.1,
		effect: clientdamage.Effect{Amount: 470, Variance: 0, PerLevel: 3, SpellLevel: 42, MaxLevel: 47}},
	{spellID: 10899, level: 48, manaCost: 355, castMS: 0, coefficient: 0.1,
		effect: clientdamage.Effect{Amount: 591, Variance: 0, PerLevel: 3.4, SpellLevel: 48, MaxLevel: 53}},
	{spellID: 10900, level: 54, manaCost: 425, castMS: 0, coefficient: 0.1,
		effect: clientdamage.Effect{Amount: 749, Variance: 0, PerLevel: 3.9, SpellLevel: 54, MaxLevel: 59}},
	{spellID: 10901, level: 60, manaCost: 500, castMS: 0, coefficient: 0.1,
		effect: clientdamage.Effect{Amount: 928, Variance: 0, PerLevel: 4.3, SpellLevel: 60, MaxLevel: 65}},
}

var holyNovaDamageRanks = []healRank{
	{spellID: 15237, level: 20, manaCost: 185, castMS: 0, coefficient: 0.107,
		effect: clientdamage.Effect{Amount: 27, Variance: 0.133333, PerLevel: 0.2, SpellLevel: 20, MaxLevel: 26}},
	{spellID: 15430, level: 28, manaCost: 290, castMS: 0, coefficient: 0.107,
		effect: clientdamage.Effect{Amount: 49, Variance: 0.148148, PerLevel: 0.4, SpellLevel: 28, MaxLevel: 34}},
	{spellID: 15431, level: 36, manaCost: 400, castMS: 0, coefficient: 0.107,
		effect: clientdamage.Effect{Amount: 75, Variance: 0.146341, PerLevel: 0.6, SpellLevel: 36, MaxLevel: 42}},
	{spellID: 27799, level: 44, manaCost: 520, castMS: 0, coefficient: 0.107,
		effect: clientdamage.Effect{Amount: 106, Variance: 0.140351, PerLevel: 0.8, SpellLevel: 44, MaxLevel: 50}},
	{spellID: 27800, level: 52, manaCost: 635, castMS: 0, coefficient: 0.107,
		effect: clientdamage.Effect{Amount: 143, Variance: 0.145695, PerLevel: 1, SpellLevel: 52, MaxLevel: 58}},
	{spellID: 27801, level: 60, manaCost: 750, castMS: 0, coefficient: 0.107,
		effect: clientdamage.Effect{Amount: 187, Variance: 0.14359, PerLevel: 1.2, SpellLevel: 60, MaxLevel: 66}},
}

var holyNovaHealRanks = []healRank{
	{spellID: 23455, level: 20, manaCost: 0, castMS: 0, coefficient: 0.107,
		effect: clientdamage.Effect{Amount: 51, Variance: 0.142857, PerLevel: 0.4, SpellLevel: 20, MaxLevel: 26}},
	{spellID: 23458, level: 28, manaCost: 0, castMS: 0, coefficient: 0.107,
		effect: clientdamage.Effect{Amount: 82, Variance: 0.130435, PerLevel: 0.5, SpellLevel: 28, MaxLevel: 34}},
	{spellID: 23459, level: 36, manaCost: 0, castMS: 0, coefficient: 0.107,
		effect: clientdamage.Effect{Amount: 116, Variance: 0.138462, PerLevel: 0.6, SpellLevel: 36, MaxLevel: 42}},
	{spellID: 27803, level: 44, manaCost: 0, castMS: 0, coefficient: 0.107,
		effect: clientdamage.Effect{Amount: 159, Variance: 0.149425, PerLevel: 0.7, SpellLevel: 44, MaxLevel: 50}},
	{spellID: 27804, level: 52, manaCost: 0, castMS: 0, coefficient: 0.107,
		effect: clientdamage.Effect{Amount: 238, Variance: 0.142292, PerLevel: 0.8, SpellLevel: 52, MaxLevel: 58}},
	{spellID: 27805, level: 60, manaCost: 0, castMS: 0, coefficient: 0.107,
		effect: clientdamage.Effect{Amount: 311, Variance: 0.147239, PerLevel: 0.9, SpellLevel: 60, MaxLevel: 66}},
}

var penanceRanks = []healRank{
	{spellID: 402174, effectSpellID: 402289, level: 30, manaCost: 100, castMS: 0, coefficient: 0.285,
		effect: clientdamage.Effect{Amount: 184, Variance: 0, PerLevel: 0, SpellLevel: 30, MaxLevel: 39}},
	{spellID: 1240720, effectSpellID: 1240723, level: 40, manaCost: 185, castMS: 0, coefficient: 0.285,
		effect: clientdamage.Effect{Amount: 291, Variance: 0, PerLevel: 0, SpellLevel: 40, MaxLevel: 49}},
	{spellID: 1240721, effectSpellID: 1240724, level: 50, manaCost: 270, castMS: 0, coefficient: 0.285,
		effect: clientdamage.Effect{Amount: 482, Variance: 0, PerLevel: 0, SpellLevel: 50, MaxLevel: 59}},
	{spellID: 1316995, effectSpellID: 1316991, level: 60, manaCost: 355, castMS: 0, coefficient: 0.285,
		effect: clientdamage.Effect{Amount: 673, Variance: 0, PerLevel: 0, SpellLevel: 60, MaxLevel: 60}},
}

var prayerOfMendingRanks = []healRank{
	{spellID: 401859, level: 40, manaCost: 210, castMS: 0, coefficient: 0.429,
		effect: clientdamage.Effect{Amount: 172, Variance: 0, PerLevel: 0, SpellLevel: 40, MaxLevel: 0}},
	{spellID: 1240826, level: 50, manaCost: 305, castMS: 0, coefficient: 0.429,
		effect: clientdamage.Effect{Amount: 298, Variance: 0, PerLevel: 0, SpellLevel: 50, MaxLevel: 0}},
	{spellID: 1240827, level: 60, manaCost: 390, castMS: 0, coefficient: 0.429,
		effect: clientdamage.Effect{Amount: 413, Variance: 0, PerLevel: 0, SpellLevel: 60, MaxLevel: 0}},
}

// innerFireRanks are Inner Fire's ranks; the effect's amount is the armor
// the buff grants.
var innerFireRanks = []healRank{
	{spellID: 588, level: 12, manaCost: 30, effect: clientdamage.Effect{Amount: 315, SpellLevel: 12}},
	{spellID: 7128, level: 20, manaCost: 65, effect: clientdamage.Effect{Amount: 495, SpellLevel: 20}},
	{spellID: 602, level: 30, manaCost: 105, effect: clientdamage.Effect{Amount: 720, SpellLevel: 30}},
	{spellID: 1006, level: 40, manaCost: 165, effect: clientdamage.Effect{Amount: 945, SpellLevel: 40}},
	{spellID: 10951, level: 50, manaCost: 235, effect: clientdamage.Effect{Amount: 1170, SpellLevel: 50}},
	{spellID: 10952, level: 60, manaCost: 315, effect: clientdamage.Effect{Amount: 1395, SpellLevel: 60}},
}
