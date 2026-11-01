package priest

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Shadow Word: Death is an instant cast on a 1500 ms GCD with a 15 s
// cooldown shared across ranks (category_cooldown_ms 15000, spell
// CooldownMS 0 on every rank -- the shape
// sim/core/spellconst.Spell.EffectiveCooldownMS documents for
// Bloodthirst). Its ranks, levels, costs and damage are the client's,
// from constants_auto_gen.go.

// earlyDemiseCritBonusPct is Early Demise's rank -> bonus crit chance, in
// percentage points, against a target at or below 20% health. Source:
// talents/priest.json node 110854: "Increases Shadow Word: Death's
// critical strike chance on targets at or below 20% health by 15%" /
// "...by 30%" for ranks 1 and 2.
var earlyDemiseCritBonusPct = [3]float64{0, 15, 30}

// earlyDemiseCritBonus is the crit rating bonus Early Demise grants
// Shadow Word: Death against a target at/below 20% health, at this
// instant -- zero when the talent isn't taken or the target isn't (per
// this harness's execute-phase approximation) in range of it.
func (priest *Priest) EarlyDemiseCritBonus(sim *core.Simulation) float64 {
	if priest.Talents.EarlyDemise == 0 || !sim.IsExecutePhase20() {
		return 0
	}
	return earlyDemiseCritBonusPct[priest.Talents.EarlyDemise] * core.CritRatingPerCritChance
}

// shadowWordDeathBacklashPct is the share of the caster's maximum health
// a landed Shadow Word: Death costs (Forever tooltip, 1309636).
const shadowWordDeathBacklashPct = 0.30

func (priest *Priest) registerShadowWordDeathSpell() {
	priest.ShadowWordDeath = make([]*core.Spell, ShadowWordDeathRanks+1)
	cdTimer := priest.NewTimer()

	for rank := 1; rank <= ShadowWordDeathRanks; rank++ {
		config := priest.getShadowWordDeathConfig(rank, cdTimer)

		if config.RequiredLevel <= int(priest.Level) {
			priest.ShadowWordDeath[rank] = priest.GetOrRegisterSpell(config)
		}
	}
}

func (priest *Priest) getShadowWordDeathConfig(rank int, cdTimer *core.Timer) core.SpellConfig {
	spellId := ShadowWordDeathSpellId[rank]
	roll := priest.clientRoll(ShadowWordDeathBaseDamage[rank], ShadowWordDeathPointsPerLevel[rank], ShadowWordDeathLevel[rank], ShadowWordDeathMaxLevel[rank])
	spellCoeff := ShadowWordDeathSpellCoeff[rank]
	manaCost := ShadowWordDeathManaCost[rank]
	level := ShadowWordDeathLevel[rank]

	return core.SpellConfig{
		SpellCode:      SpellCode_PriestShadowWordDeath,
		ActionID:       core.ActionID{SpellID: spellId},
		SpellSchool:    core.SpellSchoolShadow,
		DefenseType:    core.DefenseTypeMagic,
		ProcMask:       core.ProcMaskSpellDamage,
		Flags:          SpellFlagPriest | core.SpellFlagAPL,
		ClassSpellMask: PriestSpellMaskShadowWordDeath,

		RequiredLevel: level,
		Rank:          rank,

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			CD: core.Cooldown{
				Timer: cdTimer,
				// 18 sec since client 1.60.1.70425 (15 before).
				Duration: time.Second * 18,
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: spellCoeff,
		ClientBaseDamage: roll,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			// Early Demise: bonus crit vs a target at/below 20% health.
			// This harness's dummy targets don't carry a simulated
			// health total the way a raid boss's does, so "at/below 20%
			// health" is read the same way Warrior Execute reads its
			// own 20% threshold (sim/warrior/execute.go): the
			// simulation-wide execute-phase window, sim.IsExecutePhase20.
			// The bonus is added to the spell for this cast only and
			// removed immediately after CalcAndDealDamage resolves
			// (synchronous, so no other cast can observe it in between).
			critBonus := priest.EarlyDemiseCritBonus(sim)
			if critBonus != 0 {
				spell.BonusCritRating += critBonus
			}

			result := spell.CalcAndDealDamage(sim, target, sim.Roll(roll[0], roll[1]), spell.OutcomeMagicHitAndCrit)

			if critBonus != 0 {
				spell.BonusCritRating -= critBonus
			}

			if result.Landed() {
				priest.AddShadowWeavingStack(sim, target)
				// Backlash: Forever's tooltip (Wowhead Forever, spell
				// 1309636) reads "If your target is not killed by Shadow
				// Word: Death, you take backlash damage equal to 30% of
				// your maximum health." These dummy targets never die to
				// one cast, so a landed cast always costs it.
				priest.RemoveHealth(sim, priest.MaxHealth()*shadowWordDeathBacklashPct)
			}
		},
	}
}
