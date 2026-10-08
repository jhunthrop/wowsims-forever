package shaman

import (
	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

// lesserHealingWaveTopRank is rank 6. The generated table lists the client's
// two spells named Lesser Healing Wave rank 6 and keeps 27624 (healing 880):
// a Burning Crusade id with no SkillLineAbility learn row, so a Forever
// shaman cannot learn it. The one a player learns, at level 60, is 10468
// (healing 820), with the same cost, cast time and coefficient.
const lesserHealingWaveTopRank = 6

var lesserHealingWaveTopRankHealing = clientdamage.Effect{
	Amount: 820, Variance: 0.10909090936, PerLevel: 4.2, SpellLevel: 60, MaxLevel: 65,
}

const lesserHealingWaveTopRankSpellID int32 = 10468

// lesserHealingWaveRanks is the generated table for ranks 1 to 5 and the
// learnable rank 6 above. Index 0 of the generated arrays is the unrelated
// 28849 aura and is never read.
func lesserHealingWaveRanks() []healingRank {
	healing := clientdamage.FromTable(LesserHealingWaveBaseDamage[:], LesserHealingWavePointsPerLevel[:], LesserHealingWaveLevel[:], LesserHealingWaveMaxLevel[:])
	ranks := make([]healingRank, 0, LesserHealingWaveRanks)
	for rank := 1; rank <= LesserHealingWaveRanks; rank++ {
		entry := healingRank{
			rank:        rank,
			spellID:     LesserHealingWaveSpellId[rank],
			level:       LesserHealingWaveLevel[rank],
			manaCost:    LesserHealingWaveManaCost[rank],
			castTime:    castTimeFromMS(LesserHealingWaveCastTime[rank]),
			coefficient: LesserHealingWaveSpellCoeff[rank],
			healing:     healing[rank],
		}
		if rank == lesserHealingWaveTopRank {
			entry.spellID = lesserHealingWaveTopRankSpellID
			entry.healing = lesserHealingWaveTopRankHealing
		}
		ranks = append(ranks, entry)
	}
	return ranks
}

func (shaman *Shaman) registerLesserHealingWaveSpell() {
	shaman.LesserHealingWave = shaman.registerHealingRanks(lesserHealingWaveRanks(), func(rank healingRank) core.SpellConfig {
		config := shaman.newHealingSpellConfig(rank, ShamanSpellMaskLesserHealingWave)
		config.SpellCode = SpellCode_ShamanLesserHealingWave
		return config
	})
}
