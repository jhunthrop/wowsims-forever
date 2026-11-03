package core

import "github.com/wowsims/classic/sim/core/proto"

// ShootSpellID is the universal "Shoot" action every class learns once it equips a wand
// (spell id 5019 in the client: data/builds/1.60.1.70009/raw/Spell.csv in the site repo has
// its description as "Attack with an equipped wand."). It is not a per-class spell, so it
// never appears in any class's spellconst dump.
const ShootSpellID = 5019

// RegisterShootSpell registers the wand "Shoot" action for a character with a wand equipped
// in the ranged slot, and returns nil if there is none - callers can invoke this
// unconditionally from Initialize() without an extra equipped-item check.
//
// This engine already builds a ranged Weapon from the equipped item for every character
// (EnableAutoAttacks, WeaponFromRanged) and even has a Weapon.GetSpellSchool() fallback for
// it, but no caster ever turned that into a castable action: EnableAutoAttacks is only ever
// called with an AutoAttackOptions.Ranged weapon by classes that actually auto-fire it
// (hunter, and rogue for thrown weapons/guns), so a level 60 shadow priest with a wand in the
// ranged slot dealt zero wand damage. This registers Shoot as its own hardcast instead of
// folding it into AutoAttacks.ranged, because Blizzard's Wand Shoot rolls against the
// caster's own spell hit/crit tables - it cannot be blocked, parried or dodged, unlike a bow
// or gun sitting in the same slot (DefenseTypeMagic / OutcomeMagicHitAndCrit, not
// DefenseTypeRanged / OutcomeRangedHitAndCrit).
//
// Shoot costs no mana and triggers no GCD of its own (Cast.GCD: 0), but it is a real cast:
// it never starts while another spell's GCD is running, and its cast time (the wand's
// speed) holds the caster for that whole time. The client's Shoot (5019) is an auto-repeat
// spell with no start-recovery category, but casting any spell cancels a shot in progress
// (wand category 351), and the engine cannot cancel one. Before this rule the rotation
// fired Shoot in the same instant as every instant spell, took a free wand hit and let the
// next spell start before the GCD had run out (a 1.2 s wand ended a 1.5 s GCD 0.3 s early),
// which inflated Destruction and Affliction by about 9%.
//
// wandDamageMultiplier folds in class talents that scale wand damage specifically (e.g.
// mage/priest Wand Specialization, 13%/25%); pass 1 for classes without such a talent.
func (character *Character) RegisterShootSpell(wandDamageMultiplier float64) *Spell {
	wand := character.GetRangedWeapon()
	if wand == nil || wand.RangedWeaponType != proto.RangedWeaponType_RangedWeaponTypeWand {
		return nil
	}

	weapon := character.WeaponFromRanged()

	return character.GetOrRegisterSpell(SpellConfig{
		ActionID: ActionID{SpellID: ShootSpellID},
		// The wand's own school (Torch of Light: Arcane). A magic school is
		// never reduced by armor; an item that states none stays physical.
		SpellSchool: weapon.GetSpellSchool(),
		DefenseType: DefenseTypeMagic,
		ProcMask:    ProcMaskRangedAuto,
		Flags:       SpellFlagAPL | SpellFlagMeleeMetrics | SpellFlagCastTimeNoGCD,
		// setupAttackTables only ever builds a CastTypeMainHand AttackTable entry for
		// mage/priest/warlock (environment.go): before Shoot, none of the three ever attacked
		// out of the ranged slot, so a CastTypeRanged table for them would have been dead
		// weight. Using CastTypeRanged here would index a table that plainly does not exist
		// for these classes and nil-deref in AttackerDamageMultiplier the first time Shoot is
		// cast; DefenseTypeMagic/OutcomeMagicHitAndCrit already gets the "no block/parry"
		// behavior Shoot needs regardless of which CastType bucket it reads from.
		CastType: proto.CastType_CastTypeMainHand,
		// Shoot's (5019) client SpellMisc Speed.
		MissileSpeed: 20,

		Cast: CastConfig{
			DefaultCast: Cast{
				GCD:      0,
				CastTime: DurationFromSeconds(weapon.SwingSpeed),
			},
			IgnoreHaste: true,
		},
		ExtraCastCondition: func(sim *Simulation, _ *Unit) bool {
			return character.GCD.IsReady(sim)
		},

		DamageMultiplier: wandDamageMultiplier,
		ThreatMultiplier: 1,
		// A physical wand picks up flat "bonus physical damage" the way an auto attack
		// does (see EnableAutoAttacks); a magic one takes no spell damage. Wand damage
		// never scales off attack power or spell power.
		BonusCoefficient: TernaryFloat64(weapon.GetSpellSchool() == SpellSchoolPhysical, 1, 0),

		ApplyEffects: func(sim *Simulation, target *Unit, spell *Spell) {
			baseDamage := weapon.CalculateWeaponDamage(sim, 0)
			result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)

			spell.WaitTravelTime(sim, func(sim *Simulation) {
				spell.DealDamage(sim, result)
			})
		},
	})
}
