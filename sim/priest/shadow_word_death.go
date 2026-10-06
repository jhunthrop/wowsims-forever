package priest

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// ShadowWordDeathRanks is Shadow Word: Death's four-rank count; source:
// 1.60.1.70009 client spellconst (priest.json spells 1309595/1309633/
// 1309635/1309636). Instant cast (cast_time_ms 0), 1500 ms GCD, and a
// 15 s cooldown shared across ranks (category_cooldown_ms 15000, spell
// CooldownMS 0 on every rank -- same "cooldown lives on the category"
// shape sim/core/spellconst.Spell.EffectiveCooldownMS documents for
// Bloodthirst).
const ShadowWordDeathRanks = 4

var ShadowWordDeathSpellId = [ShadowWordDeathRanks + 1]int32{0, 1309595, 1309633, 1309635, 1309636}
var ShadowWordDeathBaseDamage = [ShadowWordDeathRanks + 1]float64{0, 295, 370, 403, 448}
var ShadowWordDeathSpellCoef = [ShadowWordDeathRanks + 1]float64{0, .429, .429, .429, .429}
var ShadowWordDeathManaCost = [ShadowWordDeathRanks + 1]float64{0, 175, 205, 250, 340}
var ShadowWordDeathLevel = [ShadowWordDeathRanks + 1]int{0, 32, 40, 48, 56}

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
	baseDamage := ShadowWordDeathBaseDamage[rank]
	spellCoeff := ShadowWordDeathSpellCoef[rank]
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
				Timer:    cdTimer,
				Duration: time.Second * 15,
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: spellCoeff,

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

			result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)

			if critBonus != 0 {
				spell.BonusCritRating -= critBonus
			}

			// Backlash: the client's second effect (index 1, effect 77,
			// amount 150, identical across all four ranks regardless of
			// the direct hit's own scaling from 295 to 448) does not
			// decode to a known mechanic through this pipeline --
			// sim/core/spellconst carries the Effect/Aura type columns
			// verbatim and never interprets them, and nothing else in
			// the client dump gives this spell a description to check
			// the number against. Per this lane's rule for an effect
			// this package cannot read, Classic's own well-documented
			// Shadow Word: Death behaviour is kept instead: the caster
			// takes damage equal to the damage just dealt. These dummy
			// targets have no simulated health that this cast could
			// bring to zero, so the "if it doesn't kill the target"
			// condition is always true here and backlash applies
			// whenever the cast landed.
			if result.Landed() {
				priest.RemoveHealth(sim, result.Damage)
			}
		},
	}
}
