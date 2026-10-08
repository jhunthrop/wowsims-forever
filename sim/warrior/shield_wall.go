package warrior

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Shield Wall (client spell 871): "-60" damage taken on every school
// (aura 87, misc 127) for 12000 ms, 900000 ms cooldown, one global
// cooldown. The vanilla 75% for 10 sec and 30 minutes are gone; the
// Deep Dive's "15 min / 60% for 12 s" is the client's own row.
const (
	shieldWallDuration       = 12 * time.Second
	shieldWallDamageMultiple = 0.4
)

func (warrior *Warrior) RegisterShieldWallCD() {
	actionID := core.ActionID{SpellID: ShieldWallSpellId[0]}
	swAura := warrior.RegisterAura(core.Aura{
		Label:    "Shield Wall",
		ActionID: actionID,
		Duration: shieldWallDuration,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			warrior.PseudoStats.DamageTakenMultiplier *= shieldWallDamageMultiple
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			warrior.PseudoStats.DamageTakenMultiplier /= shieldWallDamageMultiple
		},
	})

	swSpell := warrior.RegisterSpell(DefensiveStance, core.SpellConfig{
		ActionID: actionID,

		RequiredLevel: ShieldWallLevel[0],

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				// The client gives Shield Wall the standard GCD
				// (gcd_ms 1500); it was not off the GCD.
				GCD: core.GCDDefault,
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    warrior.NewTimer(),
				Duration: warrior.shieldWallCooldown(),
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return warrior.PseudoStats.CanBlock
		},

		RelatedSelfBuff: swAura,

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
			swAura.Activate(sim)
		},
	})

	warrior.AddMajorCooldown(core.MajorCooldown{
		Spell: swSpell.Spell,
		Type:  core.CooldownTypeSurvival,
	})
}
