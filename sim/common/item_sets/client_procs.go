package item_sets

import (
	"fmt"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// bonusModel builds the effect of one client bonus spell from its rows.
type bonusModel func(bonusID int32) core.ApplyEffect

// unstatedProcRate is the rate of a "chance on" bonus whose chance the
// client does not state: one proc per minute, which is what the engine
// used for the melee energize and rage bonuses (Rogue Armor Energize,
// Warrior's Resolve) before the client's rows were read. It is an
// assumption, listed as such in the lane report.
var unstatedProcRate = procRate{ppm: 1}

// furiousStormVanillaSpell is the vanilla bonus whose proc The Furious
// Storm and Crusader's Wrath rework: the same trigger spell, 27775 and
// 27499 alike. The reworked bonuses state no chance, so they keep its.
const furiousStormVanillaSpell int32 = 27774

// energize restores a resource when a bonus procs: "your spellcasts have a
// 5% chance to energize you for 200 mana".
func energize(kind procKind, unstated procRate) bonusModel {
	return func(bonusID int32) core.ApplyEffect {
		bonus := core.MustClientSpellRow(bonusID)
		trigger := triggerOf(bonusID)
		rate := rateOf(bonusID, unstated)
		grant := grantOf(trigger)
		return func(agent core.Agent) {
			character := agent.GetCharacter()
			give := grant.giver(character, trigger.actionID())
			core.MakeProcTriggerAura(&character.Unit, kind.trigger(bonus.Name, core.ActionID{SpellID: bonusID}, rate,
				func(sim *core.Simulation, _ *core.Spell, _ *core.SpellResult) { give(sim) }))
		}
	}
}

var (
	manaOnSpellcast    = energize(onSpellcast, procRate{})
	manaOnAutoattack   = energize(onAutoattack, procRate{})
	manaOnRangedAttack = energize(onRangedAttack, procRate{})
	resourceOnMeleeHit = energize(onMeleeHit, unstatedProcRate)
)

// burstStats are the stats a burst's trigger spell grants, read from its row.
func burstStats(trigger triggeredSpell) stats.Stats {
	flat, ok := core.DecodeClientFlatBonus(trigger.row)
	if !ok {
		panic(fmt.Sprintf("item_sets: client spell %d (%s) is not a flat stat bonus", trigger.id, trigger.row.Name))
	}
	return flat.Stats
}

// spellPowerBurst is Crusader's Wrath and The Furious Storm as Forever
// reworked them: a chance on a melee autoattack or a spellcast to raise
// spell power for the trigger spell's duration.
var spellPowerBurst = spellPowerBurstOn(onMeleeAutoattack, onSpellcast)

// spellPowerBurstOnCast is the vanilla Furious Storm (the Tier 0.5 sets
// keep the original rows): the same burst, only on a spellcast.
var spellPowerBurstOnCast = spellPowerBurstOn(onSpellcast)

func spellPowerBurstOn(kinds ...procKind) bonusModel {
	return func(bonusID int32) core.ApplyEffect {
		bonus := core.MustClientSpellRow(bonusID)
		trigger := triggerOf(bonusID)
		flat, ok := core.DecodeClientFlatBonus(trigger.row)
		if !ok {
			panic("item_sets: the spell power burst trigger is not a flat bonus")
		}
		rate := rateOf(bonusID, rateOf(furiousStormVanillaSpell, procRate{}))
		return func(agent core.Agent) {
			character := agent.GetCharacter()
			aura := character.NewTemporaryStatsAura(trigger.row.Name, trigger.actionID(), flat.Stats, trigger.duration())
			activate := func(sim *core.Simulation, _ *core.Spell, _ *core.SpellResult) { aura.Activate(sim) }
			for _, kind := range kinds {
				core.MakeProcTriggerAura(&character.Unit,
					kind.trigger(bonus.Name, core.ActionID{SpellID: bonusID}, rate, activate))
			}
		}
	}
}

// burstWhenStruck is Deathbone Surge: a chance, when struck, to raise spell
// power for the trigger spell's duration, then wait out the bonus's
// internal cooldown. It only ever fires on a character the boss hits.
func burstWhenStruck(bonusID int32) core.ApplyEffect {
	bonus := core.MustClientSpellRow(bonusID)
	trigger := triggerOf(bonusID)
	flat, ok := core.DecodeClientFlatBonus(trigger.row)
	if !ok {
		panic("item_sets: the burst trigger is not a flat bonus")
	}
	rate := rateOf(bonusID, procRate{})
	return func(agent core.Agent) {
		character := agent.GetCharacter()
		aura := character.NewTemporaryStatsAura(trigger.row.Name, trigger.actionID(), flat.Stats, trigger.duration())
		proc := onDamageTaken.trigger(bonus.Name, core.ActionID{SpellID: bonusID}, rate,
			func(sim *core.Simulation, _ *core.Spell, _ *core.SpellResult) { aura.Activate(sim) })
		proc.ICD = icdOf(bonus)
		proc.Harmful = true
		core.MakeProcTriggerAura(&character.Unit, proc)
	}
}

// roarOfTheCrowd is The Gladiator's chance on hit to gain attack speed.
// The client makes it twice as likely while no ally is within 20 yards;
// a sim raid always has allies close, so that factor is not applied.
func roarOfTheCrowd(bonusID int32) core.ApplyEffect {
	bonus := core.MustClientSpellRow(bonusID)
	trigger := triggerOf(bonusID)
	hastePercent := auraEffectOf(trigger.id, trigger.row, clientAuraAttackSpeed).Points
	haste := 1 + hastePercent/percent
	rate := rateOf(bonusID, unstatedProcRate)
	return func(agent core.Agent) {
		character := agent.GetCharacter()
		aura := character.RegisterAura(core.Aura{
			Label:    trigger.row.Name,
			ActionID: trigger.actionID(),
			Duration: trigger.duration(),
			OnGain: func(_ *core.Aura, sim *core.Simulation) {
				character.MultiplyAttackSpeed(sim, haste)
			},
			OnExpire: func(_ *core.Aura, sim *core.Simulation) {
				character.MultiplyAttackSpeed(sim, 1/haste)
			},
		})
		core.MakeProcTriggerAura(&character.Unit, onMeleeOrRangedHit.trigger(bonus.Name, core.ActionID{SpellID: bonusID}, rate,
			func(sim *core.Simulation, _ *core.Spell, _ *core.SpellResult) { aura.Activate(sim) }))
	}
}

// bleedOnMeleeHit is a chance on a melee hit to bleed the target for the
// trigger spell's periodic damage, as Defias Leather's Devious Strike and
// Bloodmail Regalia's Bloodmail do. A bonus that states no chance procs at
// the unstated rate. Devious Strike asks for a strike from behind; a sim
// melee character always strikes from behind the boss.
func bleedOnMeleeHit(unstated procRate) bonusModel {
	return func(bonusID int32) core.ApplyEffect { return bleedOnMeleeHitEffect(bonusID, unstated) }
}

var (
	deviousStrike = bleedOnMeleeHit(procRate{})
	bloodmail     = bleedOnMeleeHit(unstatedProcRate)
)

func bleedOnMeleeHitEffect(bonusID int32, unstated procRate) core.ApplyEffect {
	bonus := core.MustClientSpellRow(bonusID)
	trigger := triggerOf(bonusID)
	periodic := auraEffectOf(trigger.id, trigger.row, clientAuraPeriodicDamage)
	period := time.Duration(periodic.PeriodMS) * time.Millisecond
	ticks := int32(time.Duration(trigger.row.DurationMS) * time.Millisecond / period)
	rate := rateOf(bonusID, unstated)
	return func(agent core.Agent) {
		character := agent.GetCharacter()
		bleed := character.RegisterSpell(core.SpellConfig{
			ActionID:         trigger.actionID(),
			SpellSchool:      core.SpellSchoolPhysical,
			DefenseType:      core.DefenseTypeMelee,
			ProcMask:         core.ProcMaskEmpty,
			Flags:            core.SpellFlagNoOnCastComplete | core.SpellFlagPassiveSpell,
			DamageMultiplier: 1,
			ThreatMultiplier: 1,

			Dot: core.DotConfig{
				Aura:          core.Aura{Label: trigger.row.Name},
				NumberOfTicks: ticks,
				TickLength:    period,
				OnSnapshot: func(_ *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
					dot.Snapshot(target, periodic.Points, isRollover)
				},
				OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
					dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
				},
			},
			ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				spell.Dot(target).Apply(sim)
			},
		})
		core.MakeProcTriggerAura(&character.Unit, onMeleeHit.trigger(bonus.Name, core.ActionID{SpellID: bonusID}, rate,
			func(sim *core.Simulation, _ *core.Spell, result *core.SpellResult) { bleed.Cast(sim, result.Target) }))
	}
}

