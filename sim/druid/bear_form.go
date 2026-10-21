package druid

import (
	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

// Bear Form and Dire Bear Form, from build 1.60.1.70009's spellconst. One
// Druid form value (Bear) stands for both: Dire Bear Form (spell 9634,
// learned at 40) replaces Bear Form (spell 5487, learned at 10) the way a
// new rank would, so a level 40 and up druid reads the second tier.
//
// Each form is a shapeshift spell plus a passive that carries its numbers:
//
//   - Bear Form (Passive) 1178 / Dire Bear Form (Passive) 9635: armor from
//     items +180% / +360% (aura 142, base resistance percent, and aura 466,
//     which pairs with it on every armor form: Moonkin Form's passive 24905
//     carries the same two at 360), max health (aura 230: 20 + 18 a level /
//     600 + 32 a level) and attack power (aura 99: 30 + 3 a level / 120 + 3
//     a level, which is 3 a level at every level the tier is used).
//   - Bear Form (Passive2) 21178: +30% threat (aura 10, all schools).
//
// unconfirmed: what aura 466 modifies. It is read as the same percentage
// applied to the armor items carry as bonus armor (stats.BonusArmor), the
// pool aura 142 does not reach.
const (
	// bearFormThreatMultiplier is spell 21178's aura 10 amount 50 (30 until
	// build 1.60.1.70291).
	bearFormThreatMultiplier = 1.5

	// bearFormAnnouncedMinLevel is Bear Form's own learn level.
	bearFormAnnouncedMinLevel = 10
)

// bearFormTier is one of the two forms' client numbers.
type bearFormTier struct {
	FormSpellID    int32
	PassiveSpellID int32
	LearnLevel     int
	// ArmorFromItemsPercent is auras 142 and 466: +N% of the armor items
	// carry.
	ArmorFromItemsPercent float64
	Health                clientdamage.Effect
	AttackPower           clientdamage.Effect
}

var bearFormTiers = [...]bearFormTier{
	{
		FormSpellID: 5487, PassiveSpellID: 1178, LearnLevel: bearFormAnnouncedMinLevel,
		ArmorFromItemsPercent: 180,
		Health:                clientdamage.Effect{Amount: 20, PerLevel: 18, SpellLevel: 10, MaxLevel: 40},
		AttackPower:           clientdamage.Effect{Amount: 30, PerLevel: 3, SpellLevel: 10, MaxLevel: 40},
	},
	{
		FormSpellID: 9634, PassiveSpellID: 9635, LearnLevel: 40,
		ArmorFromItemsPercent: 360,
		Health:                clientdamage.Effect{Amount: 600, PerLevel: 32, SpellLevel: 40, MaxLevel: 70},
		AttackPower:           clientdamage.Effect{Amount: 120, PerLevel: 3, SpellLevel: 40, MaxLevel: 70},
	},
}

// bearFormTierAt is the highest tier a druid of the level has learned, or
// false below Bear Form's learn level.
func bearFormTierAt(level int32) (bearFormTier, bool) {
	for i := len(bearFormTiers) - 1; i >= 0; i-- {
		if int(level) >= bearFormTiers[i].LearnLevel {
			return bearFormTiers[i], true
		}
	}
	return bearFormTier{}, false
}

// ArmorMultiplier is the factor the form puts on the armor items carry.
func (tier bearFormTier) ArmorMultiplier() float64 {
	return 1 + tier.ArmorFromItemsPercent/100
}

// Both bear forms share the paw: 54.8 damage per second at 2.5 s a swing,
// rolled 20% either side (the cat's claw is the same 54.8 dps at 1.0 s;
// 54.8 is also the baseline the weapon's Feral Attack Power is counted
// from). The client states no form weapon, so this is Era's.
//
// unconfirmed: the paw's damage and speed; Forever's rage normalization
// (sim/core/rage.go) reads the speed as a one-handed weapon's.
const (
	formPawDamagePerSecond = 54.8
	bearPawSwingSpeed      = 2.5
	pawDamageSpread        = 0.2
)

func (druid *Druid) GetBearWeapon() core.Weapon {
	average := formPawDamagePerSecond * bearPawSwingSpeed
	return core.Weapon{
		BaseDamageMin:        average * (1 - pawDamageSpread),
		BaseDamageMax:        average * (1 + pawDamageSpread),
		SwingSpeed:           bearPawSwingSpeed,
		NormalizedSwingSpeed: bearPawSwingSpeed,
		AttackPowerPerDPS:    core.DefaultAttackPowerPerDPS,
	}
}

// bearFormStatBonus is what shifting adds to the character's stats at its
// level: the form's attack power and health, plus the talents that pay in
// feral forms.
func (druid *Druid) bearFormStatBonus(tier bearFormTier) stats.Stats {
	level := int(druid.Level)
	return druid.GetFormShiftStats().Add(stats.Stats{
		stats.AttackPower: tier.AttackPower.Center(level),
		stats.Health:      tier.Health.Center(level),
	})
}

// heartOfTheWildBearStaminaPerRank is Heart of the Wild's "while in Bear
// Form or Dire Bear Form your Stamina is increased by 4%" a rank.
const heartOfTheWildBearStaminaPerRank = 0.04

// bearFormArmorPools are the two stats the form's two armor auras scale.
var bearFormArmorPools = [...]stats.Stat{stats.Armor, stats.BonusArmor}

func (druid *Druid) registerBearFormSpell() {
	tier, learned := bearFormTierAt(druid.Level)
	if !learned {
		// Below Bear Form's learn level there is no form; the aura still
		// exists so a level 1 bear (the smoke test builds one) is a bear
		// with a paw and none of the form's numbers.
		tier = bearFormTiers[0]
	}

	actionID := core.ActionID{SpellID: tier.FormSpellID}

	statBonus := core.Ternary(learned, druid.bearFormStatBonus(tier), druid.GetFormShiftStats())
	armorMultiplier := core.TernaryFloat64(learned, tier.ArmorMultiplier(), 1)

	stamDep := druid.NewDynamicMultiplyStat(stats.Stamina,
		1+heartOfTheWildBearStaminaPerRank*float64(clampRank(druid.Talents.HeartOfTheWild, heartOfTheWildMaxRank)))
	feralApDep := druid.NewDynamicStatDependency(stats.FeralAttackPower, stats.AttackPower, 1)

	pawWeapon := druid.GetBearWeapon()
	thickHideArmor := stats.Stats{}

	druid.BearFormAura = druid.RegisterAura(core.Aura{
		Label:      "Bear Form",
		ActionID:   actionID,
		Duration:   core.NeverExpires,
		BuildPhase: core.Ternary(druid.StartingForm.Matches(Bear), core.CharacterBuildPhaseBase, core.CharacterBuildPhaseNone),
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			if !druid.Env.MeasuringStats && druid.form != Humanoid {
				druid.CancelShapeshift(sim)
			}
			druid.form = Bear
			druid.SetCurrentPowerBar(core.RageBar)
			druid.AutoAttacks.SetMH(pawWeapon)

			druid.PseudoStats.ThreatMultiplier *= bearFormThreatMultiplier
			druid.SetShapeshift(aura)

			// Armor: the form multiplies what items carry, and Thick Hide's
			// base armor is "further increased by multipliers from those
			// forms".
			for _, pool := range bearFormArmorPools {
				druid.ApplyDynamicEquipScaling(sim, pool, armorMultiplier)
			}
			thickHideArmor = druid.thickHideArmor(armorMultiplier)
			druid.AddStatsDynamic(sim, thickHideArmor)

			// Health is preserved as a fraction across the shift.
			healthFraction := druid.CurrentHealthPercent()
			druid.AddStatsDynamic(sim, statBonus)
			druid.EnableDynamicStatDep(sim, feralApDep)
			druid.EnableDynamicStatDep(sim, stamDep)
			if !druid.Env.MeasuringStats {
				druid.SetHealthFraction(healthFraction)
			}

			if !druid.Env.MeasuringStats {
				druid.AutoAttacks.SetReplaceMHSwing(druid.ReplaceBearMHFunc)
				druid.AutoAttacks.EnableAutoSwing(sim)
				druid.manageCooldownsEnabled()
				druid.UpdateManaRegenRates()
			}
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			druid.form = Humanoid
			druid.SetCurrentPowerBar(core.ManaBar)
			druid.AutoAttacks.SetMH(druid.WeaponFromMainHand())

			druid.PseudoStats.ThreatMultiplier /= bearFormThreatMultiplier
			druid.SetShapeshift(nil)

			for _, pool := range bearFormArmorPools {
				druid.RemoveDynamicEquipScaling(sim, pool, armorMultiplier)
			}
			druid.AddStatsDynamic(sim, thickHideArmor.Invert())

			healthFraction := druid.CurrentHealthPercent()
			druid.AddStatsDynamic(sim, statBonus.Invert())
			druid.DisableDynamicStatDep(sim, feralApDep)
			druid.DisableDynamicStatDep(sim, stamDep)
			if !druid.Env.MeasuringStats {
				druid.SetHealthFraction(healthFraction)
			}

			if !druid.Env.MeasuringStats {
				druid.AutoAttacks.SetReplaceMHSwing(nil)
				druid.AutoAttacks.EnableAutoSwing(sim)
				druid.manageCooldownsEnabled()
				druid.UpdateManaRegenRates()
				druid.deactivateBearAuras(sim)
			}
		},
	})

	rageMetrics := druid.NewRageMetrics(actionID)

	druid.BearForm = druid.RegisterSpell(Any, core.SpellConfig{
		ActionID: actionID,
		Flags:    core.SpellFlagNoOnCastComplete | core.SpellFlagAPL,

		RequiredLevel: tier.LearnLevel,

		// 55% of base mana, as Cat Form (spellconst cost_pct on 5487 and
		// 9634); Natural Shapeshifter takes 10% a rank off.
		ManaCost: core.ManaCostOptions{
			BaseCost:   bearFormManaFractionOfBase,
			Multiplier: 100 - 10*druid.Talents.NaturalShapeshifter,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			IgnoreHaste: true,
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			// Shifting empties the rage bar; Furor pays rage back.
			rageAfterShift := 0.0
			if sim.Proc(furorBearFormRageChance(druid.Talents.Furor), "Furor") {
				rageAfterShift = furorBearFormRage
			}
			rageDelta := rageAfterShift - druid.CurrentRage()
			if rageDelta > 0 {
				druid.AddRage(sim, rageDelta, rageMetrics)
			} else if rageDelta < 0 {
				druid.SpendRage(sim, -rageDelta, rageMetrics)
			}
			druid.BearFormAura.Activate(sim)
		},
	})
}

// bearFormManaFractionOfBase is cost_pct 55 on Bear Form and Dire Bear
// Form.
const bearFormManaFractionOfBase = 0.55

// deactivateBearAuras ends what only a bear can hold when the form ends.
func (druid *Druid) deactivateBearAuras(sim *core.Simulation) {
	for _, aura := range []*core.Aura{druid.EnrageAura, druid.MaulQueueAura, druid.FrenziedRegenerationAura} {
		if aura != nil {
			aura.Deactivate(sim)
		}
	}
}
