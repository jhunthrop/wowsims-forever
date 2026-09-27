package rogue

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// sliceAndDiceLearnLevels are Slice and Dice's two rank learn levels;
// source: 1.60.1.70009 client spell data ("Slice and Dice", ranks 1-2).
var sliceAndDiceLearnLevels = []int{10, 42}

// sliceAndDiceSpellID and sliceAndDiceHasteBonus are Slice and Dice's rank
// -> spell id / haste bonus, index 0 unused.
var sliceAndDiceSpellID = [3]int32{0, 5171, 6774}
var sliceAndDiceHasteBonus = [3]float64{0, 0.20, 0.30}

func (rogue *Rogue) registerSliceAndDice() {
	rank := core.HighestRankAtLevel(sliceAndDiceLearnLevels, rogue.Level)
	if rank == 0 {
		return
	}

	hasteBonusByRank := sliceAndDiceHasteBonus[rank]
	spellID := sliceAndDiceSpellID[rank]

	actionID := core.ActionID{SpellID: spellID}

	durationMultiplier := []float64{1, 1.15, 1.3, 1.45}[rogue.Talents.ImprovedSliceAndDice]

	rogue.sliceAndDiceDurations = [6]time.Duration{
		0,
		time.Duration(float64(time.Second*9) * durationMultiplier),
		time.Duration(float64(time.Second*12) * durationMultiplier),
		time.Duration(float64(time.Second*15) * durationMultiplier),
		time.Duration(float64(time.Second*18) * durationMultiplier),
		time.Duration(float64(time.Second*21) * durationMultiplier),
	}

	hasteBonus := 1 + hasteBonusByRank
	inverseHasteBonus := 1.0 / hasteBonus

	rogue.SliceAndDiceAura = rogue.RegisterAura(core.Aura{
		Label:    "Slice and Dice",
		ActionID: actionID,
		// This will be overridden on cast, but set a non-zero default so it doesn't crash when used in APL prepull
		Duration: rogue.sliceAndDiceDurations[5],
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			rogue.MultiplyMeleeSpeed(sim, hasteBonus)
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			rogue.MultiplyMeleeSpeed(sim, inverseHasteBonus)
		},
	})

	rogue.SliceAndDice = rogue.RegisterSpell(core.SpellConfig{
		SpellCode:    SpellCode_RogueSliceandDice,
		ActionID:     actionID,
		Flags:        core.SpellFlagAPL,
		MetricSplits: 6,

		EnergyCost: core.EnergyCostOptions{
			Cost: 25,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: time.Second,
			},
			IgnoreHaste: true,
			ModifyCast: func(sim *core.Simulation, spell *core.Spell, cast *core.Cast) {
				spell.SetMetricsSplit(spell.Unit.ComboPoints())
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return rogue.ComboPoints() > 0
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
			rogue.SliceAndDiceAura.Duration = rogue.sliceAndDiceDurations[rogue.ComboPoints()]
			rogue.SliceAndDiceAura.Activate(sim)
			rogue.SpendComboPoints(sim, spell)
		},
	})
	rogue.Finishers = append(rogue.Finishers, rogue.SliceAndDice)
}
