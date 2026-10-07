package mage

import (
	"fmt"
	"time"

	"github.com/wowsims/classic/sim/core"
)

// FireballDotTickDamage is the client's per-tick amount of the periodic
// effect (effect 1 of each rank), its own number and not a share of the
// direct hit: flat, with no growth per caster level.
var FireballDotTickDamage = [FireballRanks + 1]float64{0, 1, 1, 2, 3, 4, 6, 6, 8, 10, 12, 14, 15}

// FireballDotTicks is the client's own per-rank tick count for the
// Fireball dot: duration_ms / period_ms (period_ms is always 2000 -
// see mage.json spells 133/143/145/3140's effect index 1). Ranks 1-3
// burn for fewer than the later ranks' 4 ticks (4000/6000/6000/8000ms
// at 2s/tick = 2/3/3/4 ticks).
var FireballDotTicks = [FireballRanks + 1]int32{0, 2, 3, 3, 4, 4, 4, 4, 4, 4, 4, 4, 4}

func (mage *Mage) registerFireballSpell() {
	mage.Fireball = make([]*core.Spell, FireballRanks+1)

	maxRank := core.TernaryInt(core.IncludeAQ, FireballRanks, FireballRanks-1)
	for rank := 1; rank <= maxRank; rank++ {
		config := mage.newFireballSpellConfig(rank)

		if config.RequiredLevel <= int(mage.Level) {
			mage.Fireball[rank] = mage.GetOrRegisterSpell(config)
		}
	}
}

func (mage *Mage) newFireballSpellConfig(rank int) core.SpellConfig {
	numTicks := FireballDotTicks[rank]
	tickLength := time.Second * 2

	spellId := FireballSpellId[rank]
	roll := mage.clientRoll(FireballBaseDamage[rank], FireballPointsPerLevel[rank], FireballLevel[rank], FireballMaxLevel[rank])
	baseDamageLow, baseDamageHigh := roll[0], roll[1]
	baseDotDamage := FireballDotTickDamage[rank]
	spellCoeff := FireballSpellCoeff[rank]
	castTime := FireballCastTime[rank]
	manaCost := FireballManaCost[rank]
	level := FireballLevel[rank]

	actionID := core.ActionID{SpellID: spellId}

	return core.SpellConfig{
		ActionID:         actionID,
		ClassSpellMask:   MageSpellMaskFireball,
		SpellCode:        SpellCode_MageFireball,
		SpellSchool:      core.SpellSchoolFire,
		DefenseType:      core.DefenseTypeMagic,
		ProcMask:         core.ProcMaskSpellDamage,
		ClientBaseDamage: roll,
		Flags:            core.SpellFlagAPL | SpellFlagMage,
		MissileSpeed:     24,

		RequiredLevel: level,
		Rank:          rank,

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD:      core.GCDDefault,
				CastTime: time.Millisecond*time.Duration(castTime) - time.Millisecond*100*time.Duration(mage.Talents.ImprovedFireball),
			},
		},

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label:    fmt.Sprintf("Fireball (Rank %d)", rank),
				ActionID: actionID.WithTag(1),
			},
			NumberOfTicks: numTicks,
			TickLength:    tickLength,
			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
				dot.Snapshot(target, baseDotDamage, isRollover)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: spellCoeff,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := sim.Roll(baseDamageLow, baseDamageHigh)
			result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)
			spell.WaitTravelTime(sim, func(sim *core.Simulation) {
				spell.DealDamage(sim, result)

				if result.Landed() {
					spell.Dot(target).Apply(sim)
				}
			})
		},
	}
}
