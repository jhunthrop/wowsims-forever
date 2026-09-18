package shaman

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Totem Item IDs
const (
	StormfuryTotem           = 31031
	TotemOfAncestralGuidance = 32330
	TotemOfStorms            = 23199
	TotemOfTheVoid           = 28248
	TotemOfHex               = 40267
	VentureCoLightningRod    = 38361
	ThunderfallTotem         = 45255
)

// Shared precomputation logic for LB and CL.
func (shaman *Shaman) newElectricSpellConfig(actionID core.ActionID, baseCost float64, baseCastTime time.Duration) core.SpellConfig {
	spell := core.SpellConfig{
		ActionID:     actionID,
		SpellSchool:  core.SpellSchoolNature,
		DefenseType:  core.DefenseTypeMagic,
		ProcMask:     core.ProcMaskSpellDamage,
		Flags:        SpellFlagShaman | SpellFlagLightning | core.SpellFlagAPL,
		MetricSplits: 6,

		ManaCost: core.ManaCostOptions{
			FlatCost:   baseCost,
			Multiplier: 100 - 2*shaman.Talents.Convection,
		},

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				// FOREVER: Lightning Mastery is not in the client's trees.
				// CastTime: baseCastTime - time.Millisecond*200*time.Duration(shaman.Talents.LightningMastery),
				CastTime: baseCastTime,
				GCD:      core.GCDDefault,
			},
			ModifyCast: func(sim *core.Simulation, spell *core.Spell, cast *core.Cast) {
				castTime := shaman.ApplyCastSpeedForSpell(cast.CastTime, spell)
				shaman.AutoAttacks.StopMeleeUntil(sim, sim.CurrentTime+castTime, false)
			},
		},

		// FOREVER: Call of Thunder is a single-rank talent in the client's
		// trees, so the field is a bool now rather than an int32.
		// BonusCritRating: []float64{0, 1, 2, 3, 4, 6}[shaman.Talents.CallOfThunder] * core.CritRatingPerCritChance,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
	}

	return spell
}
