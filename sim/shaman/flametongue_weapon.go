package shaman

import (
	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

const FlametongueWeaponRanks = 6

var FlametongueWeaponSpellId = [FlametongueWeaponRanks + 1]int32{0, 8024, 8027, 8030, 16339, 16341, 16342}
var FlametongueWeaponEnchantId = [FlametongueWeaponRanks + 1]int32{0, 5, 4, 3, 523, 1665, 1666}

// FlametongueWeaponDamage is each rank's proc dummy (8026, 8028, 8029,
// 10445, 16343, 16344 in clients 1.60.1.70790 and 70888: Effect 3,
// EffectBasePointsF growing by EffectRealPointsPerLevel to the rank's
// SpellLevels.MaxLevel), the amount the rank tooltip spreads over weapon
// speeds 1.3 to 4.0 ("$m1/77 to $M1/25": rank 1 "5 to 18" at its cap of
// 440). A hit adds Amount x weapon speed / 100 Fire damage, so rank 1 is
// 3.26 a second of weapon speed at 10 and 4.40 from 16, rank 4 17.28 from
// 44, rank 6 28.10 from 60. The vanilla cap over four this replaces (18,
// 26, 42, 57, 85, 112) gave rank 1 4.5 at level 10 and rank 4 14.25 at
// its cap; foreverlogs.gg shamans land rank 2 on a 3.0 weapon for 17 at
// about 20 and 20 at its cap, rank 2 on a 3.4 for 23 (cap 22.2), rank 3
// on a 3.3 for 26 at about 28 (sim-accuracy 2026-11-12, shamans). The
// enchant (SpellItemEnchantment 5, 4, 3, 523, 1665, 1666) triggers 436519,
// ProcChance 100, so every main-hand hit adds it.
var FlametongueWeaponDamage = [FlametongueWeaponRanks + 1]clientdamage.Effect{
	{},
	{Amount: 326, PerLevel: 19, SpellLevel: 10, MaxLevel: 16},
	{Amount: 479, PerLevel: 29, SpellLevel: 18, MaxLevel: 24},
	{Amount: 716, PerLevel: 42, SpellLevel: 26, MaxLevel: 34},
	{Amount: 1144, PerLevel: 73, SpellLevel: 36, MaxLevel: 44},
	{Amount: 1876, PerLevel: 62, SpellLevel: 46, MaxLevel: 54},
	{Amount: 2498, PerLevel: 78, SpellLevel: 56, MaxLevel: 64},
}

// flametongueWeaponSpeedDivisor turns the dummy's amount into damage a
// second of weapon speed.
const flametongueWeaponSpeedDivisor = 100

// The levels the client teaches each rank at (SkillLineAbility); the
// imbue's rank is the highest one the shaman has learned, as for the
// other three imbues.
var FlametongueWeaponLevel = [FlametongueWeaponRanks + 1]int32{0, 10, 18, 26, 36, 46, 56}

func (shaman *Shaman) newFlametongueImbueSpell(weapon *core.Item) *core.Spell {
	rank := weaponImbueRankAtLevel(FlametongueWeaponLevel[:], shaman.Level)
	spellID := FlametongueWeaponSpellId[rank]

	spellCoeff := .1

	return shaman.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: spellID},
		SpellSchool: core.SpellSchoolFire,
		DefenseType: core.DefenseTypeMagic,
		ProcMask:    core.ProcMaskSpellDamageProc,
		Flags:       core.SpellFlagNoOnCastComplete | core.SpellFlagPassiveSpell,

		DamageMultiplier: []float64{1, 1.05, 1.1, 1.15}[shaman.Talents.ElementalWeapons],
		ThreatMultiplier: 1,
		BonusCoefficient: spellCoeff,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			if weapon.SwingSpeed != 0 {
				damage := flametongueWeaponHitDamage(rank, shaman.Level, weapon.SwingSpeed)
				spell.CalcAndDealDamage(sim, target, damage, spell.OutcomeMagicHitAndCrit)
			}
		},
	})
}

// flametongueWeaponHitDamage is the Fire damage one hit adds at the rank,
// level and weapon speed.
func flametongueWeaponHitDamage(rank int32, level int32, weaponSpeed float64) float64 {
	return FlametongueWeaponDamage[rank].Center(int(level)) * weaponSpeed / flametongueWeaponSpeedDivisor
}

func (shaman *Shaman) ApplyFlametongueImbueToItem(item *core.Item) {
	if item == nil {
		return
	}

	rank := weaponImbueRankAtLevel(FlametongueWeaponLevel[:], shaman.Level)
	enchantId := FlametongueWeaponEnchantId[rank]

	item.TempEnchant = enchantId
}

func (shaman *Shaman) ApplyFlametongueImbue(procMask core.ProcMask) {
	if procMask.Matches(core.ProcMaskMeleeMH) && shaman.HasMHWeapon() {
		shaman.ApplyFlametongueImbueToItem(shaman.MainHand())
	}

	if procMask.Matches(core.ProcMaskMeleeOH) && shaman.HasOHWeapon() {
		shaman.ApplyFlametongueImbueToItem(shaman.OffHand())
	}
}

func (shaman *Shaman) RegisterFlametongueImbue(procMask core.ProcMask) {
	if procMask == core.ProcMaskUnknown && !shaman.ItemSwap.IsEnabled() {
		return
	}

	rank := weaponImbueRankAtLevel(FlametongueWeaponLevel[:], shaman.Level)
	enchantId := FlametongueWeaponEnchantId[rank]

	mhSpell := shaman.newFlametongueImbueSpell(shaman.MainHand())
	ohSpell := shaman.newFlametongueImbueSpell(shaman.OffHand())

	aura := shaman.RegisterAura(core.Aura{
		Label:    "Flametongue Imbue",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if !result.Landed() || !spell.ProcMask.Matches(procMask) {
				return
			}

			if spell.IsMH() {
				mhSpell.Cast(sim, result.Target)
			} else {
				ohSpell.Cast(sim, result.Target)
			}
		},
	})

	shaman.RegisterOnItemSwapWithImbue(enchantId, &procMask, aura)
}
