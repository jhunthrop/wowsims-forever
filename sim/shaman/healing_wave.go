package shaman

import (
	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

// healingWaveRanks reads the generated Healing Wave tables as they are:
// every rank, 331 through 25357, is the client's own row (rank 1 has a
// learn row with AcquireMethod 2, a spell granted at creation).
func healingWaveRanks() []healingRank {
	healing := clientdamage.FromTable(HealingWaveBaseDamage[:], HealingWavePointsPerLevel[:], HealingWaveLevel[:], HealingWaveMaxLevel[:])
	ranks := make([]healingRank, 0, HealingWaveRanks)
	for rank := 1; rank <= HealingWaveRanks; rank++ {
		ranks = append(ranks, healingRank{
			rank:        rank,
			spellID:     HealingWaveSpellId[rank],
			level:       HealingWaveLevel[rank],
			manaCost:    HealingWaveManaCost[rank],
			castTime:    castTimeFromMS(HealingWaveCastTime[rank]),
			coefficient: HealingWaveSpellCoeff[rank],
			healing:     healing[rank],
		})
	}
	return ranks
}

func (shaman *Shaman) registerHealingWaveSpell() {
	shaman.HealingWave = shaman.registerHealingRanks(healingWaveRanks(), func(rank healingRank) core.SpellConfig {
		config := shaman.newHealingSpellConfig(rank, ShamanSpellMaskHealingWave)
		config.SpellCode = SpellCode_ShamanHealingWave
		return config
	})
}
