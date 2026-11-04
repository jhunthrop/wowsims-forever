package hunter

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// rapidFireHaste is Rapid Fire's 40% ranged and melee attack speed.
const rapidFireHaste = 1.4

func (hunter *Hunter) registerRapidFire() {
	if hunter.Level < int32(RapidFireLevel[0]) {
		return
	}

	actionID := core.ActionID{SpellID: RapidFireSpellId[0]}

	hunter.RapidFireAura = hunter.RegisterAura(core.Aura{
		Label:    "Rapid Fire",
		ActionID: actionID,
		Duration: time.Second * 15,

		// Client 3045: aura 140 (ranged haste 40) and aura 319 (melee
		// haste 40), "ranged and melee attack speed by 40%".
		// The beta logs show it on Summon Hawk's hawks too (2.26 s swings,
		// 1.62 s inside Rapid Fire).
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.MultiplyAttackSpeed(sim, rapidFireHaste)
			hunter.hasteHawks(sim, true)
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.MultiplyAttackSpeed(sim, 1/rapidFireHaste)
			hunter.hasteHawks(sim, false)
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
