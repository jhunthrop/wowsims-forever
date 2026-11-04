package druid

import (
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

// Demoralizing Roar (build 1.60.1.70009): 10 rage (cost 100, every rank) on
// the global cooldown, for 30 seconds "reduces the melee attack power of
// enemies by N". The client's aura 99 amount is negative and grows with the
// caster's level above the rank's own, up to the rank's maximum level: rank
// 5 (9898, learned at 52) is -193 and -1.4 a level up to level 62, so a
// level 60 druid takes 204.2 from the target's attack power.
//
// The threat is the client's effect 63 on each rank: 8, 16, 26, 34 and 42
// at the rank's level, plus 0.8 a level to the rank's max level, on every
// target the roar lands on (demoralizingRoarThreat).
const (
	DemoralizingRoarRageCost = 10.0
	demoralizingRoarDuration = 30 * time.Second
)

// DemoralizingRoarReduction is aura 99's amount per rank (the sign flipped,
// so it reads as the attack power taken away), with the per-level growth and
// cap the generated DemoralizingRoar tables carry.
var DemoralizingRoarReduction = demoralizingRoarEffects()

func demoralizingRoarEffects() [DemoralizingRoarRanks + 1]clientdamage.Effect {
	amounts := [DemoralizingRoarRanks + 1]float64{0, 46, 74, 95, 144, 193}
	var effects [DemoralizingRoarRanks + 1]clientdamage.Effect
	for rank := 1; rank <= DemoralizingRoarRanks; rank++ {
		effects[rank] = clientdamage.Effect{
			Amount:     amounts[rank],
			PerLevel:   -DemoralizingRoarPointsPerLevel[rank],
			SpellLevel: DemoralizingRoarLevel[rank],
			MaxLevel:   DemoralizingRoarMaxLevel[rank],
		}
	}
	return effects
}

// demoralizingRoarBaseThreat and demoralizingRoarThreatPerLevel are each
// rank's effect 63 (spellconst/druid.json 99, 1735, 9490, 9747, 9898).
var demoralizingRoarBaseThreat = [DemoralizingRoarRanks + 1]float64{0, 8, 16, 26, 34, 42}

const demoralizingRoarThreatPerLevel = 0.8

func demoralizingRoarThreat(rank int, level int32) float64 {
	levelsAbove := max(0, min(int(level), DemoralizingRoarMaxLevel[rank])-DemoralizingRoarLevel[rank])
	return demoralizingRoarBaseThreat[rank] + demoralizingRoarThreatPerLevel*float64(levelsAbove)
}

func (druid *Druid) registerDemoralizingRoarSpell() {
	rank := core.HighestRankAtLevel(DemoralizingRoarLevel[1:], druid.Level)
	if rank == 0 {
		return
	}
	spellID := DemoralizingRoarSpellId[rank]
	reduction := DemoralizingRoarReduction[rank].Center(int(druid.Level))

	druid.DemoralizingRoarAuras = druid.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
		return core.DemoralizingRoarAuraOfStrength(target, spellID, reduction)
	})

	druid.DemoralizingRoar = druid.RegisterSpell(Bear, core.SpellConfig{
		ActionID:    core.ActionID{SpellID: spellID},
		SpellSchool: core.SpellSchoolPhysical,
		ProcMask:    core.ProcMaskEmpty,
		Flags:       SpellFlagOmen | core.SpellFlagAPL,

		Rank:          rank,
		RequiredLevel: DemoralizingRoarLevel[rank],

		RageCost: core.RageCostOptions{
			Cost: DemoralizingRoarRageCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			IgnoreHaste: true,
		},

		ThreatMultiplier: 1,
		FlatThreatBonus:  demoralizingRoarThreat(rank, druid.Level),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			for _, aoeTarget := range sim.Encounter.TargetUnits {
				result := spell.CalcAndDealOutcome(sim, aoeTarget, spell.OutcomeMagicHit)
				if result.Landed() {
					druid.DemoralizingRoarAuras.Get(aoeTarget).Activate(sim)
				}
			}
		},

		RelatedAuras: []core.AuraArray{druid.DemoralizingRoarAuras},
	})
}
