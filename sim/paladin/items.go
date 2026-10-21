package paladin

import (
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

// Libram IDs
const (
	SanctifiedOrb      = 20512
	LibramOfHope       = 22401
	LibramOfFervor     = 23203
	LibramOfInvocation = 249442
	LibramOfLaw        = 272435
	LibramOfInfusion   = 279248
)

// relicClassMasks says which engine spells each client spell family is, for
// the librams in relic_mods_auto_gen.go. Only the families of spells this
// package registers are listed: a libram whose family is missing here
// (Holy Light, Cleanse, Swift Judgement) cannot be registered, and
// core.NewEquipModItemEffect refuses it.
//
// Families are the client's SpellClassOptions masks for the spells' own
// ids (Seal of the Crusader 20308, Seal of Command 20915, Seal of
// Righteousness 20154, Judgement of Righteousness 20286, Holy Shock
// 20473).
var relicClassMasks = core.ClassMaskTable{
	{Client: core.ClientClassMask{1 << 9}, Engine: PaladinSpellMaskSealOfTheCrusaderCast},
	{Client: core.ClientClassMask{1 << 25}, Engine: PaladinSpellMaskSealOfCommandCast},
	{Client: core.ClientClassMask{1 << 27}, Engine: PaladinSpellMaskSealOfRighteousnessCast},
	{Client: core.ClientClassMask{1 << 10}, Engine: PaladinSpellMaskJudgementOfRighteousness},
	{Client: core.ClientClassMask{1 << 21}, Engine: PaladinSpellMaskHolyShock},
}

// Libram of Fervor's two numbers, read from the client table: the effect on
// Seal of the Crusader's own first effect (property 3, the attack power
// bonus) and the effect on Judgement of the Crusader's (property 8, every
// effect: the holy damage increase).
const (
	clientModOpEffect1    int32 = 3
	clientModOpAllEffects int32 = 8
)

func libramOfFervorBonuses() (sealAttackPower, judgementHolyDamage float64) {
	for _, mod := range relicEquipMods[LibramOfFervor] {
		switch mod.Op {
		case clientModOpEffect1:
			sealAttackPower = float64(mod.Amount)
		case clientModOpAllEffects:
			judgementHolyDamage = float64(mod.Amount)
		}
	}
	if sealAttackPower == 0 || judgementHolyDamage == 0 {
		panic("Libram of Fervor: relic_mods_auto_gen.go lacks its Improved Seal of the Crusader effects")
	}
	return sealAttackPower, judgementHolyDamage
}

func init() {
	core.NewSimpleStatOffensiveTrinketEffect(SanctifiedOrb, stats.Stats{stats.Crit: 3 * core.CritRatingPerCritChance}, time.Second*25, time.Minute*3)

	// https://www.wowhead.com/classic/item=23203/libram-of-fervor
	// Equip: Increases the melee attack power bonus of your Seal of the
	// Crusader by 48 and the Holy damage increase of your Judgement of the
	// Crusader by 33.
	// Implemented in sotc.go (libramOfFervorBonuses).
	core.NewItemEffect(LibramOfFervor, func(core.Agent) {})

	// Client spell 1249005. Equip: Reduces the mana cost of your Seal spells by 5%.
	core.NewEquipModItemEffect(LibramOfInvocation, relicEquipMods[LibramOfInvocation], relicClassMasks)

	// Client spell 1291086. Equip: Increases the damage of your Judgement
	// ability by 4%. The client's family covers Judgement of Righteousness
	// and Judgement of Fury, not Judgement of Command.
	core.NewEquipModItemEffect(LibramOfLaw, relicEquipMods[LibramOfLaw], relicClassMasks)

	// Client spell 1306429. Equip: Increases the critical strike chance of
	// your Holy Shock spell by 6%.
	core.NewEquipModItemEffect(LibramOfInfusion, relicEquipMods[LibramOfInfusion], relicClassMasks)
}
