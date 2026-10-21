package warlock

import (
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

// Demon Armor has five ranks in the client (build 1.60.1.70009): 706,
// 1086, 11733, 11734 and 11735, learned at levels 20/30/40/50/60. Armor
// is each rank's effect 0 (the generated DemonArmorBaseDamage, which
// holds the flat amount), Shadow resistance its effect 1. The third
// effect is health regeneration, which no sim number reads.
//
// An earlier table skipped rank 2 ("never one of the ids this aura
// showed"), so a warlock between levels 30 and 39 wore rank 1.
var demonArmorShadowResistance = [DemonArmorRanks + 1]float64{0, 3, 6, 9, 12, 15}

type demonArmorRank struct {
	spellID   int32
	armor     float64
	shadowRes float64
}

// demonArmorRankAtLevel reports Demon Armor's data for the given
// character level, and false when no rank has been learned yet.
func demonArmorRankAtLevel(level int32) (demonArmorRank, bool) {
	rank := core.HighestRankAtLevel(DemonArmorLevel[1:], level)
	if rank == 0 {
		return demonArmorRank{}, false
	}
	return demonArmorRank{
		spellID:   DemonArmorSpellId[rank],
		armor:     DemonArmorBaseDamage[rank][0],
		shadowRes: demonArmorShadowResistance[rank],
	}, true
}

func (warlock *Warlock) applyDemonArmor() {
	rankData, ok := demonArmorRankAtLevel(warlock.Level)
	if !ok {
		return
	}

	warlock.AddStat(stats.Armor, rankData.armor)
	warlock.AddStat(stats.ShadowResistance, rankData.shadowRes)

	warlock.GetOrRegisterAura(core.Aura{
		Label:    "Demon Armor",
		ActionID: core.ActionID{SpellID: rankData.spellID},
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
	})
}
