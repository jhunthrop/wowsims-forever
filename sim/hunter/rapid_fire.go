package hunter

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

func (hunter *Hunter) registerRapidFire() {
	if hunter.Level < int32(RapidFireLevel[0]) {
		return
	}

	actionID := core.ActionID{SpellID: RapidFireSpellId[0]}

	hunter.RapidFireAura = hunter.RegisterAura(core.Aura{
		Label:    "Rapid Fire",
		ActionID: actionID,
		Duration: time.Second * 15,

		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.MultiplyRangedSpeed(sim, 1.4)
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.MultiplyRangedSpeed(sim, 1/1.4)
		},
	})

	hunter.RapidFire = hunter.RegisterSpell(core.SpellConfig{
		ActionID:      actionID,
		RequiredLevel: RapidFireLevel[0],

		ManaCost: core.ManaCostOptions{
			FlatCost: RapidFireManaCost[0],
		},
		Cast: core.CastConfig{
			CD: core.Cooldown{
				Timer:    hunter.NewTimer(),
				Duration: time.Millisecond * time.Duration(RapidFireCooldownMS[0]),
			},
		},

		// RapidFireAura above is the real aura this cast applies to
		// its own caster; wiring it in as RelatedSelfBuff gives
		// compare.go's engineDuration the 15000ms the client's
		// duration_ms names, instead of reporting it as missing.
		RelatedSelfBuff: hunter.RapidFireAura,

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			hunter.RapidFireAura.Activate(sim)
		},
	})

	hunter.AddMajorCooldown(core.MajorCooldown{
		Spell: hunter.RapidFire,
		Type:  core.CooldownTypeDPS,
	})
}
