package rogue

import (
	"time"

	"github.com/wowsims/classic/sim/common/clientsetbonus"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

///////////////////////////////////////////////////////////////////////////
//                            Phase 1 Item Sets - Molten Core
///////////////////////////////////////////////////////////////////////////

var ItemSetNightslayerArmor = core.NewItemSet(core.ItemSet{
	Name: "Nightslayer Armor",
	Bonuses: map[int32]core.ApplyEffect{
		// Reduces the cooldown of your Vanish ability by 30 sec.
		3: func(agent core.Agent) {
			c := agent.(RogueAgent).GetRogue()
			c.RegisterAura(core.Aura{
				Label: "Improved Vanish",
				OnInit: func(aura *core.Aura, sim *core.Simulation) {
					c.Vanish.CD.Duration -= time.Second * 30
				},
			})
		},
		// Increases your maximum Energy by 10.
		5: func(agent core.Agent) {
			c := agent.GetCharacter()
			if c.HasEnergyBar() {
				c.EnableEnergyBar(c.MaxEnergy() + 10)
			}
		},
		// Heals the rogue for 500 when Vanish is performed.
		8: func(agent core.Agent) {
			c := agent.GetCharacter()
			healthMetrics := c.NewHealthMetrics(core.ActionID{SpellID: 23582})

			core.MakePermanent(c.RegisterAura(core.Aura{
				Label: "Clean Escape",
				OnCastComplete: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell) {
					if spell.SpellCode == SpellCode_RogueVanish {
						c.GainHealth(sim, 500, healthMetrics)
					}
				},
			}))
		},
	},
})

///////////////////////////////////////////////////////////////////////////
//                            Phase 2 Item Sets - Dire Maul
///////////////////////////////////////////////////////////////////////////

///////////////////////////////////////////////////////////////////////////
//                            Phase 3 Item Sets - BWL
///////////////////////////////////////////////////////////////////////////

var ItemSetBloodfangArmor = core.NewItemSet(core.ItemSet{
	Name: "Bloodfang Armor",
	Bonuses: map[int32]core.ApplyEffect{
		// Increases the chance to apply poisons to your target by 5%.
		3: func(agent core.Agent) {
			c := agent.(RogueAgent).GetRogue()
			c.RegisterAura(core.Aura{
				Label: "Improved Poisons",
				OnInit: func(aura *core.Aura, sim *core.Simulation) {
					c.additivePoisonBonusChance += .05
				},
			})
		},
		// Improves the threat reduction of Feint by 25%.
		5: func(agent core.Agent) {
			// Feint threat reduction not currently implemented in feint.go
		},
		// Gives the Rogue a chance to inflict 283 to 317 damage on the target and heal the Rogue for 50 health every 1 sec. for 6 sec. on a melee hit.
		8: func(agent core.Agent) {
			c := agent.GetCharacter()

			bloodfangHeal := c.GetOrRegisterSpell(core.SpellConfig{
				ActionID:    core.ActionID{SpellID: 23580},
				SpellSchool: core.SpellSchoolPhysical,
				DefenseType: core.DefenseTypeMelee,
				ProcMask:    core.ProcMaskEmpty,
				Flags:       core.SpellFlagNoOnCastComplete | core.SpellFlagPassiveSpell,
				Hot: core.DotConfig{
					Aura: core.Aura{
						Label: "Bloodfang",
					},
					NumberOfTicks: 6,
					TickLength:    time.Second,
					OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, _ bool) {
						dot.SnapshotBaseDamage = 50
					},
					OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
						dot.CalcAndDealPeriodicSnapshotHealing(sim, target, dot.OutcomeTick)
					},
				},
				ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
					spell.Hot(&c.Unit).Apply(sim)
				},
			})

			procSpell := c.GetOrRegisterSpell(core.SpellConfig{
				ActionID:    core.ActionID{SpellID: 23581},
				SpellSchool: core.SpellSchoolPhysical,
				DefenseType: core.DefenseTypeMelee,
				ProcMask:    core.ProcMaskEmpty,
				Flags:       core.SpellFlagNoOnCastComplete | core.SpellFlagPassiveSpell,

				DamageMultiplier: 1,
				ThreatMultiplier: 1,

				ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
					spell.CalcAndDealDamage(sim, target, sim.Roll(283, 317), spell.OutcomeMagicCrit)
				},
			})

			core.MakeProcTriggerAura(&c.Unit, core.ProcTrigger{
				Name:              "Bloodfang",
				Callback:          core.CallbackOnSpellHitDealt,
				Outcome:           core.OutcomeLanded,
				ProcMask:          core.ProcMaskMelee,
				SpellFlagsExclude: core.SpellFlagSuppressWeaponProcs,
				PPM:               1,
				Handler: func(sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
					procSpell.Cast(sim, result.Target)
					bloodfangHeal.Cast(sim, result.Target)
				},
			})
		},
	},
})