// enragingLight is Chain of the Scarlet Crusade's chance on taking damage
// to add holy damage to melee hits, multiplied against undead.
func enragingLight(bonusID int32) core.ApplyEffect {
	bonus := core.MustClientSpellRow(bonusID)
	buff := triggerOf(bonusID)
	strike := triggerOf(buff.id)
	damage := effectOf(strike.id, strike.row, func(effect core.ClientEffect) bool { return effect.Points > 0 }).Points
	undeadMultiplier := effectOf(bonusID, bonus, func(effect core.ClientEffect) bool { return effect.Trigger != 0 }).Points
	rate := rateOf(bonusID, procRate{})
	return func(agent core.Agent) {
		character := agent.GetCharacter()
		holyStrike := procDamageSpell(character, strike.actionID(), core.SpellSchoolHoly,
			func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				amount := damage
				if target.MobType == proto.MobType_MobTypeUndead {
					amount *= undeadMultiplier
				}
				spell.CalcAndDealDamage(sim, target, amount, spell.OutcomeMagicHitAndCrit)
			})
		aura := character.RegisterAura(core.Aura{
			Label:    buff.row.Name,
			ActionID: buff.actionID(),
			Duration: buff.duration(),
			OnSpellHitDealt: func(_ *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
				if result.Landed() && spell.ProcMask.Matches(core.ProcMaskMelee) {
					holyStrike.Cast(sim, result.Target)
				}
			},
		})
		trigger := onDamageTaken.trigger(bonus.Name, core.ActionID{SpellID: bonusID}, rate,
			func(sim *core.Simulation, _ *core.Spell, _ *core.SpellResult) { aura.Activate(sim) })
		trigger.ICD = icdOf(bonus)
		trigger.Harmful = true
		core.MakeProcTriggerAura(&character.Unit, trigger)
	}
}

