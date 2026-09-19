package mage

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Cold Snap. The filename says "baseline" because an earlier draft of
// the plan called this a new baseline ability; the client disagrees and
// the client wins - Cold Snap is the Frost tree's tier-4 column-1 node
// (105766), a talent, and the file is left under its original name so
// the plan's file list still resolves.
//
// The cooldown is the client's, from constants_auto_gen.go, not a
// literal: ColdSnapCooldownMS is the one number this ability has.

func (mage *Mage) registerColdSnapSpell() {
	if !mage.Talents.ColdSnap {
		return
	}

	// "Finishes the remaining cooldown on all your other Frost spells."
	// Collected through OnSpellRegistered because ApplyTalents runs
	// before Initialize, so the Frost spells do not exist yet.
	var affectedSpells []*core.Spell
	mage.OnSpellRegistered(func(spell *core.Spell) {
		if spell.SpellSchool.Matches(core.SpellSchoolFrost) && spell.CD.Duration > 0 {
			affectedSpells = append(affectedSpells, spell)
		}
	})

	mage.ColdSnap = mage.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: ColdSnapSpellId[0]},
		ClassSpellMask: MageSpellMaskColdSnap,
		Flags:          core.SpellFlagNoOnCastComplete,

		Cast: core.CastConfig{
			CD: core.Cooldown{
				Timer:    mage.NewTimer(),
				Duration: time.Millisecond * time.Duration(ColdSnapCooldownMS[0]),
			},
		},
		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			for _, spell := range affectedSpells {
				spell.CD.Reset()
			}
		},
	})

	mage.AddMajorCooldown(core.MajorCooldown{
		Spell: mage.ColdSnap,
		Type:  core.CooldownTypeDPS,
	})
}
