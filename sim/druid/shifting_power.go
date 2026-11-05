package druid

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Shifting Power (Feral node 113563, the tier 6 talent behind Predatory
// Discipline since client 1.60.1.70707): "Instantly convert 0 Mana into 40
// Energy. Shifting Power's cost is reduced by effects that reduce the cost
// of Shapeshifting." Build 1.60.1.70291 added the spell to the tables: 55%
// of base mana, 40 energy, a 1 s global cooldown, learned at level 1, and a
// 16 s cooldown that Improved Shifting Power (-4/8 sec) shortened.
//
// The 70707 hotfix removed Improved Shifting Power and folded its full
// reduction into the spell. Since 1.60.1.70755 the client tables carry the
// 8 s themselves: SpellCooldowns 101953 (RecoveryTime 8000, StartRecoveryTime
// 1000) for spell 1322605, and Wowhead's Forever tooltip agrees.
const (
	shiftingPowerSpellID            = 1322605
	shiftingPowerManaFractionOfBase = 0.55
	ShiftingPowerEnergy             = 40.0
	shiftingPowerCooldown           = 8 * time.Second
	// shiftingPowerGCD is spell 1322605's StartRecoveryTime in build
	// 1.60.1.70291, a second shorter than the shapeshifts' 1.5 s.
	shiftingPowerGCD = time.Second
)

func (druid *Druid) registerShiftingPowerSpell() {
	if !druid.Talents.ShiftingPower {
		return
	}

	energyMetrics := druid.NewEnergyMetrics(core.ActionID{SpellID: shiftingPowerSpellID})
	wolfsheadEnergy := 0.0
	var wolfsheadMetrics *core.ResourceMetrics
	if druid.wearsWolfsheadHelm() {
		wolfsheadEnergy = wolfsheadHelmShiftingPowerEnergy
		wolfsheadMetrics = druid.NewEnergyMetrics(core.ActionID{SpellID: wolfsheadHelmEnergySpellID})
	}

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
				Duration: shiftingPowerCooldown,
			},
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			druid.AddEnergy(sim, ShiftingPowerEnergy, energyMetrics)
			if wolfsheadEnergy > 0 {
				druid.AddEnergy(sim, wolfsheadEnergy, wolfsheadMetrics)
			}
		},
	})
}