var ItemSetMadcapsOutfit = core.NewItemSet(core.ItemSet{
	Name: "Madcap's Outfit",
	Bonuses: map[int32]core.ApplyEffect{
		// +20 Attack Power.
		2: func(agent core.Agent) {
			c := agent.(RogueAgent).GetRogue()
			c.AddStats(stats.Stats{
				stats.AttackPower:       20,
				stats.RangedAttackPower: 20,
			})
		},
		// Decreases the cooldown of Blind by 20 sec.
		3: func(agent core.Agent) {
			// Blind not implemented in sim
		},
		// Decrease the energy cost of Eviscerate and Rupture by 5.
		5: func(agent core.Agent) {
			c := agent.(RogueAgent).GetRogue()

			core.MakePermanent(c.RegisterAura(core.Aura{
				Label: "Improved Eviscerate and Rupture",
				OnInit: func(aura *core.Aura, sim *core.Simulation) {
					c.Eviscerate.Cost.FlatModifier -= 5
					c.Rupture.Cost.FlatModifier -= 5
				},
			}))
		},
	},
})

///////////////////////////////////////////////////////////////////////////
//                            Phase 4 Item Sets - AQ
///////////////////////////////////////////////////////////////////////////

// https://www.wowhead.com/classic/item-set=498/emblems-of-veiled-shadows
var ItemSetEmblemsOfVeiledShadows = core.NewItemSet(core.ItemSet{
	Name: "Emblems of Veiled Shadows",
	Bonuses: map[int32]core.ApplyEffect{
		3: func(agent core.Agent) {
			rogue := agent.(RogueAgent).GetRogue()
			rogue.RegisterAura(core.Aura{
				Label: "Emblems of Veiled Shadows 3pc",
				OnInit: func(aura *core.Aura, sim *core.Simulation) {
					rogue.SliceAndDice.Cost.FlatModifier -= 10
				},
			})
		},
	},
})

// https://www.wowhead.com/classic/item-set=497/deathdealers-embrace
var ItemSetDeathdealersEmbrace = core.NewItemSet(core.ItemSet{
	Name: "Deathdealer's Embrace",
	Bonuses: map[int32]core.ApplyEffect{
		// Reduces the cooldown of your Evasion ability by -1 min.
		3: func(agent core.Agent) {
			rogue := agent.(RogueAgent).GetRogue()
			rogue.RegisterAura(core.Aura{
				Label: "Deathdealer Evasion Bonus",
				OnInit: func(aura *core.Aura, sim *core.Simulation) {
					if rogue.Evasion != nil {
						rogue.Evasion.CD.Duration -= time.Second * 60
					}
				},
			})
		},
		// 15% increased damage to your Eviscerate ability.
		5: func(agent core.Agent) {
			rogue := agent.(RogueAgent).GetRogue()
			rogue.RegisterAura(core.Aura{
				Label: "Deathdealer Eviscerate Bonus",
				OnInit: func(aura *core.Aura, sim *core.Simulation) {
					rogue.Eviscerate.DamageMultiplierAdditive += 0.15
				},
			})
		},
	},
})

///////////////////////////////////////////////////////////////////////////
//                            Phase 5 Item Sets - Naxx
///////////////////////////////////////////////////////////////////////////

// Bonescythe Armor's bonus spells and the spells they trigger, whose rows
// state the numbers below by hand.
const (
	bonescytheHeadRushBonus      int32 = 28812
	bonescytheReducedThreatBonus int32 = 28811
	bonescytheRevealedFlawBuff   int32 = 28815
)

// bonescytheHeadRushCooldown is the internal cooldown of Head Rush.
func bonescytheHeadRushCooldown() time.Duration {
	return time.Duration(core.MustClientSpellRow(bonescytheHeadRushBonus).InternalCooldownMS) * time.Millisecond
}

// bonescytheRevealedFlawCooldown is the cooldown of the Revealed Flaw buff.
func bonescytheRevealedFlawCooldown() time.Duration {
	return time.Duration(core.MustClientSpellRow(bonescytheRevealedFlawBuff).CooldownMS) * time.Millisecond
}

