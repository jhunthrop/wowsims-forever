package mage

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Heating Up is the Fire tree's tier-3 talent (node 105786, one rank,
// renamed from Hot Streak in the live tree): "Non-periodic critical
// strikes with Fireball, Frostfire Bolt, Fire Blast, and Scorch reduce
// the cast time of your next Pyroblast cast within 20 sec by 25%,
// stacking up to 3 times."
//
// The buff is the client's spell 400625: duration_ms 20000, effect 0 a
// cast-time modifier of -25 (aura 108, misc 10) and effect 1 a dummy of
// -3, the stack ceiling. "Your next Pyroblast cast" is a charge-style
// buff, so a Pyroblast that completes spends every stack at once.
const (
	heatingUpBuffSpellId int32 = 400625
	heatingUpDuration          = time.Second * 20
	heatingUpMaxStacks         = 3

	// heatingUpCastTimeReductionPerStack is effect 0's -25 as the
	// fraction CastTimeMultiplier takes: three stacks leave a quarter
	// of the cast.
	heatingUpCastTimeReductionPerStack = 0.25
)

// heatingUpFeederMask is the four abilities whose non-periodic crits
// build the buff. Pyroblast itself is not one of them.
const heatingUpFeederMask = MageSpellMaskFireball | MageSpellMaskFrostfireBolt |
	MageSpellMaskFireBlast | MageSpellMaskScorch

func (mage *Mage) applyHeatingUp() {
	if !mage.Talents.HeatingUp {
		return
	}

	// Pyroblast registers from Initialize, after ApplyTalents, so it is
	// captured as it registers (see registerMissileBarrage).
	var pyroblastSpells []*core.Spell
	mage.OnSpellRegistered(func(spell *core.Spell) {
		if spell.ClassSpellMask == MageSpellMaskPyroblast {
			pyroblastSpells = append(pyroblastSpells, spell)
		}
	})

	mage.HeatingUpAura = mage.RegisterAura(core.Aura{
		Label:     "Heating Up",
		ActionID:  core.ActionID{SpellID: heatingUpBuffSpellId},
		Duration:  heatingUpDuration,
		MaxStacks: heatingUpMaxStacks,
		OnStacksChange: func(aura *core.Aura, sim *core.Simulation, oldStacks int32, newStacks int32) {
			reduction := heatingUpCastTimeReductionPerStack * float64(newStacks-oldStacks)
			for _, spell := range pyroblastSpells {
				spell.CastTimeMultiplier -= reduction
			}
		},
		OnCastComplete: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell) {
			if spell.ClassSpellMask == MageSpellMaskPyroblast {
				aura.Deactivate(sim)
			}
		},
	})

	mage.RegisterAura(core.Aura{
		Label:    "Heating Up Trigger",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		// OnSpellHitDealt is the direct hit only; dot ticks arrive through
		// the periodic callbacks, which is the "non-periodic" of the text.
		OnSpellHitDealt: func(_ *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if spell.ClassSpellMask&heatingUpFeederMask == 0 || !result.DidCrit() {
				return
			}
			mage.HeatingUpAura.Activate(sim)
			mage.HeatingUpAura.AddStack(sim)
		},
	})
}
