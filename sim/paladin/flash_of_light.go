package paladin

import (
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
)

const (
	flashOfLightCastTime    = 1500 * time.Millisecond
	flashOfLightCoefficient = 0.429
)

// flashOfLightRanks are the six ranks the client teaches (trainables
// "Flash of Light"). The generated FlashOfLight* tables carry an NPC
// variant at index 0 (1313346) and are not used; TestFlashOfLightMatchesClient
// pins every rank to the client.
var flashOfLightRanks = []healRank{
	{spellID: 19750, level: 20, manaCost: 35, heal: clientdamage.Effect{Amount: 46, Variance: 0.149254, PerLevel: 1.0, SpellLevel: 20, MaxLevel: 25}},
	{spellID: 19939, level: 26, manaCost: 50, heal: clientdamage.Effect{Amount: 66, Variance: 0.135922, PerLevel: 1.3, SpellLevel: 26, MaxLevel: 31}},
	{spellID: 19940, level: 34, manaCost: 70, heal: clientdamage.Effect{Amount: 101, Variance: 0.116883, PerLevel: 1.6, SpellLevel: 34, MaxLevel: 39}},
	{spellID: 19941, level: 42, manaCost: 90, heal: clientdamage.Effect{Amount: 151, Variance: 0.114833, PerLevel: 1.9, SpellLevel: 42, MaxLevel: 47}},
	{spellID: 19942, level: 50, manaCost: 115, heal: clientdamage.Effect{Amount: 223, Variance: 0.113074, PerLevel: 2.2, SpellLevel: 50, MaxLevel: 55}},
	{spellID: 19943, level: 58, manaCost: 140, heal: clientdamage.Effect{Amount: 303, Variance: 0.110193, PerLevel: 2.6, SpellLevel: 58, MaxLevel: 63}},
}

func (paladin *Paladin) registerFlashOfLight() {
	paladin.registerDirectHeal(directHeal{
		ranks:       flashOfLightRanks,
		castTime:    flashOfLightCastTime,
		coefficient: flashOfLightCoefficient,
		spellCode:   SpellCode_PaladinFlashOfLight,
		classMask:   PaladinSpellMaskFlashOfLight,
		blessing:    blessingOfLightFlashOfLight,
	})
}
