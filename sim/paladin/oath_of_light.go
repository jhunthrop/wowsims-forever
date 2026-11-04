package paladin

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Oath of Light is the Holy tier-4 talent's spell (1328548), which the
// 70707 hotfix push adds: "Swear an Oath to protect target ally for 5 min,
// increasing the healing of your Holy Light on them by up to $s1, your
// Flash of Light by up to $s2, and your Holy Shock by up to $s3. Only one
// Oath of Light may be sworn by the Paladin at any one time." Its effects
// carry no base points and scale with the caster's level alone (effect 0
// 6.6667, effect 1 1.9167, effect 2 3.8333 a level; no SpellLevels row),
// 400, 115 and 230 at 60: Blessing of Light's two amounts on one ally, with
// Holy Shock added. It costs 8% of base mana.
//
// The amount lands whole on a rank near the caster's level, with no
// cast-time coefficient, and a rank learned far below the caster keeps
// only the share its healing power keeps (core.SpellLevelPenalty, applied
// by the heal that adds the bonus). That is how Blizzard's server treated
// Blessing of Light, the Oath's model (dummy auras by script, the same
// "by up to" text): in Classic Era Flash of Light rank 1 took the whole
// +60 and Holy Light rank 1 took 60 of its 210, the level-1 rank's sub-20
// share (us.forums.blizzard.com/en/wow/t/blessing-of-light-bug/297989,
// September 2019); warcraft.wiki.gg/wiki/Blessing_of_Light has the whole
// amount and patch 2.3.0's "lower ranks ... are now properly penalized".
// Decided in sim-accuracy round 20 (design/reviews/2026-11-04-sim-audit-
// round20/paladin.md); no Forever log at 25+ with the Oath exists yet.
// Effect 0's 1.5 s periodic dummy has no stated job and is not modelled.
const (
	oathOfLightSpellID        = 1328548
	oathOfLightDuration       = 5 * time.Minute
	oathOfLightBaseManaCost   = 0.08
	oathOfLightHolyLightLevel = 6.6666698455811
	oathOfLightFlashLevel     = 1.9166699647903
	oathOfLightHolyShockLevel = 3.8333299160004
)

// oathOfLightState is the ally the paladin has sworn, if any.
type oathOfLightState struct {
	aura   *core.Aura
	target *core.Unit
}

func (paladin *Paladin) registerOathOfLight() {
	if !paladin.Talents.OathOfLight || !paladin.isHealer() {
		return
	}
	paladin.oathOfLight.aura = paladin.RegisterAura(core.Aura{
		Label:    "Oath of Light",
		ActionID: core.ActionID{SpellID: oathOfLightSpellID},
		Duration: oathOfLightDuration,
		OnExpire: func(*core.Aura, *core.Simulation) {
			paladin.oathOfLight.target = nil
		},
	})
	paladin.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: oathOfLightSpellID},
		SpellSchool: core.SpellSchoolHoly,
		ProcMask:    core.ProcMaskEmpty,
		Flags:       core.SpellFlagHelpful | core.SpellFlagAPL,

		ManaCost: core.ManaCostOptions{BaseCost: oathOfLightBaseManaCost},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{GCD: core.GCDDefault},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, _ *core.Spell) {
			// One Oath at a time: swearing a new ally ends the old one.
			paladin.oathOfLight.aura.Deactivate(sim)
			paladin.oathOfLight.target = target
			paladin.oathOfLight.aura.Activate(sim)
		},
	})
}

// isSworn is whether target carries the paladin's Oath of Light.
func (paladin *Paladin) isSworn(target *core.Unit) bool {
	return paladin.oathOfLight.aura != nil && paladin.oathOfLight.aura.IsActive() && paladin.oathOfLight.target == target
}

// oathOfLightBonus is the bonus healing the Oath adds to a heal of the
// given class mask on target, before the heal's coefficient; zero for an
// unsworn target or a heal the Oath does not name.
func (paladin *Paladin) oathOfLightBonus(target *core.Unit, classMask uint64) float64 {
	if !paladin.isSworn(target) {
		return 0
	}
	level := float64(paladin.Level)
	switch {
	case classMask&PaladinSpellMaskHolyLight != 0:
		return oathOfLightHolyLightLevel * level
	case classMask&PaladinSpellMaskFlashOfLight != 0:
		return oathOfLightFlashLevel * level
	case classMask&PaladinSpellMaskHolyShockHeal != 0:
		return oathOfLightHolyShockLevel * level
	}
	return 0
}
