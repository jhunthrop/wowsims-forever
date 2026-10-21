package rogue

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

func (rogue *Rogue) registerPremeditation() {
	if !rogue.Talents.Premeditation {
		return
	}

	actionID := core.ActionID{SpellID: 14183}
	comboMetrics := rogue.NewComboPointMetrics(actionID)

	// Premeditation's own combo points are not tracked as expiring (the
	// real ability forfeits any of these 2 combo points still unspent
	// after 20s; this engine does not model that forfeiture), but the
	// window itself is tracked as a plain aura so the conformance report
	// can see its real 20s duration_ms - see
	// priest/vampiric_embrace.go's RelatedSelfBuff comment.
	premeditationAura := rogue.RegisterAura(core.Aura{
		Label:    "Premeditation",
		ActionID: actionID,
		Duration: time.Second * 20,
	})

	rogue.Premeditation = rogue.RegisterSpell(core.SpellConfig{
		ActionID:      actionID,
		Flags:         core.SpellFlagAPL,
		RequiredLevel: 20,

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				Cost: 0,
				GCD:  0,
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    rogue.NewTimer(),
				Duration: time.Minute * 2,
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return rogue.IsStealthed()
		},

		RelatedSelfBuff: premeditationAura,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			premeditationAura.Activate(sim)
			rogue.AddComboPoints(sim, 2, target, comboMetrics)
		},
	})

	rogue.AddMajorCooldown(core.MajorCooldown{
		Spell:    rogue.Premeditation,
		Type:     core.CooldownTypeDPS,
		Priority: core.CooldownPriorityLow,
	})
}
