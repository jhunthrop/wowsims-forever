package warrior

import (
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

// Reckless Swings (Fury node 113636, tier 3 column 1, one rank; new in
// 1.60.1.70755) is a learned active ability: its spell, 1329555, is on a
// SkillLineAbility row (59905, skill line 256) and the talent teaches it.
// "Your next 4 melee attacks generate 5 additional Rage. Lasts 8 sec."
//
// The client rows, every one of which TestRecklessSwingsMatchesTheClient
// pins:
//
//   - 1329555 SpellCooldowns: RecoveryTime 15000 (RecklessSwingsCooldownMS),
//     StartRecoveryTime 1000 (a 1 s global cooldown).
//   - 1329555 SpellMisc DurationIndex 31: 8000 ms.
//   - 1329555 SpellAuraOptions: ProcCharges 4, ProcChance 100,
//     ProcTypeMask 0x4 (PROC_FLAG_DONE_MELEE_AUTO_ATTACK, the white swing
//     of either hand). With no proc entry of its own the server's default
//     hit mask is a normal or critical hit, so a swing that misses, or is
//     dodged or parried, neither grants rage nor spends a charge.
//   - 1329555 e0: aura 42 (proc trigger spell) of 1329554, whose e0 is
//     effect 30 (energize) misc 1 (rage), 50 tenths: 5 rage a swing.
//   - 1329555 SpellPower: PowerType -2, PowerCostPct 25. -2 is the
//     client's POWER_HEALTH, the same cost Bloodrage (2687, PowerCostPct
//     20) carries, and a percent health cost is a percent of the caster's
//     base health (Stamina's health excluded): 25% of base health a cast.
//
// The health cost follows Life Tap's convention (sim/warlock/lifetap.go):
// it is paid, and gates the cast (the client refuses a health cost the
// caster cannot pay without dying), only when the warrior is tanking,
// the one sim that models incoming damage and healing. A DPS sim models
// neither, so there the cost is taken as healed, as a raid heals it.
const (
	recklessSwingsCharges     = 4
	recklessSwingsRage        = 5.0
	recklessSwingsDuration    = 8 * time.Second
	recklessSwingsGCD         = time.Second
	recklessSwingsRageSpellID = 1329554
)

// recklessSwingsHealthCost is the health one cast of Reckless Swings
// costs: RecklessSwingsManaCostPct (the SpellPower row's PowerCostPct,
// whose power type is health) of the warrior's base health.
func (warrior *Warrior) recklessSwingsHealthCost() float64 {
	return warrior.GetBaseStats()[stats.Health] * RecklessSwingsManaCostPct[0] / 100
}

func (warrior *Warrior) registerRecklessSwingsSpell() {
	if !warrior.Talents.RecklessSwings {
		return
	}

	actionID := core.ActionID{SpellID: RecklessSwingsSpellId[0]}
	rageMetrics := warrior.NewRageMetrics(core.ActionID{SpellID: recklessSwingsRageSpellID})

	warrior.RecklessSwingsAura = warrior.RegisterAura(core.Aura{
		Label:     "Reckless Swings",
		ActionID:  actionID,
		Duration:  recklessSwingsDuration,
		MaxStacks: recklessSwingsCharges,
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if !result.Landed() || !spell.ProcMask.Matches(core.ProcMaskMeleeWhiteHit) {
				return
			}
			warrior.AddRage(sim, recklessSwingsRage, rageMetrics)
			aura.RemoveStack(sim)
		},
	})

	warrior.RecklessSwings = warrior.RegisterSpell(AnyStance, core.SpellConfig{
		ActionID: actionID,
		Flags:    core.SpellFlagAPL | core.SpellFlagHelpful,

		ExtraCastCondition: func(sim *core.Simulation, _ *core.Unit) bool {
			return !warrior.IsTanking() || warrior.CurrentHealth() > warrior.recklessSwingsHealthCost()
		},

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: recklessSwingsGCD,
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    warrior.NewTimer(),
				Duration: time.Duration(RecklessSwingsCooldownMS[0]) * time.Millisecond,
			},
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			if warrior.IsTanking() {
				warrior.RemoveHealth(sim, warrior.recklessSwingsHealthCost())
			}
			warrior.RecklessSwingsAura.Activate(sim)
			warrior.RecklessSwingsAura.SetStacks(sim, recklessSwingsCharges)
		},
	})
}
