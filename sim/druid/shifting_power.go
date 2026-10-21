package druid

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Shifting Power (Feral node 104951, hotfix_only) replaced King of the
// Jungle in the live tree. Wowhead's text, the only source for it: "Instantly
// convert 0 Mana into 40 Energy. Shifting Power's cost is reduced by
// effects that reduce the cost of Shapeshifting." The 0 is an unresolved
// hotfix cost; the lane brief gives 55% of base mana (the same fraction
// Cat Form costs) and a 16 s cooldown. Improved Shifting Power (node
// 113563, hotfix_only): "Reduces the cooldown of your Shifting Power spell
// by 4/8 sec." Neither spell is in the client tables, so every number
// here comes from the live text and the brief.
//
// Build 1.60.1.70291 added the spell to the tables: 16 s cooldown, 55% of
// base mana, 40 energy, a 1 s global cooldown, learned at level 1.
const (
	shiftingPowerSpellID            = 1322605
	shiftingPowerManaFractionOfBase = 0.55
	ShiftingPowerEnergy             = 40.0
	shiftingPowerCooldown           = 16 * time.Second
	// shiftingPowerGCD is spell 1322605's StartRecoveryTime in build
	// 1.60.1.70291, a second shorter than the shapeshifts' 1.5 s.
	shiftingPowerGCD                   = time.Second
	improvedShiftingPowerPerRankCDDrop = 4 * time.Second
	improvedShiftingPowerMaxRank       = 2
)

// shiftingPowerCooldownAt is the spell's cooldown for a rank of Improved
// Shifting Power.
func shiftingPowerCooldownAt(improvedRank int32) time.Duration {
	return shiftingPowerCooldown - improvedShiftingPowerPerRankCDDrop*time.Duration(clampRank(improvedRank, improvedShiftingPowerMaxRank))
}

func (druid *Druid) registerShiftingPowerSpell() {
	if !druid.Talents.ShiftingPower {
		return
	}

	energyMetrics := druid.NewEnergyMetrics(core.ActionID{SpellID: shiftingPowerSpellID})

	druid.ShiftingPower = druid.RegisterSpell(Cat, core.SpellConfig{
		ActionID:       core.ActionID{SpellID: shiftingPowerSpellID},
		ClassSpellMask: DruidSpellMaskShiftingPower,
		Flags:          core.SpellFlagNoOnCastComplete | core.SpellFlagAPL,
		RequiredLevel:  1,

		ManaCost: core.ManaCostOptions{
			BaseCost:   shiftingPowerManaFractionOfBase,
			Multiplier: 100 - 10*druid.Talents.NaturalShapeshifter,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: shiftingPowerGCD,
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    druid.NewTimer(),
				Duration: shiftingPowerCooldownAt(druid.Talents.ImprovedShiftingPower),
			},
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			druid.AddEnergy(sim, ShiftingPowerEnergy, energyMetrics)
		},
	})
}
