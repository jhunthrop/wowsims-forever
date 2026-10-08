package item_sets

import (
	"github.com/wowsims/classic/sim/core"
)

// clientEffectSchoolDamage is the client's direct school damage effect.
const clientEffectSchoolDamage int32 = 2

// procDamageSpell registers the passive spell a damage proc casts: it
// neither triggers on-cast effects nor shows as a player cast.
func procDamageSpell(character *core.Character, actionID core.ActionID, school core.SpellSchool, effects core.ApplySpellResults) *core.Spell {
	return character.RegisterSpell(core.SpellConfig{
		ActionID:         actionID,
		SpellSchool:      school,
		DefenseType:      core.DefenseTypeMagic,
		ProcMask:         core.ProcMaskEmpty,
		Flags:            core.SpellFlagNoOnCastComplete | core.SpellFlagPassiveSpell,
		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		ApplyEffects:     effects,
	})
}

// damageOnMeleeHit is a chance on a landed melee hit to deal the trigger
// spell's direct damage in a school: Volcanic Armor's Firebolt and
// Stormshroud Armor's Lightning.
func damageOnMeleeHit(school core.SpellSchool) bonusModel {
	return func(bonusID int32) core.ApplyEffect {
		bonus := core.MustClientSpellRow(bonusID)
		trigger := triggerOf(bonusID)
		damage := effectOf(trigger.id, trigger.row, func(effect core.ClientEffect) bool { return effect.Effect == clientEffectSchoolDamage }).Points
		rate := rateOf(bonusID, procRate{})
		return func(agent core.Agent) {
			character := agent.GetCharacter()
			bolt := procDamageSpell(character, trigger.actionID(), school,
				func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
					spell.CalcAndDealDamage(sim, target, damage, spell.OutcomeMagicHitAndCrit)
				})
			core.MakeProcTriggerAura(&character.Unit, onMeleeHit.trigger(bonus.Name, core.ActionID{SpellID: bonusID}, rate,
				func(sim *core.Simulation, _ *core.Spell, result *core.SpellResult) { bolt.Cast(sim, result.Target) }))
		}
	}
}

// necropileDrain is Necropile Raiment's life leech on every harmful
// spellcast, on the bonus's internal cooldown (which stops one cast's
// several damage events from leeching twice). The client states no school
// for the leech; it is Shadow, as the engine's earlier model had it, and a
// leech cannot crit. The life it returns to the wearer is not modelled: the
// sim measures no healing taken.
func necropileDrain(bonusID int32) core.ApplyEffect {
	bonus := core.MustClientSpellRow(bonusID)
	drain := triggerOf(bonusID)
	damage := effectOf(drain.id, drain.row, func(effect core.ClientEffect) bool { return effect.Effect == clientEffectHealthLeech }).Points
	return func(agent core.Agent) {
		character := agent.GetCharacter()
		leech := procDamageSpell(character, drain.actionID(), core.SpellSchoolShadow,
			func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				spell.CalcAndDealDamage(sim, target, damage, spell.OutcomeMagicHit)
			})
		trigger := onHarmfulSpellcast.trigger(bonus.Name, core.ActionID{SpellID: bonusID}, alwaysProcs,
			func(sim *core.Simulation, _ *core.Spell, _ *core.SpellResult) {
				leech.Cast(sim, character.CurrentTarget)
			})
		trigger.ICD = icdOf(bonus)
		core.MakeProcTriggerAura(&character.Unit, trigger)
	}
}
