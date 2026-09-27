package mage

import (
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

// frostIceArmorLearnLevels is the real client learn level (build
// 1.60.1.70009) for each id this aura already cycled through under the
// old SoD-phase bracket map (25/40/50/60): Frost Armor rank 3 (7301,
// learn level 20) is worn until Ice Armor ranks 2-4 (7320/10219/10220,
// learn levels 40/50/60) replace it. Ice Armor rank 1 (7302, learn level
// 30) was never one of the ids this aura showed and stays out of the
// table, same as before.
var frostIceArmorLearnLevels = []int{20, 40, 50, 60}

type frostIceArmorRank struct {
	spellID  int32
	armor    float64
	frostRes float64
}

var frostIceArmorRanks = map[int]frostIceArmorRank{
	1: {spellID: 7301, armor: 200, frostRes: 0},
	2: {spellID: 7320, armor: 380, frostRes: 9},
	3: {spellID: 10219, armor: 470, frostRes: 12},
	4: {spellID: 10220, armor: 560, frostRes: 15},
}

// frostIceArmorRankAtLevel reports the aura's data for the given
// character level, and false when no rank has been learned yet.
func frostIceArmorRankAtLevel(level int32) (frostIceArmorRank, bool) {
	rank := core.HighestRankAtLevel(frostIceArmorLearnLevels, level)
	if rank == 0 {
		return frostIceArmorRank{}, false
	}
	return frostIceArmorRanks[rank], true
}

func (mage *Mage) applyFrostIceArmor() {
	rankData, ok := frostIceArmorRankAtLevel(mage.Level)
	if !ok {
		return
	}
	spellID := rankData.spellID
	armor := rankData.armor
	frostRes := rankData.frostRes

	mage.IceArmorAura = core.MakePermanent(mage.RegisterAura(core.Aura{
		Label:    "Ice Armor",
		ActionID: core.ActionID{SpellID: spellID},
		// BuildPhase: core.CharacterBuildPhaseBuffs,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			if aura.Unit.Env.MeasuringStats && aura.Unit.Env.State != core.Finalized {
				mage.AddStat(stats.BonusArmor, armor)
				mage.AddStat(stats.FrostResistance, frostRes)
			} else {
				mage.AddStatDynamic(sim, stats.Armor, armor)
				mage.AddStatDynamic(sim, stats.FrostResistance, frostRes)
			}
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			if aura.Unit.Env.MeasuringStats && aura.Unit.Env.State != core.Finalized {
				mage.AddStat(stats.BonusArmor, -1*armor)
				mage.AddStat(stats.FrostResistance, -1*frostRes)
			} else {
				mage.AddStatDynamic(sim, stats.Armor, -1*armor)
				mage.AddStatDynamic(sim, stats.FrostResistance, -1*frostRes)
			}
		},
	}))
}

// mageArmorLearnLevels is the real client learn level (build
// 1.60.1.70009) for Mage Armor's three ranks (6117/22782/22783),
// replacing the old SoD-phase bracket map (40/50/60).
var mageArmorLearnLevels = []int{34, 46, 58}

type mageArmorRank struct {
	spellID  int32
	spellRes float64
}

var mageArmorRanksData = map[int]mageArmorRank{
	1: {spellID: 6117, spellRes: 5},
	2: {spellID: 22782, spellRes: 10},
	3: {spellID: 22783, spellRes: 15},
}

// mageArmorRankAtLevel reports Mage Armor's data for the given character
// level, and false when no rank has been learned yet.
func mageArmorRankAtLevel(level int32) (mageArmorRank, bool) {
	rank := core.HighestRankAtLevel(mageArmorLearnLevels, level)
	if rank == 0 {
		return mageArmorRank{}, false
	}
	return mageArmorRanksData[rank], true
}

func (mage *Mage) applyMageArmor() {
	rankData, ok := mageArmorRankAtLevel(mage.Level)
	if !ok {
		return
	}
	spellID := rankData.spellID
	spellRes := rankData.spellRes

	mage.MageArmorAura = core.MakePermanent(mage.RegisterAura(core.Aura{
		Label:      "Mage Armor",
		ActionID:   core.ActionID{SpellID: spellID},
		BuildPhase: core.CharacterBuildPhaseBuffs,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			mage.PseudoStats.SpiritRegenRateCasting += .3

			if aura.Unit.Env.MeasuringStats && aura.Unit.Env.State != core.Finalized {
				mage.AddResistances(spellRes)
			} else {
				mage.AddResistancesDynamic(sim, spellRes)
			}
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			mage.PseudoStats.SpiritRegenRateCasting -= .3

			if aura.Unit.Env.MeasuringStats && aura.Unit.Env.State != core.Finalized {
				mage.AddResistances(-1 * spellRes)
			} else {
				mage.AddResistancesDynamic(sim, -1*spellRes)
			}
		},
	}))
}
