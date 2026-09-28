package warlock

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Incinerate is Destruction's level-40 talent-granted filler
// (talents/warlock.json node 105874, prereq Bane of Havoc rank 1): a
// castable, ranked direct-damage bolt that deals extra damage while
// Immolate is up on the target. Talent text (rank 1): "Deals 97 Fire
// damage to your target and an additional 25% damage if the target is
// afflicted by Immolate."
//
// spellranks.json lists the castable chain: 412758 (rank 1, level 40),
// 1293812 (rank 2, level 50), 1293813 (rank 3, level 60) -- the talent
// grants the ability, but like any other warlock nuke it then has
// normal per-level ranks.
//
// Client numbers (spellconst/warlock.json), all three ranks share
// cast_time_ms 2500, gcd_ms 1500, cooldown_ms 0:
//   - rank 1 (412758): cost 205, amount 97, sp_coefficient 0.714
//   - rank 2 (1293812): cost 265, amount 145, sp_coefficient 0.714
//   - rank 3 (1293813): cost 325, amount 217, sp_coefficient 0.714
//   - every rank's effect 1 is effect=3 (Dummy), amount 25 -- the flat
//     "+25% if Immolate" from the talent text, present unchanged at
//     every rank.
//
// The pipeline's spellconst effects carry no dice/variance columns (see
// sim/core/spellconst's Effect shape), and wowhead's Forever page for
// 412758 shows a single base value with no min-max tooltip either, so
// Incinerate is implemented as a flat per-rank amount (no sim.Roll),
// the same convention Wrack's DoT tick uses, rather than the classic
// min/max-roll convention ShadowBolt/SearingPain use for their
// pre-Forever ranks.
const IncinerateRanks = 3

func (warlock *Warlock) getIncinerateBaseConfig(rank int) core.SpellConfig {
	spellId := [IncinerateRanks + 1]int32{0, 412758, 1293812, 1293813}[rank]
	baseDamage := [IncinerateRanks + 1]float64{0, 97, 145, 217}[rank]
	manaCost := [IncinerateRanks + 1]float64{0, 205, 265, 325}[rank]
	level := [IncinerateRanks + 1]int{0, 40, 50, 60}[rank]
	spellCoeff := 0.71399998665

	const immolateBonusMultiplier = 1.25

	return core.SpellConfig{
		SpellCode:     SpellCode_WarlockIncinerate,
		ActionID:      core.ActionID{SpellID: spellId},
		SpellSchool:   core.SpellSchoolFire,
		DefenseType:   core.DefenseTypeMagic,
		ProcMask:      core.ProcMaskSpellDamage,
		Flags:         core.SpellFlagAPL | core.SpellFlagResetAttackSwing | WarlockFlagDestruction,
		RequiredLevel: level,
		Rank:          rank,

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD:      core.GCDDefault,
				CastTime: time.Millisecond * 2500,
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: spellCoeff,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			oldMultiplier := spell.DamageMultiplier
			if warlock.getActiveImmolateSpell(target) != nil {
				spell.DamageMultiplier *= immolateBonusMultiplier
			}

			result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)
			spell.DamageMultiplier = oldMultiplier

			spell.DealDamage(sim, result)
		},
	}
}

func (warlock *Warlock) registerIncinerateSpell() {
	if !warlock.Talents.Incinerate {
		return
	}

	warlock.Incinerate = make([]*core.Spell, 0)
	for rank := 1; rank <= IncinerateRanks; rank++ {
		config := warlock.getIncinerateBaseConfig(rank)

		if config.RequiredLevel <= int(warlock.Level) {
			warlock.Incinerate = append(warlock.Incinerate, warlock.GetOrRegisterSpell(config))
		}
	}
}
