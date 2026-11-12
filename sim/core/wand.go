package core

import (
	"time"

	"github.com/wowsims/classic/sim/core/proto"
)

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
// it never starts while another spell's GCD is running, and its cast time holds the caster
// until the shot fires. The client's Shoot (5019) is an auto-repeat spell with no
// start-recovery category, but casting any spell cancels a shot in progress (wand category
// 351), and the engine cannot cancel one. Before this rule the rotation fired Shoot in the
// same instant as every instant spell, took a free wand hit and let the next spell start
// before the GCD had run out (a 1.2 s wand ended a 1.5 s GCD 0.3 s early), which inflated
// Destruction and Affliction by about 9%.
//
// How long a shot holds the caster is the wand's own timing, read from Forever logs
// (foreverlogs.gg reports 2820, 2839 and 2847; 2026-11-12 sim-accuracy, shadow), not a
// full wand speed every time: every shot is logged as a start and a fire 0.50 s apart (the
// windup, wandWindup); after an instant the first fire sits at the global cooldown's end,
// its windup having run inside the GCD; after a hardcast or a channel the first fire sits
// a windup after the cast ends; and the shots of a run are the wand's speed apart (the
// swing timer). shootCastTime is that rule. Holding a full wand speed before every first
// shot cost one shot per run, which is most of a leveling caster's wand damage.
//
// wandDamageMultiplier folds in class talents that scale wand damage specifically (e.g.
// mage/priest Wand Specialization, 13%/25%); pass 1 for classes without such a talent.
func (character *Character) RegisterShootSpell(wandDamageMultiplier float64) *Spell {
	wand := character.GetRangedWeapon()
	if wand == nil || wand.RangedWeaponType != proto.RangedWeaponType_RangedWeaponTypeWand {
		return nil
	}

	weapon := character.WeaponFromRanged()
	swing := DurationFromSeconds(weapon.SwingSpeed)
	timing := newWandTiming(character)

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
			// The default is the wand's speed, a run's cadence; ModifyCast
			// shortens a first shot to the windup left (shootCastTime).
			DefaultCast: Cast{
				GCD:      0,
				CastTime: swing,
			},
			IgnoreHaste: true,
			ModifyCast: func(sim *Simulation, _ *Spell, cast *Cast) {
				cast.CastTime = shootCastTime(sim.CurrentTime, timing.lastSpellEnd, timing.lastShot, swing)
			},
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
			timing.lastShot = sim.CurrentTime
			baseDamage := weapon.CalculateWeaponDamage(sim, 0)
			result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)

			spell.WaitTravelTime(sim, func(sim *Simulation) {
				spell.DealDamage(sim, result)
			})
		},
	})
}

// wandWindup is the time from pressing Shoot, or from the previous shot of
// a run, to the shot firing: the 0.50 s between every logged start and fire
// pair (see RegisterShootSpell). It runs inside a global cooldown, so a shot
// pressed after an instant fires when the GCD ends.
const wandWindup = 500 * time.Millisecond

// neverWanded is a time long before any fight, the tracker's "nothing yet".
const neverWanded = -time.Hour

// wandTiming is what the wand's cast time is read from: when the caster's
// last spell ended (an instant or hardcast completing, a channel's last
// tick) and when the last shot fired.
type wandTiming struct {
	lastSpellEnd time.Duration
	lastShot     time.Duration
}

// newWandTiming tracks the character's casts for the wand and clears the
// tracker at every reset. Shoot's own completion is left out: a shot ends no
// spell, it is the run.
func newWandTiming(character *Character) *wandTiming {
	timing := &wandTiming{lastSpellEnd: neverWanded, lastShot: neverWanded}
	isShoot := func(spell *Spell) bool { return spell.ActionID.SpellID == ShootSpellID }
	MakePermanent(character.RegisterAura(Aura{
		Label: "Wand Timing",
		OnCastComplete: func(_ *Aura, sim *Simulation, spell *Spell) {
			if !isShoot(spell) {
				timing.lastSpellEnd = sim.CurrentTime
			}
		},
		OnPeriodicDamageDealt: func(_ *Aura, sim *Simulation, spell *Spell, _ *SpellResult) {
			if spell.Flags.Matches(SpellFlagChanneled) {
				timing.lastSpellEnd = sim.CurrentTime
			}
		},
		OnPeriodicHealDealt: func(_ *Aura, sim *Simulation, spell *Spell, _ *SpellResult) {
			if spell.Flags.Matches(SpellFlagChanneled) {
				timing.lastSpellEnd = sim.CurrentTime
			}
		},
	}))
	character.RegisterResetEffect(func(_ *Simulation) {
		timing.lastSpellEnd, timing.lastShot = neverWanded, neverWanded
	})
	return timing
}

// shootCastTime is how long a Shoot cast holds the caster before its shot
// fires: the windup left since the last spell ended, or the swing left since
// the last shot, whichever is longer, and never less than nothing.
func shootCastTime(now, lastSpellEnd, lastShot, swing time.Duration) time.Duration {
	windupLeft := wandWindup - (now - lastSpellEnd)
	swingLeft := lastShot + swing - now
	return max(0, windupLeft, swingLeft)
}
