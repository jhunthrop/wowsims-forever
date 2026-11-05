package druid

import (
	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

const SwipeRanks = 5

// Swipe's rank tables: ids, learn levels and the school-damage amount of
// spellconst/druid.json (1.60.1.70009), which states no spread and no growth
// per level.
var SwipeSpellId = [SwipeRanks + 1]int32{0, 779, 780, 769, 9754, 9908}
var SwipeBaseDamage = [SwipeRanks + 1]float64{0, 18, 25, 36, 60, 83}
var SwipeLevel = [SwipeRanks + 1]int{0, 16, 24, 34, 44, 54}

// SwipeDamage is SwipeBaseDamage as the Effect per rank the conformance
// report and the spellconst test compare.
var SwipeDamage = swipeDamageEffects()

func swipeDamageEffects() [SwipeRanks + 1]clientdamage.Effect {
	var effects [SwipeRanks + 1]clientdamage.Effect
	for rank := 1; rank <= SwipeRanks; rank++ {
		effects[rank] = clientdamage.Effect{Amount: SwipeBaseDamage[rank], SpellLevel: SwipeLevel[rank]}
	}
	return effects
}

const (
	// swipeRageCost is the client's 200 (tenths of a point) on every rank.
	swipeRageCost = 20.0
	// swipeMaxTargets: Swipe strikes up to three enemies in front of the
	// bear.
	swipeMaxTargets = 3
	// swipeAttackPowerShare: "Fixed a bug causing Swipe to not scale with
	// Attack Power. It will now correctly gain 3% of the Druid's attack
	// power." (Blizzard's Forever patch notes, 1 October 2026). The client
	// row's BonusCoefficientFromAP is 0 on every rank; the server adds it.
	swipeAttackPowerShare = 0.03

	// SwipeThreatMultiplier is Classic's 1.75 (ClassicBearThreatMultiplier).
	// It is not Season of Discovery's 2.0: that season's tuning passive
	// (436895, "Modifies Threat +101%" on Swipe) is not taught to a Forever
	// druid.
	SwipeThreatMultiplier = ClassicBearThreatMultiplier

	// feralInstinctSwipePerRank is Feral Instinct's "Increases damage done
	// by your Swipe ability by 10%" a rank (node 104940, three ranks).
	feralInstinctSwipePerRank = 0.1
	feralInstinctMaxRank      = 3
)

func (druid *Druid) feralInstinctSwipeBonus() float64 {
	return feralInstinctSwipePerRank * float64(clampRank(druid.Talents.FeralInstinct, feralInstinctMaxRank))
}

func (druid *Druid) registerSwipeBearSpell() {
	rank := core.HighestRankAtLevel(SwipeLevel[1:], druid.Level)
	if rank == 0 {
		return
	}
	damage := SwipeDamage[rank]
	casterLevel := int(druid.Level)

	// Pool-sized ceiling, live-bounded loop; see APLActionMultidot.
	results := make([]*core.SpellResult, min(swipeMaxTargets, len(druid.Env.Encounter.AllTargetUnits)))

	druid.SwipeBear = druid.RegisterSpell(Bear, core.SpellConfig{
		SpellCode:      SpellCode_DruidSwipe,
		ClassSpellMask: DruidSpellMaskSwipe,
		ActionID:       core.ActionID{SpellID: SwipeSpellId[rank]},
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          SpellFlagOmen | core.SpellFlagMeleeMetrics | core.SpellFlagAPL,

		Rank:          rank,
		RequiredLevel: SwipeLevel[rank],

		RageCost: core.RageCostOptions{
			Cost: swipeRageCost - ferocityRageDiscount(druid.Talents.Ferocity),
		},

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			IgnoreHaste: true,
		},

		// Savage Fury and Feral Instinct are both additive damage
		// modifiers on the same spell.
		DamageMultiplierAdditive: druid.savageFuryDamageMultiplier() + druid.feralInstinctSwipeBonus(),
		DamageMultiplier:         1,
		ThreatMultiplier:         SwipeThreatMultiplier,
		BonusCoefficient:         1,
		ClientBaseDamage:         damage.Range(casterLevel),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			numHits := min(len(results), len(sim.Encounter.TargetUnits))
			for idx := 0; idx < numHits; idx++ {
				baseDamage := SwipeHitDamage(damage.Roll(sim, casterLevel), spell.MeleeAttackPower(target)) * druid.RendAndTearMultiplier(target)
				results[idx] = spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMeleeSpecialHitAndCrit)
				target = sim.Environment.NextTargetUnit(target)
			}

			for _, result := range results[:numHits] {
				spell.DealDamage(sim, result)
			}
		},
	})
}

// SwipeHitDamage is one target's hit before multipliers: the rank's roll plus
// swipeAttackPowerShare of the druid's attack power.
func SwipeHitDamage(roll, attackPower float64) float64 {
	return roll + swipeAttackPowerShare*attackPower
}
