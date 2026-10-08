package mage

import (
	"fmt"
	"time"

	"github.com/wowsims/classic/sim/core"
)

const BlizzardRanks = 6

var BlizzardSpellId = [BlizzardRanks + 1]int32{0, 10, 6141, 8427, 10185, 10186, 10187}
var BlizzardTickBaseDamage = [BlizzardRanks + 1][]float64{{0, 0}, {24, 24}, {42, 42}, {62, 62}, {87, 87}, {114, 114}, {146, 146}}
var BlizzardTickPointsPerLevel = [BlizzardRanks + 1]float64{0, 0.1, 0.2, 0.2, 0.3, 0.3, 0.4}
var BlizzardTickMaxLevel = [BlizzardRanks + 1]int{0, 25, 33, 41, 49, 57, 65}
var BlizzardTickSpellCoeff = [BlizzardRanks + 1]float64{0, 0.042, 0.042, 0.042, 0.042, 0.042, 0.042}
var BlizzardManaCost = [BlizzardRanks + 1]float64{0, 320, 520, 720, 935, 1160, 1400}
var BlizzardLevel = [BlizzardRanks + 1]int{0, 20, 28, 36, 44, 52, 60}

func (mage *Mage) registerBlizzardSpell() {
	mage.Blizzard = make([]*core.Spell, BlizzardRanks+1)

	for rank := 1; rank <= BlizzardRanks; rank++ {
		config := mage.newBlizzardSpellConfig(rank)

		if config.RequiredLevel <= int(mage.Level) {
			mage.Blizzard[rank] = mage.GetOrRegisterSpell(config)
			// The channel's duration lives on the AOE dot's own Aura
			// (registered on the caster, since IsAOE dots key off the
			// caster rather than a target - sim/core/dot.go's
			// createDots), not a RelatedSelfBuff set at config time;
			// wired here, post-registration, so the conformance report
			// can read it like any other self-buff spell's duration.
			mage.Blizzard[rank].RelatedSelfBuff = mage.Blizzard[rank].AOEDot().Aura
		}
	}
}

func (mage *Mage) newBlizzardSpellConfig(rank int) core.SpellConfig {
	numTicks := int32(8)
	tickLength := time.Second * 1

	spellId := BlizzardSpellId[rank]
	tickRoll := mage.clientRoll(BlizzardTickBaseDamage[rank], BlizzardTickPointsPerLevel[rank], BlizzardLevel[rank], BlizzardTickMaxLevel[rank])
	baseDamage := tickRoll[0]
	manaCost := BlizzardManaCost[rank]
	level := BlizzardLevel[rank]

	spellCoeff := BlizzardTickSpellCoeff[rank]

	var improvedBlizzardProcApplication *core.Spell
	if mage.Talents.ImprovedBlizzard > 0 {
		impId := []int32{0, 11185, 12487, 12488}[mage.Talents.ImprovedBlizzard]
		auras := mage.NewEnemyAuraArray(func(unit *core.Unit) *core.Aura {
			return unit.GetOrRegisterAura(core.Aura{
				ActionID: core.ActionID{SpellID: impId},
				Label:    "Improved Blizzard",
				Duration: time.Millisecond * 1500,
			})
		})
		improvedBlizzardProcApplication = mage.RegisterSpell(core.SpellConfig{
			ActionID: core.ActionID{SpellID: impId},
			ProcMask: core.ProcMaskSpellProc,
			Flags:    SpellFlagMage | core.SpellFlagNoLogs | SpellFlagChillSpell,
			ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				auras.Get(target).Activate(sim)
				mage.chillApplied(sim, spell, target)
			},
		})
	}

	return core.SpellConfig{
		ActionID:         core.ActionID{SpellID: spellId},
		ClassSpellMask:   MageSpellMaskBlizzard,
		SpellSchool:      core.SpellSchoolFrost,
		ProcMask:         core.ProcMaskSpellDamage,
		ClientBaseDamage: tickRoll,
		Flags:            SpellFlagMage | core.SpellFlagChanneled | core.SpellFlagAPL,

		RequiredLevel: level,
		Rank:          rank,

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
		},

		Dot: core.DotConfig{
			IsAOE: true,
			Aura: core.Aura{
				Label: fmt.Sprintf("Blizzard (Rank %d)", rank),
			},
			NumberOfTicks:    numTicks,
			TickLength:       tickLength,
			BonusCoefficient: spellCoeff,
			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
				dot.Snapshot(target, baseDamage, isRollover)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				for _, aoeTarget := range sim.Encounter.TargetUnits {
					dot.CalcAndDealPeriodicSnapshotDamage(sim, aoeTarget, dot.OutcomeTick)

					if improvedBlizzardProcApplication != nil {
						improvedBlizzardProcApplication.Cast(sim, aoeTarget)
					}
				}
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.AOEDot().Apply(sim)
		},
	}
}
