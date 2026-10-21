package hunter

import (
	"github.com/wowsims/classic/sim/core"
)

// heartOfTheLionLevel is the level the spell is learned at: every hunter
// from the first (SpellLevels level 1, SkillLineAbility row 49612).
const heartOfTheLionLevel = 1

// heartOfTheLionBaseManaCostPercent is the client's cost for spell 409580:
// SpellPower PowerCostPct 8 of base mana, on the usual 1.5 second global
// cooldown (SpellCooldowns StartRecoveryTime 1500).
const heartOfTheLionBaseManaCostPercent = 0.08

// registerHeartOfTheLion models the hunter's own Heart of the Lion (client
// spell 409580), which lasts until cancelled and re-sends its area buff
// (409583) every 14 seconds:
//
//   - the hunter is always wearing it, so the aura is permanent from the
//     start of the build (it is registered in ApplyTalents, ahead of the
//     buff phase that applies it) and the cast is never needed in a
//     rotation; it stays registered so the spell, its cost and its global
//     cooldown can be read against the client;
//   - the area buff it sends (+10% stats, attack power) is the raid buff
//     HeartOfTheLion, which AddRaidBuffs sets for the hunter's raid. The
//     hunter and the hunter's pet therefore receive it through the same
//     path as every other raid member and a second hunter's copy adds
//     nothing.
func (hunter *Hunter) registerHeartOfTheLion() {
	aura := core.HeartOfTheLionSelfAura(&hunter.Unit)

	hunter.GetOrRegisterSpell(core.SpellConfig{
		ActionID:      core.ActionID{SpellID: core.HeartOfTheLionSpellID},
		Flags:         core.SpellFlagAPL,
		RequiredLevel: heartOfTheLionLevel,

		ManaCost: core.ManaCostOptions{
			BaseCost: heartOfTheLionBaseManaCostPercent,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
		},
		ExtraCastCondition: func(_ *core.Simulation, _ *core.Unit) bool {
			return !aura.IsActive()
		},
		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			aura.Activate(sim)
		},
	})
}
