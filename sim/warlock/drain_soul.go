package warlock

import (
	"strconv"
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

const DrainSoulRanks = 4

// DrainSoulTickDamage is spellconst/warlock.json's own per-tick amount for
// ids 1120 through 11675 (rank 4: 84 over 5 ticks, no growth).
var DrainSoulTickDamage = [DrainSoulRanks + 1]clientdamage.Effect{
	{},
	{Amount: 17, SpellLevel: 10},
	{Amount: 34, SpellLevel: 24},
	{Amount: 54, SpellLevel: 38},
	{Amount: 84, SpellLevel: 52},
}

func (warlock *Warlock) getDrainSoulBaseConfig(rank int) core.SpellConfig {
	baseNumTicks := int32(5)
	numTicks := baseNumTicks
	tickLength := time.Second * 3

	spellId := [DrainSoulRanks + 1]int32{0, 1120, 8288, 8289, 11675}[rank]
	spellCoeff := [DrainSoulRanks + 1]float64{0, 0.1, 0.1, 0.1, 0.1}[rank]
	// Per-tick base damage, straight from spellconst/warlock.json's own
	// flat "amount" for each rank's damage effect (period_ms 3000, 5
	// ticks): 1120/8288/8289/11675 -> 17/34/54/84. This used to be a
	// classic-tooltip total (55/155/295/455) divided by baseNumTicks,
	// which drifted 10-30% high of the client's real per-tick number
	// (rank 4: 91 vs 84).
	damage := DrainSoulTickDamage[rank]
	casterLevel := int(warlock.Level)
	baseDamage := damage.Center(casterLevel)
	manaCost := [DrainSoulRanks + 1]float64{0, 55, 125, 210, 290}[rank]
	level := [DrainSoulRanks + 1]int{0, 10, 24, 38, 52}[rank]

	return core.SpellConfig{
		SpellCode:   SpellCode_WarlockDrainSoul,
		ActionID:    core.ActionID{SpellID: spellId},
		SpellSchool: core.SpellSchoolShadow,
		DefenseType: core.DefenseTypeMagic,
		ProcMask:    core.ProcMaskSpellDamage,
		Flags:       core.SpellFlagAPL | core.SpellFlagChanneled | core.SpellFlagResetAttackSwing | WarlockFlagAffliction,

		RequiredLevel:    level,
		Rank:             rank,
		ClientBaseDamage: damage.Range(casterLevel),

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: spellCoeff, // the report compares the spell's, which a pure DoT never reads

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "DrainSoul-" + warlock.Label + strconv.Itoa(rank),
			},
			NumberOfTicks:    numTicks,
			TickLength:       tickLength,
			BonusCoefficient: spellCoeff,

			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
				dot.Snapshot(target, baseDamage, isRollover)
				if !isRollover {
					// Soul Siphon (talents.go): see drain_life.go's
					// identical hook for the talent's wording.
					dot.SnapshotAttackerMultiplier *= warlock.soulSiphonMultiplier(target, dot.Spell)
				}
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcOutcome(sim, target, spell.OutcomeMagicHitNoHitCounter)
			if result.Landed() {

				dot := spell.Dot(target)
				dot.Apply(sim)
			}
			spell.DealOutcome(sim, result)
		},
		ExpectedTickDamage: func(sim *core.Simulation, target *core.Unit, spell *core.Spell, useSnapshot bool) *core.SpellResult {
			if useSnapshot {
				dot := spell.Dot(target)
				return dot.CalcSnapshotDamage(sim, target, spell.OutcomeExpectedMagicAlwaysHit)
			} else {
				return spell.CalcPeriodicDamage(sim, target, baseDamage, spell.OutcomeExpectedMagicAlwaysHit)
			}
		},
	}
}

func (warlock *Warlock) registerDrainSoulSpell() {
	warlock.DrainSoul = make([]*core.Spell, 0)
	for rank := 1; rank <= DrainSoulRanks; rank++ {
		config := warlock.getDrainSoulBaseConfig(rank)

		if config.RequiredLevel <= int(warlock.Level) {
			warlock.DrainSoul = append(warlock.DrainSoul, warlock.GetOrRegisterSpell(config))
		}
	}
}
