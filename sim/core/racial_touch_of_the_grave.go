package core

import (
	"time"

	"github.com/wowsims/classic/sim/core/proto"
)

// Touch of the Grave is Undead's passive, read from the client tables
// (build 1.60.1.70707): "Your spells and attacks have a 20% (30%) chance to
// drain Health from the target, up to 5% of your maximum Health."
//
//   - The passive comes in two class versions (SkillLineAbility 55923 and
//     55924): 1260189 for warriors, paladins and rogues (ClassMask 11),
//     ProcChance 20, and 1260201 for priests, mages and warlocks (ClassMask
//     400), ProcChance 30. Both have ProcTypeMask 69972 and a shipped
//     25000 ms ProcCategoryRecovery; the internal cooldown is 15 s since
//     Blizzard's 27 October notes ("reduced to 15 seconds (was 25
//     seconds)"), which the 30 October hotfix cache carries. Build 1.60.1.70009 had one passive at 10% with a
//     1000 ms internal cooldown; the beta notes of 15 October 2026 give the
//     new figures ("a 30% chance for casters and 20% for Paladins, Rogues
//     and Warriors, a 25-second internal cooldown").
//   - 1260198 is the drain it casts: effect 9 (health leech) for 5 of the
//     caster's maximum Health, Shadow school, no coefficient.
//
// ProcTypeMask 69972 is 0x11154: melee auto (0x4), melee ability (0x10),
// ranged auto (0x40), ranged ability (0x100), and the negative none and
// magic damage classes (0x1000, 0x10000) that the engine files under spell
// damage. The periodic-damage bit (0x40000) is absent, so ticks never
// trigger it; that is why the trigger listens to OnSpellHitDealt only.
//
// DAMAGE MODEL. The effect is a health leech with no coefficient. It rolls
// the spell hit table (SpellCategories DefenseType 1, magic) and never crits
// (SpellMisc Attributes_2 0x20000000), so it lands as OutcomeMagicHit: it
// can miss a higher-level target, and it neither crits nor scales with spell
// power (nothing adds bonus damage to a zero-coefficient effect), nor takes
// attacker or target damage modifiers or resistances. It deals exactly 5%
// of the undead's maximum Health. The matching heal on the caster is not
// modelled, since it does not change damage dealt.
const (
	touchOfTheGraveMeleeSpellID      int32 = 1260189
	touchOfTheGraveCasterSpellID     int32 = 1260201
	touchOfTheGraveDrainSpellID      int32 = 1260198
	touchOfTheGraveMeleeProcChance         = 0.20
	touchOfTheGraveCasterProcChance        = 0.30
	touchOfTheGraveICD                     = 15 * time.Second
	touchOfTheGraveMaxHealthFraction       = 0.05

	touchOfTheGraveProcMask = ProcMaskMeleeOrRanged | ProcMaskSpellDamage
)

// touchOfTheGravePassive is the class's version of the passive: its spell
// id and proc chance. Every class outside the casters' ClassMask 400 (the
// Undead classes are warrior, paladin, rogue, priest, mage and warlock)
// takes the melee version.
func touchOfTheGravePassive(class proto.Class) (int32, float64) {
	switch class {
	case proto.Class_ClassPriest, proto.Class_ClassMage, proto.Class_ClassWarlock:
		return touchOfTheGraveCasterSpellID, touchOfTheGraveCasterProcChance
	default:
		return touchOfTheGraveMeleeSpellID, touchOfTheGraveMeleeProcChance
	}
}

func touchOfTheGraveDamage(maxHealth float64) float64 {
	return maxHealth * touchOfTheGraveMaxHealthFraction
}

func touchOfTheGraveTrigger(drain *Spell, class proto.Class) ProcTrigger {
	spellID, chance := touchOfTheGravePassive(class)
	return ProcTrigger{
		Name:       "Touch of the Grave",
		ActionID:   ActionID{SpellID: spellID},
		Callback:   CallbackOnSpellHitDealt,
		ProcMask:   touchOfTheGraveProcMask,
		Outcome:    OutcomeLanded,
		ProcChance: chance,
		ICD:        touchOfTheGraveICD,
		Handler: func(sim *Simulation, _ *Spell, result *SpellResult) {
			drain.CalcAndDealDamage(sim, result.Target, touchOfTheGraveDamage(drain.Unit.MaxHealth()), drain.OutcomeMagicHit)
		},
	}
}

func applyTouchOfTheGrave(character *Character) {
	drain := character.RegisterSpell(SpellConfig{
		ActionID:    ActionID{SpellID: touchOfTheGraveDrainSpellID},
		SpellSchool: SpellSchoolShadow,
		DefenseType: DefenseTypeMagic,
		ProcMask:    ProcMaskSpellDamageProc,
		Flags: SpellFlagNoOnCastComplete | SpellFlagIgnoreModifiers | SpellFlagIgnoreResists |
			SpellFlagNoOnDamageDealt | SpellFlagSuppressWeaponProcs | SpellFlagSuppressEquipProcs,
		DamageMultiplier: 1,
		ThreatMultiplier: 1,
	})
	MakeProcTriggerAura(&character.Unit, touchOfTheGraveTrigger(drain, character.Class))
}
