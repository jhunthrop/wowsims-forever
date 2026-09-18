package hunter

import (
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

// FOREVER: the client's trait trees replaced vanilla's, so some of the
// talents this file reaches for no longer exist under these names, and
// some changed their rank count and so their proto type. Their behaviour
// is rewritten when this spec is brought up, in rankings-population order
// (design section 2.3). Every site is commented rather than deleted, so
// the diff shows a reviewer exactly what the old tree did, and each is
// left reading the value an untalented character would have read - which
// is what a talent nobody can now take is worth.

func (hunter *Hunter) ApplyTalents() {
	if hunter.pet != nil {
		hunter.applyFrenzy()
		hunter.registerBestialWrathCD()

		// Forever: merged from MeleeCrit 3/rank + SpellCrit 3/rank. One
		// effect under a unified stat gets one write.
		hunter.pet.AddStat(stats.Crit, core.CritRatingPerCritChance*3*float64(hunter.Talents.Ferocity))

		hunter.pet.PseudoStats.DamageDealtMultiplier *= 1 + 0.04*float64(hunter.Talents.UnleashedFury)

		if hunter.Talents.EnduranceTraining > 0 {
			hunter.pet.MultiplyStat(stats.Health, 1+(0.03*float64(hunter.Talents.EnduranceTraining)))
		}
	}

	/*
		if hunter.Talents.MonsterSlaying+hunter.Talents.HumanoidSlaying > 0 {
			hunter.Env.RegisterPostFinalizeEffect(func() {
				for _, t := range hunter.Env.Encounter.Targets {
					switch t.MobType {
					case proto.MobType_MobTypeHumanoid:
						multiplier := []float64{1, 1.01, 1.02, 1.03}[hunter.Talents.HumanoidSlaying]
						for _, at := range hunter.AttackTables[t.UnitIndex] {
							at.DamageDealtMultiplier *= multiplier
							at.CritMultiplier *= multiplier
						}
					case proto.MobType_MobTypeBeast, proto.MobType_MobTypeGiant, proto.MobType_MobTypeDragonkin:
						multiplier := []float64{1, 1.01, 1.02, 1.03}[hunter.Talents.MonsterSlaying]
						for _, at := range hunter.AttackTables[t.UnitIndex] {
							at.DamageDealtMultiplier *= multiplier
							at.CritMultiplier *= multiplier
						}
					}
				}
			})
		}
	*/

	if hunter.Talents.BestialDiscipline > 0 {
		core.MakePermanent(hunter.RegisterAura(core.Aura{
			Label: "Bestial Discipline",
			OnInit: func(aura *core.Aura, sim *core.Simulation) {
				if hunter.pet != nil {
					hunter.pet.AddFocusRegenMultiplier(0.1 * float64(hunter.Talents.BestialDiscipline))
				}
			},
		}))
	}

	// Forever: merged from MeleeHit 1/rank + SpellHit 1/rank. One effect
	// under a unified stat gets one write.
	hunter.AddStat(stats.Hit, float64(hunter.Talents.Surefooted)*1*core.HitRatingPerHitChance)

	/*
		hunter.AddStat(stats.Crit, float64(hunter.Talents.KillerInstinct)*1*core.CritRatingPerCritChance)
	*/

	/*
		if hunter.Talents.LethalShots > 0 {
			lethalBonus := 1 * float64(hunter.Talents.LethalShots) * core.CritRatingPerCritChance
			hunter.OnSpellRegistered(func(spell *core.Spell) {
				if spell.Flags.Matches(SpellFlagShot) {
					spell.BonusCritRating += lethalBonus
				}
			})
			hunter.AutoAttacks.RangedConfig().BonusCritRating += lethalBonus
		}
	*/

	if hunter.Talents.RangedWeaponSpecialization > 0 {
		mult := 1 + 0.01*float64(hunter.Talents.RangedWeaponSpecialization)
		hunter.OnSpellRegistered(func(spell *core.Spell) {
			if spell.ProcMask.Matches(core.ProcMaskRanged) && spell.SpellCode != SpellCode_HunterSerpentSting {
				spell.DamageMultiplier *= mult
			}
		})
	}

	if hunter.Talents.Survivalist > 0 {
		hunter.MultiplyStat(stats.Health, 1.0+0.02*float64(hunter.Talents.Survivalist))
	}

	if hunter.Talents.LightningReflexes > 0 {
		agiBonus := 0.03 * float64(hunter.Talents.LightningReflexes)
		hunter.MultiplyStat(stats.Agility, 1.0+agiBonus)
	}

	hunter.applyEfficiency()
	hunter.applyTrapMastery()
	hunter.applyCleverTraps()
}

func (hunter *Hunter) applyFrenzy() {
	if hunter.Talents.Frenzy == 0 {
		return
	}

	procChance := 0.2 * float64(hunter.Talents.Frenzy)

	procAura := hunter.pet.RegisterAura(core.Aura{
		Label:    "Frenzy Proc",
		ActionID: core.ActionID{SpellID: 19625},
		Duration: time.Second * 8,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.MultiplyAttackSpeed(sim, 1.3)
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.MultiplyAttackSpeed(sim, 1/1.3)
		},
	})

	hunter.pet.RegisterAura(core.Aura{
		Label:    "Frenzy",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, spellResult *core.SpellResult) {
			if !spellResult.Outcome.Matches(core.OutcomeCrit) {
				return
			}
			if procChance == 1 || sim.RandomFloat("Frenzy") < procChance {
				procAura.Activate(sim)
			}
		},
	})
}

