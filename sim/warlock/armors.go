package warlock

import (
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

// demonArmorLearnLevels is the real client learn level (build
// 1.60.1.70009) for each id this aura already cycled through under the
// old SoD-phase bracket map (25/40/50/60): Demon Armor rank 1 (706,
// learn level 20) is worn until ranks 3-5 (11733/11734/11735, learn
// levels 40/50/60) replace it. Demon Armor rank 2 (1086, learn level 30)
// was never one of the ids this aura showed and stays out of the table,
// same as before.
var demonArmorLearnLevels = []int{20, 40, 50, 60}

type demonArmorRank struct {
	spellID   int32
	armor     float64
	shadowRes float64
}

var demonArmorRanks = map[int]demonArmorRank{
	1: {spellID: 706, armor: 210.0, shadowRes: 3.0},
	2: {spellID: 11733, armor: 390.0, shadowRes: 9.0},
	3: {spellID: 11734, armor: 480.0, shadowRes: 12.0},
	4: {spellID: 11735, armor: 570.0, shadowRes: 15.0},
}

// demonArmorRankAtLevel reports Demon Armor's data for the given
// character level, and false when no rank has been learned yet.
func demonArmorRankAtLevel(level int32) (demonArmorRank, bool) {
	rank := core.HighestRankAtLevel(demonArmorLearnLevels, level)
	if rank == 0 {
		return demonArmorRank{}, false
	}
	return demonArmorRanks[rank], true
}

func (warlock *Warlock) applyDemonArmor() {
	rankData, ok := demonArmorRankAtLevel(warlock.Level)
	if !ok {
		return
	}
	spellID := rankData.spellID
	armor := rankData.armor
	shadowRes := rankData.shadowRes

	warlock.AddStat(stats.Armor, armor)
	warlock.AddStat(stats.ShadowResistance, shadowRes)

	warlock.GetOrRegisterAura(core.Aura{
		Label:    "Demon Armor",
		ActionID: core.ActionID{SpellID: spellID},
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
	})
}