// https://www.wowhead.com/classic/item-set=524/bonescythe-armor
var ItemSetBonescytheArmor = core.NewItemSet(core.ItemSet{
	Name: "Bonescythe Armor",
	Bonuses: map[int32]core.ApplyEffect{
		// Your normal melee swings have a chance to Invigorate you, healing you for 90 to 110.
		2: func(agent core.Agent) {
			c := agent.(RogueAgent).GetRogue()
			actionID := core.ActionID{SpellID: 28817}
			healthMetrics := c.NewHealthMetrics(actionID)

			core.MakeProcTriggerAura(&c.Unit, core.ProcTrigger{
				ActionID: actionID,
				Name:     "Invigorate",
				Callback: core.CallbackOnSpellHitDealt,
				Outcome:  core.OutcomeLanded,
				ProcMask: core.ProcMaskMelee,
				PPM:      1,
				Handler: func(sim *core.Simulation, spell *core.Spell, _ *core.SpellResult) {
					c.GainHealth(sim, sim.Roll(90, 110), healthMetrics)
				},
			})
		},
		// Your Backstab, Sinister Strike, and Hemorrhage critical hits cause you to regain 5 energy.
		4: func(agent core.Agent) {
			c := agent.(RogueAgent).GetRogue()
			actionID := core.ActionID{SpellID: 28813}
			energyMetrics := c.NewEnergyMetrics(actionID)

			icd := core.Cooldown{Timer: c.NewTimer(), Duration: bonescytheHeadRushCooldown()}

			c.RegisterAura(core.Aura{
				Label:    "Head Rush",
				Duration: core.NeverExpires,
				OnReset: func(aura *core.Aura, sim *core.Simulation) {
					aura.Activate(sim)
				},
				OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
					if (spell.SpellCode == SpellCode_RogueBackstab || spell.SpellCode == SpellCode_RogueSinisterStrike || spell.SpellCode == SpellCode_RogueHemorrhage) && result.DidCrit() && icd.IsReady(sim) {
						icd.Use(sim)
						c.AddEnergy(sim, 5, energyMetrics)
					}
				},
			})
		},
		// Reduces the threat from your Backstab, Sinister Strike, Hemorrhage, and Eviscerate abilities.
		6: func(agent core.Agent) {
			c := agent.(RogueAgent).GetRogue()
			threat := 1 + clientsetbonus.PercentModifier(bonescytheReducedThreatBonus, clientsetbonus.ModOpThreat)
			c.RegisterAura(core.Aura{
				Label: "Reduced Threat",
				OnInit: func(aura *core.Aura, sim *core.Simulation) {
					c.Backstab.ThreatMultiplier *= threat
					c.SinisterStrike.ThreatMultiplier *= threat
					if c.Talents.Hemorrhage {
						c.Hemorrhage.ThreatMultiplier *= threat
					}
					c.Eviscerate.ThreatMultiplier *= threat
				},
			})
		},
		// Your Eviscerate has a chance per combo point to reveal a flaw in your opponent's armor, granting a 100% critical hit chance for your next Backstab, Sinister Strike, or Hemorrhage.
		8: func(agent core.Agent) {
			c := agent.(RogueAgent).GetRogue()
			actionID := core.ActionID{SpellID: 28815}

			aura := c.RegisterAura(core.Aura{
				Label:    "Revealed Flaw",
				ActionID: actionID,
				Duration: core.NeverExpires,
				OnGain: func(aura *core.Aura, sim *core.Simulation) {
					c.Backstab.BonusCritRating += 100 * core.CritRatingPerCritChance
					c.SinisterStrike.BonusCritRating += 100 * core.CritRatingPerCritChance
					if c.Talents.Hemorrhage {
						c.Hemorrhage.BonusCritRating += 100 * core.CritRatingPerCritChance
					}
				},
				OnExpire: func(aura *core.Aura, sim *core.Simulation) {
					c.Backstab.BonusCritRating -= 100 * core.CritRatingPerCritChance
					c.SinisterStrike.BonusCritRating -= 100 * core.CritRatingPerCritChance
					if c.Talents.Hemorrhage {
						c.Hemorrhage.BonusCritRating -= 100 * core.CritRatingPerCritChance
					}
				},
				OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
					if spell.SpellCode == SpellCode_RogueBackstab || spell.SpellCode == SpellCode_RogueSinisterStrike || spell.SpellCode == SpellCode_RogueHemorrhage {
						aura.Deactivate(sim)
					}
				},
			})

			// The client puts the buff spell on a cooldown of its own.
			icd := core.Cooldown{Timer: c.NewTimer(), Duration: bonescytheRevealedFlawCooldown()}
			c.OnComboPointsSpent(func(sim *core.Simulation, spell *core.Spell, comboPoints int32) {
				if spell.SpellCode == SpellCode_RogueEviscerate && icd.IsReady(sim) {
					// Proc rate from Simonize Era sheet
					if sim.Proc(0.05*float64(comboPoints), "Revealed Flaw") {
						icd.Use(sim)
						aura.Activate(sim)
					}
				}
			})
		},
	},
})
