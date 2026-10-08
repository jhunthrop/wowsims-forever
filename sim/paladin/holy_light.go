package paladin

import (
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
)

const (
	holyLightCastTime    = 2500 * time.Millisecond
	holyLightCoefficient = 0.714
)

// holyLightRanks are the nine ranks the client teaches (trainables
// "Holy Light"), each with the heal effect of its own row. The generated
// HolyLight* tables are not used: they carry an NPC variant at index 0
// (1313351) and shift every rank by one. TestHolyLightMatchesClient pins
// every rank to the client.
var holyLightRanks = []healRank{
	{spellID: 635, level: 1, manaCost: 35, heal: clientdamage.Effect{Amount: 43, Variance: 0.186047, PerLevel: 0.8, SpellLevel: 1, MaxLevel: 5}},
	{spellID: 639, level: 6, manaCost: 60, heal: clientdamage.Effect{Amount: 83, Variance: 0.168675, PerLevel: 1.1, SpellLevel: 6, MaxLevel: 11}},
	{spellID: 647, level: 14, manaCost: 110, heal: clientdamage.Effect{Amount: 155, Variance: 0.16185, PerLevel: 1.7, SpellLevel: 14, MaxLevel: 19}},
	{spellID: 1026, level: 22, manaCost: 190, heal: clientdamage.Effect{Amount: 287, Variance: 0.138138, PerLevel: 2.4, SpellLevel: 22, MaxLevel: 27}},
	{spellID: 1042, level: 30, manaCost: 275, heal: clientdamage.Effect{Amount: 452, Variance: 0.118774, PerLevel: 3.1, SpellLevel: 30, MaxLevel: 35}},
	{spellID: 3472, level: 38, manaCost: 365, heal: clientdamage.Effect{Amount: 646, Variance: 0.110961, PerLevel: 3.8, SpellLevel: 38, MaxLevel: 43}},
	{spellID: 10328, level: 46, manaCost: 465, heal: clientdamage.Effect{Amount: 899, Variance: 0.108108, PerLevel: 4.6, SpellLevel: 46, MaxLevel: 51}},
	{spellID: 10329, level: 54, manaCost: 580, heal: clientdamage.Effect{Amount: 1217, Variance: 0.107821, PerLevel: 5.2, SpellLevel: 54, MaxLevel: 59}},
	{spellID: 25292, level: 60, manaCost: 660, heal: clientdamage.Effect{Amount: 1580, Variance: 0.107143, PerLevel: 5.8, SpellLevel: 60, MaxLevel: 65}},
}

func (paladin *Paladin) registerHolyLight() {
	paladin.registerDirectHeal(directHeal{
		ranks:       holyLightRanks,
		castTime:    holyLightCastTime,
		coefficient: holyLightCoefficient,
		spellCode:   SpellCode_PaladinHolyLight,
		classMask:   PaladinSpellMaskHolyLight,
		blessing:    blessingOfLightHolyLight,
	})
}
