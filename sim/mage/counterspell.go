package mage

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// This exists purely so that it can be used to extend the arcane buff from the mage T1 4pc
// Not relevant in classic currently but will keep
func (mage *Mage) registerCounterspellSpell() {
	// Counterspell trains at 24 (trainables 2139). Nothing checks
	// RequiredLevel at cast time, so a mage below it must not have the spell.
	if int(mage.Level) < CounterspellLevel[0] {
		return
	}
	mage.Counterspell = mage.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: 2139},
		ClassSpellMask: MageSpellMaskCounterspell,
		SpellSchool:    core.SpellSchoolArcane,
		ProcMask:       core.ProcMaskSpellDamage,
		Flags:          core.SpellFlagAPL | SpellFlagMage | core.SpellFlagCastTimeNoGCD,
		RequiredLevel:  CounterspellLevel[0],

		ManaCost: core.ManaCostOptions{
			FlatCost: 100,
		},

		Cast: core.CastConfig{
			CD: core.Cooldown{
				Timer:    mage.NewTimer(),
				Duration: time.Second * 30,
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			// Do nothing
			// TODO: Generates a high amount of threat
		},
	})
}
