package hunter

import (
	"math"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// Hydra Shot (spell 1293020, trainable at 60): "Fires an instant Shot
// that divides multiple times to strike up to 5 targets. Your initial
// target takes weapon damage plus 260 and each subsequent target takes
// 35% less damage. Hydra Shot shares its cooldown with Arcane Shot."
// The client's effect is code 121 (normalized weapon damage, the same
// effect Aimed Shot reads) with amount 260, chain targets 5 and chain
// amplitude 0.65, on category_cooldown_ms 6000 for 250 mana. Barrage's
// text names Multi-Shot, Aimed Shot and Volley only, so it is not applied.
const (
	hydraShotMaxTargets      = 5
	hydraShotChainMultiplier = 0.65
	hydraShotRank            = HydraShotRanks
)

func (hunter *Hunter) getHydraShotConfig(timer *core.Timer) core.SpellConfig {
	flatDamage := HydraShotDamage[hydraShotRank]
	casterLevel := int(hunter.Level)

	// Pool-sized ceiling, live-bounded loop; see Multi-Shot.
	maxHits := min(hydraShotMaxTargets, len(hunter.Env.Encounter.AllTargetUnits))
	results := make([]*core.SpellResult, maxHits)

	return core.SpellConfig{
		SpellCode:     SpellCode_HunterHydraShot,
		ActionID:      core.ActionID{SpellID: HydraShotSpellId[hydraShotRank]},
		SpellSchool:   core.SpellSchoolPhysical,
		DefenseType:   core.DefenseTypeRanged,
		ProcMask:      core.ProcMaskRangedSpecial,
		Flags:         core.SpellFlagMeleeMetrics | core.SpellFlagAPL | SpellFlagShot,
		CastType:      proto.CastType_CastTypeRanged,
		Rank:          hydraShotRank,
		RequiredLevel: HydraShotLevel[hydraShotRank],
		MissileSpeed:  24,

		ManaCost: core.ManaCostOptions{
			FlatCost: HydraShotManaCost[hydraShotRank],
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			IgnoreHaste: true, // Hunter GCD is locked at 1.5s
			CD: core.Cooldown{
				Timer:    timer,
				Duration: time.Duration(HydraShotCooldownMS[hydraShotRank]) * time.Millisecond,
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return hunter.DistanceFromTarget >= core.MinRangedAttackDistance
		},

		CritDamageBonus: hunter.mortalShots(),

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: 1,
		ClientBaseDamage: flatDamage.Range(casterLevel),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := hunter.AutoAttacks.Ranged().CalculateNormalizedWeaponDamage(sim, spell.RangedAttackPower(target, false)) +
				hunter.AmmoDamageBonus +
				flatDamage.Roll(sim, casterLevel)

			// Fixed for this cast so the landing loop deals exactly the
			// results the hit loop calculated.
			numHits := min(maxHits, len(sim.Encounter.TargetUnits))
			curTarget := target
			for hitIndex := 0; hitIndex < numHits; hitIndex++ {
				chain := math.Pow(hydraShotChainMultiplier, float64(hitIndex))
				results[hitIndex] = spell.CalcDamage(sim, curTarget, baseDamage*chain, spell.OutcomeRangedHitAndCrit)
				curTarget = sim.Environment.NextTargetUnit(curTarget)
			}

			spell.WaitTravelTime(sim, func(sim *core.Simulation) {
				for hitIndex := 0; hitIndex < numHits; hitIndex++ {
					spell.DealDamage(sim, results[hitIndex])
				}
			})
		},
	}
}

// registerHydraShotSpell registers Hydra Shot on Arcane Shot's timer
// (the client's shared cooldown category) for a level-60 hunter.
func (hunter *Hunter) registerHydraShotSpell(arcaneShotTimer *core.Timer) {
	if int(hunter.Level) < HydraShotLevel[hydraShotRank] {
		return
	}
	hunter.HydraShot = hunter.GetOrRegisterSpell(hunter.getHydraShotConfig(arcaneShotTimer))
}