// The spells Wildheart Raiment's tooltip names for its three returns; the
// bonus row carries only the dummy that holds the chance.
const (
	wildheartManaSpell   int32 = 27782
	wildheartRageSpell   int32 = 27783
	wildheartEnergySpell int32 = 27784
)

// wildheartReturns is Wildheart Raiment's and Feralheart Raiment's chance
// to restore mana on a spellcast, energy over time on a melee hit and rage
// when struck. The chance is the bonus's dummy effect. The energy spell is
// a periodic energize whose row names no power type; the sim restores
// energy, the resource of the form that melee-hits.
func wildheartReturns(bonusID int32) core.ApplyEffect {
	bonus := core.MustClientSpellRow(bonusID)
	chance := auraEffectOf(bonusID, bonus, clientAuraDummy).Points / percent
	rate := procRate{chance: chance}
	actionID := core.ActionID{SpellID: bonusID}
	manaGrant := grantOf(spellOf(wildheartManaSpell))
	rageGrant := grantOf(spellOf(wildheartRageSpell))
	energy := spellOf(wildheartEnergySpell)
	energyTick := auraEffectOf(energy.id, energy.row, clientAuraPeriodicEnergize)
	energyGrant := resourceGrant{power: clientPowerEnergy, amount: energyTick.Points}
	energyPeriod := time.Duration(energyTick.PeriodMS) * time.Millisecond
	energyTicks := int32(energy.duration() / energyPeriod)
	return func(agent core.Agent) {
		character := agent.GetCharacter()
		mana := manaGrant.giver(character, actionID)
		rage := rageGrant.giver(character, actionID)
		giveEnergy := energyGrant.giver(character, actionID)

		name := bonus.Name
		core.MakeProcTriggerAura(&character.Unit, onSpellcast.trigger(name, actionID, rate,
			func(sim *core.Simulation, _ *core.Spell, _ *core.SpellResult) { mana(sim) }))
		core.MakeProcTriggerAura(&character.Unit, onMeleeAutoattack.trigger(name, actionID, rate,
			func(sim *core.Simulation, _ *core.Spell, _ *core.SpellResult) {
				core.StartPeriodicAction(sim, core.PeriodicActionOptions{
					Period:   energyPeriod,
					NumTicks: int(energyTicks),
					OnAction: giveEnergy,
				})
			}))
		struck := onDamageTaken
		struck.procMask = core.ProcMaskMelee
		core.MakeProcTriggerAura(&character.Unit, struck.trigger(name, actionID, rate,
			func(sim *core.Simulation, _ *core.Spell, _ *core.SpellResult) { rage(sim) }))
	}
}

// spellPenetration is a bonus that decreases the magical resistances of
// the wearer's spell targets, which is the engine's spell penetration.
func spellPenetration(bonusID int32) core.ApplyEffect {
	reduction := auraEffectOf(bonusID, core.MustClientSpellRow(bonusID), clientAuraModTargetResist).Points
	return func(agent core.Agent) {
		agent.GetCharacter().AddStat(stats.SpellPenetration, -reduction)
	}
}

// daggerSkill is a bonus that raises dagger weapon skill.
func daggerSkill(bonusID int32) core.ApplyEffect {
	skill := auraEffectOf(bonusID, core.MustClientSpellRow(bonusID), clientAuraSkill).Points
	return func(agent core.Agent) {
		agent.GetCharacter().PseudoStats.DaggersSkill += skill
	}
}
