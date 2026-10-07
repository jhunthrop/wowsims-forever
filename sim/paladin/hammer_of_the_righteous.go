package paladin

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Hammer of the Righteous (spell 407632) is learned at level 40 on the
// Protection skill line. The client states three effects: a Holy
// school-damage effect of 1, and two dummies, 120 and 3, which the spell
// script reads. The script is not published; this reads them as the
// ability's SoD namesake does, a strike for 120% of weapon damage on up to
// 3 targets, with the flat 1 added to each hit. That reading is
// UNCONFIRMED (the hit being the raw rather than the normalized weapon,
// the 120% being of one swing), and the cooldown and cost are the client's.
const (
	hammerOfTheRighteousActionID  = 407632
	hammerOfTheRighteousLevel     = 40
	hammerOfTheRighteousCooldown  = 6 * time.Second
	hammerOfTheRighteousCostShare = 0.06 // cost_pct 6.0: of base mana
	hammerOfTheRighteousFlat      = 1.0
	hammerOfTheRighteousWeaponPct = 1.20
	hammerOfTheRighteousTargets   = 3
)

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

		DamageMultiplier: hammerOfTheRighteousWeaponPct * paladin.getWeaponSpecializationModifier(),
		ThreatMultiplier: 1,
		ClientBaseDamage: [2]float64{hammerOfTheRighteousFlat, hammerOfTheRighteousFlat},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			struck := results[:min(hammerOfTheRighteousTargets, int(sim.GetNumTargets()))]
			for i := range struck {
				baseDamage := hammerOfTheRighteousFlat + spell.Unit.MHWeaponDamage(sim, spell.MeleeAttackPower(target))
				struck[i] = spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMeleeSpecialHitAndCrit)
				target = sim.Environment.NextTargetUnit(target)
			}
			for _, result := range struck {
				spell.DealDamage(sim, result)
			}
		},
	})
}
