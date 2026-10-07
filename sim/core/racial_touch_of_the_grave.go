package core

import "time"

// Touch of the Grave is Undead's passive, read from the client tables
// (data/builds/1.60.1.70009/raw): "Your spells and attacks have a 10% chance
// to drain Health from the target, up to 5% of your maximum Health."
//
//   - 1260201 is the passive: SpellAuraOptions ProcChance 10, internal
//     cooldown 1000 ms, ProcTypeMask 69972.
//   - 1260198 is the drain it casts: effect 9 (health leech) for 5 of the
//     caster's maximum Health, Shadow school, no coefficient.
//
// ProcTypeMask 69972 is 0x11154: melee auto (0x4), melee ability (0x10),
// ranged auto (0x40), ranged ability (0x100), and the negative none and
// magic damage classes (0x1000, 0x10000) that the engine files under spell
// damage. The periodic-damage bit (0x40000) is absent, so ticks never
// trigger it; that is why the trigger listens to OnSpellHitDealt only.
//
// DAMAGE MODEL. The effect is a health leech with no coefficient, so the
// drain is flat: it neither crits (OutcomeAlwaysHit), nor scales with spell
// power (nothing adds bonus damage to a zero-coefficient effect), nor takes
// attacker or target damage modifiers or resistances. It deals exactly 5%
// of the undead's maximum Health. The matching heal on the caster is not
// modelled, since it does not change damage dealt.
const (
	touchOfTheGraveSpellID           int32 = 1260201
	touchOfTheGraveDrainSpellID      int32 = 1260198
	touchOfTheGraveProcChance              = 0.10
	touchOfTheGraveICD                     = time.Second
	touchOfTheGraveMaxHealthFraction       = 0.05

	touchOfTheGraveProcMask = ProcMaskMeleeOrRanged | ProcMaskSpellDamage
)

func touchOfTheGraveDamage(maxHealth float64) float64 {
	return maxHealth * touchOfTheGraveMaxHealthFraction
}

func touchOfTheGraveTrigger(drain *Spell) ProcTrigger {
	return ProcTrigger{
		Name:       "Touch of the Grave",
		ActionID:   ActionID{SpellID: touchOfTheGraveSpellID},
		Callback:   CallbackOnSpellHitDealt,
		ProcMask:   touchOfTheGraveProcMask,
		Outcome:    OutcomeLanded,
		ProcChance: touchOfTheGraveProcChance,
		ICD:        touchOfTheGraveICD,
		Handler: func(sim *Simulation, _ *Spell, result *SpellResult) {
			drain.CalcAndDealDamage(sim, result.Target, touchOfTheGraveDamage(drain.Unit.MaxHealth()), drain.OutcomeAlwaysHit)
		},
	}
}

func applyTouchOfTheGrave(character *Character) {
	drain := character.RegisterSpell(SpellConfig{
		ActionID:    ActionID{SpellID: touchOfTheGraveDrainSpellID},
		SpellSchool: SpellSchoolShadow,
		ProcMask:    ProcMaskSpellDamageProc,
		Flags: SpellFlagNoOnCastComplete | SpellFlagIgnoreModifiers | SpellFlagIgnoreResists |
			SpellFlagNoOnDamageDealt | SpellFlagSuppressWeaponProcs | SpellFlagSuppressEquipProcs,
		DamageMultiplier: 1,
		ThreatMultiplier: 1,
	})
	MakeProcTriggerAura(&character.Unit, touchOfTheGraveTrigger(drain))
}
