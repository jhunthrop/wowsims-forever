package warrior

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// TODO: Classic Update
func (warrior *Warrior) RegisterShieldWallCD() {
	duration := time.Duration(10+improvedShieldWallDuration[rankIndex(warrior.Talents.ImprovedShieldWall, improvedShieldWallDuration[:])]) * time.Second
	//This is the inverse of the tooltip since it is a damage TAKEN coefficient
	damageTaken := 0.25

	actionID := core.ActionID{SpellID: 871}
	swAura := warrior.RegisterAura(core.Aura{
		Label:    "Shield Wall",
		ActionID: actionID,
		Duration: duration,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			warrior.PseudoStats.DamageTakenMultiplier *= damageTaken
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			warrior.PseudoStats.DamageTakenMultiplier /= damageTaken
		},
	})

	// 900000ms (15 min): ShieldWallCooldownMS[0] (constants_auto_gen.go).
	// The old time.Minute*30 was vanilla's cooldown; Forever's client
	// halved it and this literal was never updated.
	cooldownDur := time.Duration(ShieldWallCooldownMS[0]) * time.Millisecond

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
				Duration: cooldownDur,
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return warrior.PseudoStats.CanBlock
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
			swAura.Activate(sim)
		},
	})

	warrior.AddMajorCooldown(core.MajorCooldown{
		Spell: swSpell.Spell,
		Type:  core.CooldownTypeSurvival,
	})
}
