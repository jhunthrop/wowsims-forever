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

	// Index 0 is 0 combo points, which ExtraCastCondition below never
	// actually allows (Slice and Dice requires at least 1), but it is the
	// formula's own base term (6s + 3s/combo), and it is also exactly the
	// client's duration_ms for this spell (6000, every rank) - so it
	// doubles as the aura's pre-cast registration default, read by the
	// conformance report's engineDuration before any real cast overrides
	// it (see priest/vampiric_embrace.go's RelatedSelfBuff comment for the
	// same report-visibility reasoning).
	rogue.sliceAndDiceDurations = [6]time.Duration{
		time.Duration(float64(time.Second*6) * durationMultiplier),
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
		// Overridden on every real cast (ApplyEffects below); the base
		// (0-combo-point) term is a safe non-zero default for APL prepull.
		Duration: rogue.sliceAndDiceDurations[0],
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			rogue.MultiplyMeleeSpeed(sim, hasteBonus)
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			rogue.MultiplyMeleeSpeed(sim, inverseHasteBonus)
		},
	})

	rogue.SliceAndDice = rogue.RegisterSpell(core.SpellConfig{
		SpellCode:       SpellCode_RogueSliceandDice,
		ActionID:        actionID,
		Flags:           core.SpellFlagAPL,
		MetricSplits:    6,
		RequiredLevel:   sliceAndDiceLearnLevels[rank-1],
		RelatedSelfBuff: rogue.SliceAndDiceAura,

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
