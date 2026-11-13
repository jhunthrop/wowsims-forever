package paladin

import (
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// Hammer of the Righteous (spell 407632) is learned at level 40 on the
// Protection skill line. The client states three effects: a Holy
// school-damage effect of 1, a dummy of 120 the spell script reads, and a
// dummy of 3 that its text names: "causing Holy damage to each equal to
// $s3 times the damage per second of your main hand weapon" to the
// current target and up to 3 additional nearby targets (Wowhead Forever,
// read 2026-11-01; effects 0 and 2 carry EffectChainTargets 4). So the hit is 3 times the weapon's
// damage per second, with the flat 1 added to each: a 2.5 second weapon
// hits for 120% of a swing and a 1.5 second one for 200%, the figure the
// ability's Season of Discovery namesake rounds to. The 120 dummy is not
// read (the script is not published; a flat 120% of a swing would make the
// weapon's speed irrelevant, which the text contradicts). Whether "damage
// per second" counts attack power the way the character sheet's figure
// does is UNCONFIRMED; it is read as the sheet does, so the attack power
// term is 3 times AP over 14 whatever the weapon. The cooldown and cost
// are the client's.
const (
	hammerOfTheRighteousActionID    = 407632
	hammerOfTheRighteousLevel       = 40
	hammerOfTheRighteousCooldown    = 6 * time.Second
	hammerOfTheRighteousCostShare   = 0.06 // cost_pct 6.0: of base mana
	hammerOfTheRighteousFlat        = 1.0
	hammerOfTheRighteousDPSMultiple = 3
	hammerOfTheRighteousTargets     = 4
)

// hammerOfTheRighteousWeaponAllowed is the client's SpellEquippedItems row
// for 407632 (1.60.1.70888, hotfixed): item class 2, subclass mask 145, a
// one-handed axe, mace or sword in the main hand. A two-hander cannot cast
// it, so a Retribution paladin never presses it.
func hammerOfTheRighteousWeaponAllowed(weapon *core.Item) bool {
	if weapon == nil || weapon.HandType == proto.HandType_HandTypeTwoHand {
		return false
	}
	switch weapon.WeaponType {
	case proto.WeaponType_WeaponTypeAxe, proto.WeaponType_WeaponTypeMace, proto.WeaponType_WeaponTypeSword:
		return true
	}
	return false
}

// hammerOfTheRighteousSwingShare is the share of one main hand swing's
// damage that "3 times the damage per second" makes of a weapon of the
// given speed.
func hammerOfTheRighteousSwingShare(weaponSpeed float64) float64 {
	return hammerOfTheRighteousDPSMultiple / weaponSpeed
}

func (paladin *Paladin) registerHammerOfTheRighteous() {
	if paladin.Level < hammerOfTheRighteousLevel {
		return
	}

	var results [hammerOfTheRighteousTargets]*core.SpellResult

	paladin.RegisterSpell(core.SpellConfig{
		SpellCode:   SpellCode_PaladinHammerOfTheRighteous,
		ActionID:    core.ActionID{SpellID: hammerOfTheRighteousActionID},
		SpellSchool: core.SpellSchoolHoly,
		DefenseType: core.DefenseTypeMelee,
		ProcMask:    core.ProcMaskMeleeMHSpecial,
		Flags:       core.SpellFlagMeleeMetrics | core.SpellFlagAPL,

		RequiredLevel: hammerOfTheRighteousLevel,

		ManaCost: core.ManaCostOptions{
			BaseCost: hammerOfTheRighteousCostShare,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			CD: core.Cooldown{
				Timer:    paladin.NewTimer(),
				Duration: hammerOfTheRighteousCooldown,
			},
		},

		ExtraCastCondition: func(_ *core.Simulation, _ *core.Unit) bool {
			return hammerOfTheRighteousWeaponAllowed(paladin.MainHand())
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		ClientBaseDamage: [2]float64{hammerOfTheRighteousFlat, hammerOfTheRighteousFlat},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			struck := results[:min(hammerOfTheRighteousTargets, int(sim.GetNumTargets()))]
			swingShare := hammerOfTheRighteousSwingShare(spell.Unit.AutoAttacks.MH().SwingSpeed)
			for i := range struck {
				baseDamage := hammerOfTheRighteousFlat + swingShare*spell.Unit.MHWeaponDamage(sim, spell.MeleeAttackPower(target))
				struck[i] = spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMeleeSpecialHitAndCrit)
				target = sim.Environment.NextTargetUnit(target)
			}
			for _, result := range struck {
				spell.DealDamage(sim, result)
			}
		},
	})
}
