package rogue

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// premeditationMaxComboPointsBefore is the most combo points a rogue can hold
// and still keep both of Premeditation's 2 under the cap of 5.
const premeditationMaxComboPointsBefore = 3

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
		RelatedSelfBuff: premeditationAura,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			premeditationAura.Activate(sim)
			rogue.AddComboPoints(sim, 2, target, comboMetrics)
		},
	})

	// Forever's Premeditation needs no Stealth (SpellShapeshift row 39913
	// has ShapeshiftMask 0 where Garrote, Ambush and Cheap Shot carry the
	// Stealth bit, and Attributes_0 lacks ONLY_STEALTHED; Wowhead's Forever
	// tooltip has no "Requires Stealth" line), so it comes back in combat
	// every 2 minutes. Used on cooldown it would waste combo points past
	// the cap of 5, so the automatic use waits until both fit.
	rogue.AddMajorCooldown(core.MajorCooldown{
		Spell:    rogue.Premeditation,
		Type:     core.CooldownTypeDPS,
		Priority: core.CooldownPriorityLow,
		ShouldActivate: func(sim *core.Simulation, character *core.Character) bool {
			return rogue.ComboPoints() <= premeditationMaxComboPointsBefore
		},
	})
}
