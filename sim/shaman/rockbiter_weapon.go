package shaman

import (
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

const RockbiterWeaponRanks = 7

var RockbiterWeaponEnchantId = [RockbiterWeaponRanks + 1]int32{0, 29, 6, 1, 503, 1663, 683, 1664}

// RockbiterWeaponBonusAP is each rank's attack power, its enchant spell's
// aura 99 (1.60.1.70707 SpellEffect and SpellLevels: 10400, 15567, 15568,
// 15569, 16311, 16312, 16313), growing by EffectRealPointsPerLevel from the
// rank's level to its MaxLevel. One number per rank gave a level-20 shaman
// rank 3's level-22 amount (118, not 108) and a level-30 one 138 where the
// client gives 177.6.
var RockbiterWeaponBonusAP = [RockbiterWeaponRanks + 1]clientdamage.Effect{
	{},
	{Amount: 29, PerLevel: 4.1, SpellLevel: 1, MaxLevel: 6},
	{Amount: 58, PerLevel: 3.5, SpellLevel: 8, MaxLevel: 14},
	{Amount: 88, PerLevel: 5, SpellLevel: 16, MaxLevel: 22},
	{Amount: 129, PerLevel: 8.1, SpellLevel: 24, MaxLevel: 32},
	{Amount: 211, PerLevel: 18, SpellLevel: 34, MaxLevel: 42},
	{Amount: 393, PerLevel: 16.1, SpellLevel: 44, MaxLevel: 52},
	{Amount: 554, PerLevel: 16.5, SpellLevel: 54, MaxLevel: 62},
}
var RockbiterWeaponBonusTPS = [RockbiterWeaponRanks + 1]float64{0, 6, 10, 16, 27, 41, 55, 72}
var RockbiterWeaponLevel = [RockbiterWeaponRanks + 1]int32{0, 1, 8, 16, 24, 34, 44, 54}

func (shaman *Shaman) RegisterRockbiterImbue(procMask core.ProcMask) {
	if procMask == core.ProcMaskUnknown {
		return
	}

	rank := weaponImbueRankAtLevel(RockbiterWeaponLevel[:], shaman.Level)
	enchantId := RockbiterWeaponEnchantId[rank]
	bonusThreat := RockbiterWeaponBonusTPS[rank]

	duration := time.Minute * 5

	hasMHImbue := procMask.Matches(core.ProcMaskMeleeMH)
	hasOHImbue := procMask.Matches(core.ProcMaskMeleeOH)

	if hasMHImbue {
		shaman.MainHand().TempEnchant = enchantId
		shaman.AutoAttacks.MHConfig().FlatThreatBonus += bonusThreat * shaman.AutoAttacks.MH().SwingSpeed
	}
	if hasOHImbue {
		shaman.OffHand().TempEnchant = enchantId
		shaman.AutoAttacks.MHConfig().FlatThreatBonus += bonusThreat * shaman.AutoAttacks.OH().SwingSpeed
	}

	shaman.OnSpellRegistered(func(spell *core.Spell) {
		if spell.ProcMask.Matches(procMask) {
			spell.FlatThreatBonus += bonusThreat
		}
	})

	aura := shaman.RegisterAura(core.Aura{
		Label:    "Rockbiter Imbue",
		Duration: duration,
	})

	shaman.RegisterOnItemSwapWithImbue(enchantId, &procMask, aura)
}

func (shaman *Shaman) ApplyRockbiterImbue(procMask core.ProcMask) {
	if procMask.Matches(core.ProcMaskMeleeMH) && shaman.HasMHWeapon() {
		shaman.ApplyRockbiterImbueToItem(shaman.MainHand())
	}

	if procMask.Matches(core.ProcMaskMeleeOH) && shaman.HasOHWeapon() {
		shaman.ApplyRockbiterImbueToItem(shaman.OffHand())
	}
}

// elementalWeaponsRockbiterBonus is the Rockbiter Weapon attack power
// multiplier by Elemental Weapons rank, Forever's text: "Increases the melee
// attack power bonus of your Rockbiter Weapon by 7/13/20%".
var elementalWeaponsRockbiterBonus = [4]float64{1, 1.07, 1.13, 1.20}

// rockbiterWeaponBonusAttackPower is the rank's attack power at the
// shaman's level, raised by Elemental Weapons.
func rockbiterWeaponBonusAttackPower(rank int32, elementalWeapons int32, level int32) float64 {
	return RockbiterWeaponBonusAP[rank].Center(int(level)) *
		elementalWeaponsRockbiterBonus[rankIndex(elementalWeapons, len(elementalWeaponsRockbiterBonus))]
}

func (shaman *Shaman) ApplyRockbiterImbueToItem(item *core.Item) {
	if item == nil {
		return
	}

	rank := weaponImbueRankAtLevel(RockbiterWeaponLevel[:], shaman.Level)
	enchantId := RockbiterWeaponEnchantId[rank]

	bonusAP := rockbiterWeaponBonusAttackPower(rank, shaman.Talents.ElementalWeapons, shaman.Level)

	newStats := stats.Stats{stats.AttackPower: bonusAP}

	item.Stats = item.Stats.Add(newStats)
	item.TempEnchant = enchantId
}
