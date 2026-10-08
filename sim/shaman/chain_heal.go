package shaman

import (
	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

// The client's spell constants carry Chain Heal's heal but not how it
// chains (the chain target count and the falloff per jump live in effect
// columns the constants do not export), so both are an assumption taken
// from the spell's vanilla behaviour: it heals the target, then jumps to
// the two most injured other members, each jump healing half of the
// previous heal.
const (
	chainHealJumps      = 2
	chainHealJumpFactor = 0.5
)

func chainHealRanks() []healingRank {
	healing := clientdamage.FromTable(ChainHealBaseDamage[:], ChainHealPointsPerLevel[:], ChainHealLevel[:], ChainHealMaxLevel[:])
	ranks := make([]healingRank, 0, ChainHealRanks)
	for rank := 1; rank <= ChainHealRanks; rank++ {
		ranks = append(ranks, healingRank{
			rank:        rank,
			spellID:     ChainHealSpellId[rank],
			level:       ChainHealLevel[rank],
			manaCost:    ChainHealManaCost[rank],
			castTime:    castTimeFromMS(ChainHealCastTime[rank]),
			coefficient: ChainHealSpellCoeff[rank],
			healing:     healing[rank],
		})
	}
	return ranks
}

func (shaman *Shaman) registerChainHealSpell() {
	shaman.ChainHeal = shaman.registerHealingRanks(chainHealRanks(), shaman.newChainHealConfig)
}

func (shaman *Shaman) newChainHealConfig(rank healingRank) core.SpellConfig {
	config := shaman.newHealingSpellConfig(rank, ShamanSpellMaskChainHeal)
	config.SpellCode = SpellCode_ShamanChainHeal
	casterLevel := int(shaman.Level)

	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		baseHealing := rank.healing.Roll(sim, casterLevel)
		falloff := 1.0
		for i, healed := range shaman.chainHealTargets(target) {
			result := spell.CalcHealing(sim, healed, baseHealing, spell.OutcomeHealingCrit)
			result.Damage *= falloff
			if i == 0 {
				// Riptide boosts only the Chain Heal cast directly on its target.
				result.Damage *= shaman.riptideChainHealBonus(healed)
			}
			spell.DealHealing(sim, result)
			falloff *= chainHealJumpFactor
		}
	}
	return config
}

// chainHealTargets is the primary target followed by the most injured
// other members of the raid, up to chainHealJumps of them. Members without a
// health bar (the healer itself) cannot be jumped to.
func (shaman *Shaman) chainHealTargets(primary *core.Unit) []*core.Unit {
	targets := make([]*core.Unit, 1, 1+chainHealJumps)
	targets[0] = primary
	for len(targets) <= chainHealJumps {
		next := shaman.mostInjuredMember(targets)
		if next == nil {
			break
		}
		targets = append(targets, next)
	}
	return targets
}

// mostInjuredMember is the raid member with the lowest health share that
// is not in exclude, or nil when none can be healed.
func (shaman *Shaman) mostInjuredMember(exclude []*core.Unit) *core.Unit {
	var best *core.Unit
	for _, member := range shaman.Env.Raid.AllPlayerUnits {
		if !member.HasHealthBar() || containsUnit(exclude, member) {
			continue
		}
		if best == nil || member.CurrentHealthPercent() < best.CurrentHealthPercent() {
			best = member
		}
	}
	return best
}

func containsUnit(units []*core.Unit, unit *core.Unit) bool {
	for _, candidate := range units {
		if candidate == unit {
			return true
		}
	}
	return false
}
