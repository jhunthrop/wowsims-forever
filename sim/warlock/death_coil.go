package warlock

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

const DeathCoilRanks = 3

func (warlock *Warlock) getDeathCoilBaseConfig(rank int) core.SpellConfig {
	spellId := [DeathCoilRanks + 1]int32{0, 6789, 17925, 17926}[rank]
	baseDamage := [DeathCoilRanks + 1]float64{0, 301, 375, 476}[rank]
	// The client's cost is 435/525/600 (spellconst/warlock.json, build
	// 1.60.1.70009), not 430/495/565 - conformance golden
	// sim/core/testdata/conformance/warlock.golden.md flagged each rank
	// as cost 435->430 / 525->495 / 600->565 (client->engine).
	//
	// Not modeled: the client's own duration_ms (3000) is the 3-second
	// Fear the target takes if the hit doesn't kill it (effect 1, aura
	// 7) - a crowd-control effect with no bearing on a raid boss's
	// damage taken, so this package (like the rest of sim/warlock) only
	// implements the damage-plus-self-heal half.
	manaCost := [DeathCoilRanks + 1]float64{0, 435, 525, 600}[rank]
	level := [DeathCoilRanks + 1]int{0, 42, 50, 58}[rank]
	spellCoeff := 0.214

	baseDamage *= 1 + warlock.shadowMasteryBonus()

	healingSpell := warlock.GetOrRegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: spellId}.WithTag(1),
		SpellSchool: core.SpellSchoolPhysical,
		ProcMask:    core.ProcMaskSpellHealing,
		Flags:       core.SpellFlagPassiveSpell | core.SpellFlagHelpful,

		DamageMultiplier: 1,
		ThreatMultiplier: 0,
	})

	return core.SpellConfig{
		SpellCode:     SpellCode_WarlockDeathCoil,
		ActionID:      core.ActionID{SpellID: spellId},
		SpellSchool:   core.SpellSchoolShadow,
		DefenseType:   core.DefenseTypeMagic,
		ProcMask:      core.ProcMaskSpellDamage,
		Flags:         core.SpellFlagAPL | core.SpellFlagResetAttackSwing | core.SpellFlagBinary | WarlockFlagAffliction,
		RequiredLevel: level,
		Rank:          rank,
		MissileSpeed:  24,

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			CD: core.Cooldown{
				Timer:    warlock.NewTimer(),
				Duration: time.Minute * 2,
			},
		},

		DamageMultiplierAdditive: 1,
		DamageMultiplier:         1,
		ThreatMultiplier:         1,
		BonusCoefficient:         spellCoeff,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			results := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)

			spell.WaitTravelTime(sim, func(s *core.Simulation) {
				spell.DealDamage(sim, results)
				if results.Landed() {
					healingSpell.CalcAndDealHealing(sim, healingSpell.Unit, results.Damage, healingSpell.OutcomeHealing)
				}
			})
		},
	}
}

func (warlock *Warlock) registerDeathCoilSpell() {
	warlock.DeathCoil = make([]*core.Spell, 0)
	for rank := 1; rank <= DeathCoilRanks; rank++ {
		config := warlock.getDeathCoilBaseConfig(rank)

		if config.RequiredLevel <= int(warlock.Level) {
			warlock.DeathCoil = append(warlock.DeathCoil, warlock.GetOrRegisterSpell(config))
		}
	}
}
