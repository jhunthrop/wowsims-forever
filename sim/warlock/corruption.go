package warlock

import (
	"strconv"
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

const CorruptionRanks = 7

// CorruptionTickDamage is spellconst/warlock.json's own per-tick amount for
// ids 172 through 25311 (rank 7: 73, period 3 s, no growth), roughly half the
// classic total-over-duration divided by the tick count - the same halving
// shadowbolt.go's comment documents across the rest of the kit.
var CorruptionTickDamage = [CorruptionRanks + 1]clientdamage.Effect{
	{},
	{Amount: 10, SpellLevel: 4, MaxLevel: 9},
	{Amount: 13, SpellLevel: 14, MaxLevel: 19},
	{Amount: 22, SpellLevel: 24, MaxLevel: 29},
	{Amount: 28, SpellLevel: 34, MaxLevel: 39},
	{Amount: 40, SpellLevel: 44, MaxLevel: 49},
	{Amount: 57, SpellLevel: 54, MaxLevel: 59},
	{Amount: 73, SpellLevel: 60, MaxLevel: 65},
}

func (warlock *Warlock) getCorruptionConfig(rank int) core.SpellConfig {
	dotTickCoeff := [CorruptionRanks + 1]float64{0, .2, .2, .2, .2, .2, .2, .2}[rank] // per tick
	ticks := [CorruptionRanks + 1]int32{0, 4, 5, 6, 6, 6, 6, 6}[rank]
	damage := CorruptionTickDamage[rank]
	casterLevel := int(warlock.Level)
	baseDamage := damage.Center(casterLevel)
	spellId := [CorruptionRanks + 1]int32{0, 172, 6222, 6223, 7648, 11671, 11672, 25311}[rank]
	manaCost := [CorruptionRanks + 1]float64{0, 35, 55, 100, 160, 225, 290, 340}[rank]
	level := [CorruptionRanks + 1]int{0, 4, 14, 24, 34, 44, 54, 60}[rank]

	castTime := time.Millisecond * (2000 - (400 * time.Duration(warlock.Talents.ImprovedCorruption)))

	return core.SpellConfig{
		ActionID:         core.ActionID{SpellID: spellId},
		SpellSchool:      core.SpellSchoolShadow,
		SpellCode:        SpellCode_WarlockCorruption,
		ProcMask:         core.ProcMaskSpellDamage,
		DefenseType:      core.DefenseTypeMagic,
		Flags:            core.SpellFlagAPL | core.SpellFlagResetAttackSwing | core.SpellFlagPureDot | WarlockFlagAffliction,
		Rank:             rank,
		ClientBaseDamage: damage.Range(casterLevel),
		RequiredLevel:    level,

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				CastTime: castTime,
				GCD:      core.GCDDefault,
			},
		},

		CritDamageBonus:  0,
		BonusCoefficient: dotTickCoeff, // the report compares the spell's, which a pure DoT never reads

		DamageMultiplier: 1 + improvedCorruptionDamagePerRank*float64(warlock.Talents.ImprovedCorruption),
		ThreatMultiplier: 1,

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "Corruption-" + warlock.Label + strconv.Itoa(rank),
			},

			NumberOfTicks:    ticks,
			TickLength:       time.Second * 3,
			BonusCoefficient: dotTickCoeff,

			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
				dot.Snapshot(target, baseDamage, isRollover)
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
				return dot.CalcSnapshotDamage(sim, target, dot.Spell.OutcomeExpectedMagicAlwaysHit)
			} else {
				return spell.CalcPeriodicDamage(sim, target, baseDamage, spell.OutcomeExpectedMagicAlwaysHit)
			}
		},
	}
}

func (warlock *Warlock) registerCorruptionSpell() {
	warlock.Corruption = make([]*core.Spell, 0)

	maxRank := core.TernaryInt(core.IncludeAQ, CorruptionRanks, CorruptionRanks-1)
	for rank := 1; rank <= maxRank; rank++ {
		config := warlock.getCorruptionConfig(rank)

		if config.RequiredLevel <= int(warlock.Level) {
			warlock.Corruption = append(warlock.Corruption, warlock.GetOrRegisterSpell(config))
		}
	}
}
