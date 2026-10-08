package shaman

import (
	"fmt"
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

// Riptide (talent 104724, learned from the trainer in three ranks) heals
// its target and leaves a heal over time that makes the shaman's Chain Heal
// cast directly on that target heal more.
const (
	riptideCooldown = 6 * time.Second
	riptideTicks    = 5
	riptideTickGap  = 3 * time.Second
	// riptideTickCoefficient is the client's per-tick spell-power share
	// (effect 1); the direct heal's 0.214 is the generated column.
	riptideTickCoefficient = 0.1
	// riptideChainHealBonusPercent is effect 2 of every rank, the "25%"
	// of "increases the effectiveness of your Chain Heal casts directly
	// on that target by 25%".
	riptideChainHealBonusPercent = 25
)

// riptideTickHealing is each rank's heal-over-time tick as the client
// states it (spellconst effect 1, amounts 89, 115 and 161); rank 3 alone
// grows with level. The generated tables carry only the direct heal.
var riptideTickHealing = [RiptideRanks + 1]clientdamage.Effect{
	{},
	{Amount: 89, SpellLevel: 40, MaxLevel: 48},
	{Amount: 115, SpellLevel: 50, MaxLevel: 58},
	{Amount: 161, PerLevel: 3.7, SpellLevel: 60, MaxLevel: 68},
}

// riptideRanks skips the generated table's index 0 (409954, a talent
// passive that shares the name).
func riptideRanks() []healingRank {
	healing := clientdamage.FromTable(RiptideBaseDamage[:], RiptidePointsPerLevel[:], RiptideLevel[:], RiptideMaxLevel[:])
	ranks := make([]healingRank, 0, RiptideRanks)
	for rank := 1; rank <= RiptideRanks; rank++ {
		ranks = append(ranks, healingRank{
			rank:        rank,
			spellID:     RiptideSpellId[rank],
			level:       RiptideLevel[rank],
			manaCost:    RiptideManaCost[rank],
			castTime:    castTimeFromMS(RiptideCastTime[rank]),
			coefficient: RiptideSpellCoeff[rank],
			healing:     healing[rank],
		})
	}
	return ranks
}

func (shaman *Shaman) registerRiptideSpell() {
	if !shaman.Talents.Riptide {
		return
	}
	// The ranks are one spell to the client, so they share the cooldown.
	cooldown := core.Cooldown{Timer: shaman.NewTimer(), Duration: riptideCooldown}
	shaman.Riptide = shaman.registerHealingRanks(riptideRanks(), func(rank healingRank) core.SpellConfig {
		return shaman.newRiptideConfig(rank, cooldown)
	})
}

func (shaman *Shaman) newRiptideConfig(rank healingRank, cooldown core.Cooldown) core.SpellConfig {
	config := shaman.newHealingSpellConfig(rank, ShamanSpellMaskRiptide)
	config.Cast.CD = cooldown

	casterLevel := int(shaman.Level)
	tick := riptideTickHealing[rank.rank]
	directHeal := config.ApplyEffects

	config.Hot = core.DotConfig{
		Aura:             core.Aura{Label: fmt.Sprintf("Riptide (Rank %d)", rank.rank)},
		NumberOfTicks:    riptideTicks,
		TickLength:       riptideTickGap,
		BonusCoefficient: riptideTickCoefficient,
		OnSnapshot: func(_ *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
			dot.SnapshotHeal(target, tick.Center(casterLevel), isRollover)
			if !isRollover {
				// SnapshotHeal takes the damage-side multiplier; a heal
				// scales with the healing-side one.
				dot.SnapshotAttackerMultiplier = dot.Spell.CasterHealingMultiplier() * dot.DamageMultiplier
			}
		},
		OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
			dot.CalcAndDealPeriodicSnapshotHealing(sim, target, dot.Spell.OutcomeHealing)
		},
	}

	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		directHeal(sim, target, spell)
		shaman.cancelRiptide(sim, target)
		spell.Hot(target).Apply(sim)
	}
	return config
}

// cancelRiptide ends every rank's Riptide on target: the ranks are one
// spell to the client, so a new cast replaces whichever rank was there.
func (shaman *Shaman) cancelRiptide(sim *core.Simulation, target *core.Unit) {
	for _, riptide := range shaman.Riptide {
		if riptide != nil {
			riptide.Hot(target).Cancel(sim)
		}
	}
}

// riptideChainHealBonus is the multiplier on a Chain Heal's first heal:
// 1.25 while any rank of Riptide is on its target, else 1.
func (shaman *Shaman) riptideChainHealBonus(target *core.Unit) float64 {
	for _, riptide := range shaman.Riptide {
		if riptide != nil && riptide.Hot(target).IsActive() {
			return 1 + riptideChainHealBonusPercent/100.0
		}
	}
	return 1
}
