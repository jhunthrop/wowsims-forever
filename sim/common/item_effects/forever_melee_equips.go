package item_effects

// Forever melee equip effects the engine did not implement (sim audit
// round 21, items lane). Every amount, school, defense type and trigger is
// the 70790 client's own row: ItemXItemEffect -> ItemEffect for the equip
// spell, SpellAuraOptions for its trigger (ProcTypeMask 4 is a landed auto
// attack, 16 a landed melee special), SpellEffect for the amount
// (EffectBasePointsF as the centre, Variance as the whole width of the
// roll), SpellCategories.DefenseType for the table it rolls and SpellMisc
// for its school and attributes. Each matches Wowhead Forever's tooltip
// (nether.wowhead.com/forever/tooltip/spell/<id>, read 2026-11-05).

import (
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

const (
	BladeOfHanna         = 2801
	StormGauntlets       = 12632
	WarbladeOfCaerDarrow = 13982
)

const (
	// improvingFormStacks is 1325509's aura 542 threshold (misc 30).
	improvingFormStacks = 30
	// perfectedFormDamage and perfectedFormStats are 1325508's effects 0
	// (Effect 2, 490) and 1 (aura 29 on every stat, 20).
	perfectedFormDamage = 490
	perfectedFormStats  = 20
	// perfectedFormDuration is 1325508's duration index 18.
	perfectedFormDuration = 20 * time.Second
)

// onEveryLandedMeleeHit casts spell at the target of every landed melee hit
// whose proc mask matches procMask: a ProcChance 100 equip on ProcTypeMask 20.
func onEveryLandedMeleeHit(character *core.Character, name string, procMask core.ProcMask, spell *core.Spell) {
	core.MakeProcTriggerAura(&character.Unit, core.ProcTrigger{
		Name:              name,
		Callback:          core.CallbackOnSpellHitDealt,
		Outcome:           core.OutcomeLanded,
		ProcMask:          procMask,
		SpellFlagsExclude: core.SpellFlagSuppressEquipProcs,
		Handler: func(sim *core.Simulation, _ *core.Spell, result *core.SpellResult) {
			spell.Cast(sim, result.Target)
		},
	})
}

func init() {
	// 13982: ItemEffect 234705 is 1298500, SpellAuraOptions 246336
	// (ProcChance 100, ProcTypeMask 20), triggering 1298499: Frost, base
	// 27.72 with variance 0.4, "Melee attacks with this weapon deal 22 to 33
	// Frost damage." Its DefenseType is 2 (melee) and its Attributes_3
	// carry ALWAYS_HIT and suppress both sides' procs: it never misses and
	// crits on the melee table.
	core.NewItemEffect(WarbladeOfCaerDarrow, func(agent core.Agent) {
		character := agent.GetCharacter()
		frost := character.RegisterSpell(core.SpellConfig{
			ActionID:         core.ActionID{SpellID: 1298499},
			SpellSchool:      core.SpellSchoolFrost,
			DefenseType:      core.DefenseTypeMelee,
			ProcMask:         core.ProcMaskEmpty,
			Flags:            core.SpellFlagNoOnCastComplete | core.SpellFlagPassiveSpell,
			DamageMultiplier: 1,
			ThreatMultiplier: 1,
			ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				spell.CalcAndDealDamage(sim, target, clientRoll(sim, 27.71999931335, 0.40000000596), spell.OutcomeMeleeSpecialCritOnly)
			},
		})
		onEveryLandedMeleeHit(character, "Warblade of Caer Darrow", character.GetProcMaskForItem(WarbladeOfCaerDarrow), frost)
	})

	// 12632: ItemEffect 99469 is 16615, SpellAuraOptions 128107 (ProcChance
	// 100, ProcTypeMask 20), triggering 16614 Lightning Strike: Nature, 3,
	// DefenseType 1 (magic), "Adds 3 Lightning damage to your melee attacks."
	core.NewItemEffect(StormGauntlets, func(agent core.Agent) {
		character := agent.GetCharacter()
		lightning := character.RegisterSpell(core.SpellConfig{
			ActionID:         core.ActionID{SpellID: 16614},
			SpellSchool:      core.SpellSchoolNature,
			DefenseType:      core.DefenseTypeMagic,
			ProcMask:         core.ProcMaskEmpty,
			Flags:            core.SpellFlagNoOnCastComplete | core.SpellFlagPassiveSpell,
			DamageMultiplier: 1,
			ThreatMultiplier: 1,
			ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				spell.CalcAndDealDamage(sim, target, 3, spell.OutcomeMagicHitAndCrit)
			},
		})
		onEveryLandedMeleeHit(character, "Storm Gauntlets", core.ProcMaskMelee, lightning)
	})

	// 2801: ItemEffect 238936 is 1325535 Improving Form, SpellAuraOptions
	// 248750 (ProcChance 101, ProcTypeMask 4: auto attacks only), adding a
	// stack of 1325509. At 30 stacks 1325509 casts 1325508 Perfected Form:
	// 490 Physical on the melee table (DefenseType 2) and 20 to all stats
	// for 20 sec, and its effect 164 clears the stacks. "Each melee hit
	// grants you a stack of Improving Form. At 30 stacks perfect your form,
	// inflicting 490 Physical damage to your current target and increasing
	// all your stats by 20 for 20 sec."
	core.NewItemEffect(BladeOfHanna, func(agent core.Agent) {
		character := agent.GetCharacter()
		form := character.NewTemporaryStatsAura("Perfected Form", core.ActionID{SpellID: 1325508}, stats.Stats{
			stats.Strength:  perfectedFormStats,
			stats.Agility:   perfectedFormStats,
			stats.Stamina:   perfectedFormStats,
			stats.Intellect: perfectedFormStats,
			stats.Spirit:    perfectedFormStats,
		}, perfectedFormDuration)
		strike := character.RegisterSpell(core.SpellConfig{
			ActionID:         core.ActionID{SpellID: 1325508},
			SpellSchool:      core.SpellSchoolPhysical,
			DefenseType:      core.DefenseTypeMelee,
			ProcMask:         core.ProcMaskEmpty,
			Flags:            core.SpellFlagNoOnCastComplete | core.SpellFlagPassiveSpell,
			DamageMultiplier: 1,
			ThreatMultiplier: 1,
			ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				spell.CalcAndDealDamage(sim, target, perfectedFormDamage, spell.OutcomeMeleeSpecialHitAndCrit)
				form.Activate(sim)
			},
		})
		stacks := character.GetOrRegisterAura(core.Aura{
			Label:     "Improving Form",
			ActionID:  core.ActionID{SpellID: 1325509},
			Duration:  core.NeverExpires,
			MaxStacks: improvingFormStacks,
		})
		core.MakeProcTriggerAura(&character.Unit, core.ProcTrigger{
			Name:              "Blade of Hanna",
			Callback:          core.CallbackOnSpellHitDealt,
			Outcome:           core.OutcomeLanded,
			ProcMask:          core.ProcMaskMeleeWhiteHit,
			SpellFlagsExclude: core.SpellFlagSuppressEquipProcs,
			Handler: func(sim *core.Simulation, _ *core.Spell, result *core.SpellResult) {
				stacks.Activate(sim)
				stacks.AddStack(sim)
				if stacks.GetStacks() < improvingFormStacks {
					return
				}
				stacks.Deactivate(sim)
				strike.Cast(sim, result.Target)
			},
		})
	})
}