func (hunter *Hunter) registerBestialWrathCD() {
	if !hunter.Talents.BestialWrath {
		return
	}

	actionID := core.ActionID{SpellID: 19574}

	hunter.BestialWrathPetAura = hunter.pet.RegisterAura(core.Aura{
		Label:    "Bestial Wrath Pet",
		ActionID: actionID,
		Duration: time.Second * 18,
	}).AttachMultiplicativePseudoStatBuff(&hunter.pet.PseudoStats.DamageDealtMultiplier, 1.5)

	bwSpell := hunter.RegisterSpell(core.SpellConfig{
		ActionID: actionID,
		Flags:    core.SpellFlagAPL,

		ManaCost: core.ManaCostOptions{
			BaseCost: 0.12,
		},

		Cast: core.CastConfig{
			CD: core.Cooldown{
				Timer:    hunter.NewTimer(),
				Duration: time.Minute * 2,
			},
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			hunter.BestialWrathPetAura.Activate(sim)
		},
	})

	hunter.AddMajorCooldown(core.MajorCooldown{
		Spell: bwSpell,
		Type:  core.CooldownTypeDPS,
	})
}

func (hunter *Hunter) mortalShots() float64 {
	return 0.06 * float64(hunter.Talents.MortalShots)
}

func (hunter *Hunter) applyTrapMastery() {
	/*
		if hunter.Talents.TrapMastery == 0 {
			return
		}

		hunter.OnSpellRegistered(func(spell *core.Spell) {
			if spell.Flags.Matches(SpellFlagTrap) {
				spell.BonusHitRating += 5 * float64(hunter.Talents.TrapMastery)
			}
		})
	*/
}

func (hunter *Hunter) applyCleverTraps() {
	if hunter.Talents.CleverTraps == 0 {
		return
	}

	hunter.OnSpellRegistered(func(spell *core.Spell) {
		if spell.Flags.Matches(SpellFlagTrap) {
			spell.DamageMultiplier *= 1 + 0.15*float64(hunter.Talents.CleverTraps)
		}
	})
}

func (hunter *Hunter) applyEfficiency() {
	hunter.OnSpellRegistered(func(spell *core.Spell) {
		// applies to Stings, Shots, and Volley
		if spell.Cost != nil && spell.Flags.Matches(SpellFlagSting|SpellFlagShot) || spell.SpellCode == SpellCode_HunterVolley {
			spell.Cost.Multiplier -= 2 * hunter.Talents.Efficiency
		}
	})
}
