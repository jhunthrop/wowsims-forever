package warlock

import (
	"slices"
	"time"

	"github.com/wowsims/classic/sim/core"
)

func (warlock *Warlock) registerFelDominationCD() {
	if !warlock.Talents.FelDomination {
		return
	}

	actionID := core.ActionID{SpellID: 18708}

	aura := warlock.RegisterAura(core.Aura{
		ActionID: actionID,
		Label:    "Fel Domination",
		Duration: time.Second * 15,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			for _, spell := range warlock.SummonDemonSpells {
				spell.DefaultCast.CastTime -= time.Millisecond * 5500
				spell.Cost.Multiplier -= 50
			}
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			for _, spell := range warlock.SummonDemonSpells {
				spell.DefaultCast.CastTime += time.Millisecond * 5500
				spell.Cost.Multiplier += 50
			}
		},
		OnCastComplete: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell) {
			if slices.Contains(warlock.SummonDemonSpells, spell) {
				aura.Deactivate(sim)
			}
		},
	})

	spell := warlock.RegisterSpell(core.SpellConfig{
		ActionID:        actionID,
		SpellSchool:     core.SpellSchoolShadow,
		ProcMask:        core.ProcMaskEmpty,
		RelatedSelfBuff: aura,

		Cast: core.CastConfig{
			CD: core.Cooldown{
				Timer: warlock.NewTimer(),
				// The client's cooldown_ms is 300000 (5 min,
				// spellconst/warlock.json build 1.60.1.70009), not 15 -
				// conformance golden
				// sim/core/testdata/conformance/warlock.golden.md
				// flagged this as cooldown_ms 300000->900000
				// (client->engine).
				Duration: time.Minute * 5,
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			aura.Activate(sim)
		},
	})

	warlock.AddMajorCooldown(core.MajorCooldown{
		Spell: spell,
		Type:  core.CooldownTypeUnknown,
	})
}
