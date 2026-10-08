package priest

import (
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

// Per-rank values of the healing talents, read off the client's rank
// descriptions for build 1.60.1.70009 (data/builds/1.60.1.70009/talents/
// priest.json). Some ranks are not a flat step (Improved Power Word:
// Shield is 7/14/20, Mental Agility 3/7/10, Spiritual Healing 3/7/10,
// Inspiration 8/17/25), so those are tables indexed by rank. A Forever
// patch that changes a number changes a line here and nothing else.
const (
	mentalStrengthIntellectPerRank    = 0.03
	spiritualGuidanceHealingPerSpirit = 0.05
	improvedRenewHealingPerRank       = 0.05
	improvedHealingCostPctPerRank     = -5
	divineFuryCastTimePerRank         = -100 * time.Millisecond
	soulWardingCostPct                = -15
	soulWardingCooldown               = -4 * time.Second
	divineAegisAbsorbPerRank          = 0.05
	divineAegisSpellID                = 431624
	divineAegisDuration               = 12 * time.Second
	renewedHopeCritPerRank            = 2.0 // percentage points of crit
	renewedHopeWeakenedSoulPerRank    = time.Second
	litanyOfLightManaPerRank          = 0.05
	litanyOfLightManaSpellID          = 1317006
	inspirationAuraDuration           = 15 * time.Second
	inspirationTalentSpellID          = 14892
)

var (
	improvedPowerWordShieldAbsorb = [...]float64{0, 0.07, 0.14, 0.20}
	mentalAgilityCostPct          = [...]int64{0, -3, -7, -10}
	spiritualHealingHealing       = [...]float64{0, 0.03, 0.07, 0.10}
	inspirationArmor              = [...]float64{0, 0.08, 0.17, 0.25}
)

// applyDeclarativeHealingTalents is every healing talent that is a pure
// modifier on a set of spells, in the same style as the Shadow ones.
func (priest *Priest) applyDeclarativeHealingTalents() {
	t := priest.Talents
	var mods []core.SpellModConfig

	if rank := rankOf("improved_renew", t.ImprovedRenew); rank > 0 {
		mods = append(mods, core.SpellModConfig{
			Kind: core.SpellMod_DamageDone_Pct, ClassMask: PriestSpellMaskRenew,
			FloatValue: 1 + improvedRenewHealingPerRank*float64(rank),
		})
	}
	if rank := rankOf("improved_power_word_shield", t.ImprovedPowerWordShield); rank > 0 {
		mods = append(mods, core.SpellModConfig{
			Kind: core.SpellMod_DamageDone_Pct, ClassMask: PriestSpellMaskPowerWordShield,
			FloatValue: 1 + improvedPowerWordShieldAbsorb[rank],
		})
	}
	if rank := rankOf("improved_healing", t.ImprovedHealing); rank > 0 {
		mods = append(mods, core.SpellModConfig{
			Kind: core.SpellMod_PowerCost_Pct,
			ClassMask: PriestSpellMaskLesserHeal | PriestSpellMaskHeal | PriestSpellMaskGreaterHeal |
				PriestSpellMaskPenance | PriestSpellMaskPrayerOfMending,
			IntValue: improvedHealingCostPctPerRank * int64(rank),
		})
	}
	if rank := rankOf("mental_agility", t.MentalAgility); rank > 0 {
		mods = append(mods, core.SpellModConfig{
			Kind:      core.SpellMod_PowerCost_Pct,
			ClassMask: PriestSpellMaskSmite | PriestSpellMaskHolyFire | PriestSpellMaskInstantCast,
			IntValue:  mentalAgilityCostPct[rank],
		})
	}
	if rank := rankOf("divine_fury", t.DivineFury); rank > 0 {
		mods = append(mods, core.SpellModConfig{
			Kind:      core.SpellMod_CastTime_Flat,
			ClassMask: PriestSpellMaskSmite | PriestSpellMaskHolyFire | PriestSpellMaskHeal | PriestSpellMaskGreaterHeal,
			TimeValue: divineFuryCastTimePerRank * time.Duration(rank),
		})
	}
	if t.SoulWarding {
		mods = append(mods,
			core.SpellModConfig{Kind: core.SpellMod_PowerCost_Pct, ClassMask: PriestSpellMaskPowerWordShield, IntValue: soulWardingCostPct},
			core.SpellModConfig{Kind: core.SpellMod_Cooldown_Flat, ClassMask: PriestSpellMaskPowerWordShield, TimeValue: soulWardingCooldown},
		)
	}
	for _, mod := range mods {
		priest.AddStaticMod(mod)
	}

	// "Increases the amount healed by your spells": a shield is not healed,
	// so this is the healing multiplier and not the spell's damage one.
	if rank := rankOf("spiritual_healing", t.SpiritualHealing); rank > 0 {
		priest.PseudoStats.HealingDealtMultiplier *= 1 + spiritualHealingHealing[rank]
	}
}

// renewedHopeCrit is the extra crit rating Renewed Hope adds to a heal it
// names on a target carrying Weakened Soul.
func (priest *Priest) renewedHopeCrit(spell *core.Spell, target *core.Unit) float64 {
	if priest.Talents.RenewedHope == 0 || spell.ClassSpellMask&PriestSpellMaskRenewedHope == 0 {
		return 0
	}
	if weakened := priest.weakenedSoul(target); weakened == nil || !weakened.IsActive() {
		return 0
	}
	rank := rankOf("renewed_hope", priest.Talents.RenewedHope)
	return renewedHopeCritPerRank * float64(rank) * core.CritRatingPerCritChance
}

// shortenWeakenedSoul is Renewed Hope's second half: a heal it names takes
// time off the target's Weakened Soul.
func (priest *Priest) shortenWeakenedSoul(sim *core.Simulation, spell *core.Spell, target *core.Unit) {
	if priest.Talents.RenewedHope == 0 || spell.ClassSpellMask&PriestSpellMaskRenewedHope == 0 {
		return
	}
	weakened := priest.weakenedSoul(target)
	if weakened == nil || !weakened.IsActive() {
		return
	}
	cut := renewedHopeWeakenedSoulPerRank * time.Duration(rankOf("renewed_hope", priest.Talents.RenewedHope))
	if weakened.RemainingDuration(sim) <= cut {
		weakened.Deactivate(sim)
		return
	}
	weakened.UpdateExpires(sim, weakened.ExpiresAt()-cut)
}

// applyDivineAegis turns the priest's critical heals into an absorb on the
// target. Assumption: the client states no stacking rule, so a second
// critical heal within the 12 s adds to the shield and renews it, which is
// how the talent plays in later versions of the game.
func (priest *Priest) applyDivineAegis() {
	rank := rankOf("divine_aegis", priest.Talents.DivineAegis)
	if rank == 0 {
		return
	}
	absorbShare := divineAegisAbsorbPerRank * float64(rank)
	aegis := priest.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: divineAegisSpellID},
		SpellSchool: core.SpellSchoolHoly,
		ProcMask:    core.ProcMaskEmpty,
		Flags:       SpellFlagPriest | core.SpellFlagHelpful | core.SpellFlagNoOnCastComplete,
		Shield: core.ShieldConfig{
			Aura: core.Aura{Label: "Divine Aegis", Duration: divineAegisDuration},
		},
		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		ApplyEffects:     func(*core.Simulation, *core.Unit, *core.Spell) {},
	})
	priest.RegisterAura(core.Aura{
		Label:    "Divine Aegis Talent",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnHealDealt: func(_ *core.Aura, sim *core.Simulation, _ *core.Spell, result *core.SpellResult) {
			if !result.DidCrit() {
				return
			}
			shield := aegis.Shield(result.Target)
			shield.Apply(sim, shield.Remaining()+result.Damage*absorbShare)
		},
	})
}

// applyInspiration raises the armor of whoever the priest critically
// heals with a non-periodic heal: OnHealDealt is not called for ticks.
func (priest *Priest) applyInspiration() {
	rank := rankOf("inspiration", priest.Talents.Inspiration)
	if rank == 0 {
		return
	}
	auras := priest.NewRaidAuraArray(func(unit *core.Unit) *core.Aura {
		armor := unit.NewDynamicMultiplyStat(stats.Armor, 1+inspirationArmor[rank])
		return unit.GetOrRegisterAura(core.Aura{
			Label:    "Inspiration",
			ActionID: core.ActionID{SpellID: inspirationTalentSpellID},
			Duration: inspirationAuraDuration,
			OnGain: func(_ *core.Aura, sim *core.Simulation) {
				unit.EnableDynamicStatDep(sim, armor)
			},
			OnExpire: func(_ *core.Aura, sim *core.Simulation) {
				unit.DisableDynamicStatDep(sim, armor)
			},
		})
	})
	priest.RegisterAura(core.Aura{
		Label:    "Inspiration Talent",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnHealDealt: func(_ *core.Aura, sim *core.Simulation, _ *core.Spell, result *core.SpellResult) {
			if result.DidCrit() {
				auras.Get(result.Target).Activate(sim)
			}
		},
	})
}

// applyLitanyOfLight returns mana when a heal follows a different heal.
// Assumption: the first heal of a fight has no previous heal to differ
// from, so it returns nothing; a shield is not a heal.
func (priest *Priest) applyLitanyOfLight() {
	rank := rankOf("litany_of_light", priest.Talents.LitanyOfLight)
	if rank == 0 {
		return
	}
	metrics := priest.NewManaMetrics(core.ActionID{SpellID: litanyOfLightManaSpellID})
	priest.RegisterAura(core.Aura{
		Label:    "Litany of Light",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			priest.lastHealSpellCode = SpellCode_PriestNone
			aura.Activate(sim)
		},
		OnCastComplete: func(_ *core.Aura, sim *core.Simulation, spell *core.Spell) {
			if !isHealingSpell(spell) {
				return
			}
			previous := priest.lastHealSpellCode
			priest.lastHealSpellCode = spell.SpellCode
			if previous != SpellCode_PriestNone && previous != spell.SpellCode && spell.Cost != nil {
				priest.AddMana(sim, spell.Cost.BaseCost*litanyOfLightManaPerRank*float64(rank), metrics)
			}
		},
	})
}

// isHealingSpell is whether spell heals (a shield absorbs and does not).
func isHealingSpell(spell *core.Spell) bool {
	return spell.ProcMask.Matches(core.ProcMaskSpellHealing) && spell.ClassSpellMask&PriestSpellMaskPowerWordShield == 0
}
