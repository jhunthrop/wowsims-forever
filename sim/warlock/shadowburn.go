package warlock

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

const ShadowburnRanks = 6

// baseDamage was the classic tooltip roll (rank 6 {462, 514}) until
// the rotation-accuracy audit compared it against spellconst/
// warlock.json's own per-rank flat "amount" (rank 6, 18871, amount
// 266, sp_coefficient 0.429 unchanged, corroborated by wowhead's
// Forever page showing a single "Value: 267") - see shadowbolt.go's
// comment.
func (warlock *Warlock) registerShadowBurnBaseConfig(rank int) core.SpellConfig {
	spellId := [ShadowburnRanks + 1]int32{0, 17877, 18867, 18868, 18869, 18870, 18871}[rank]
	baseDamage := [ShadowburnRanks + 1]float64{0, 66, 80, 118, 148, 203, 266}[rank]
	manaCost := [ShadowburnRanks + 1]float64{0, 105, 130, 190, 245, 305, 365}[rank]
	// Rank 1 (17877) is learned at level 20 in the client's own data
	// (1.60.1.70009), not 15.
	level := [ShadowburnRanks + 1]int{0, 20, 24, 32, 40, 48, 56}[rank]

	spellCoeff := 0.429

	return core.SpellConfig{
		ActionID:      core.ActionID{SpellID: spellId},
		SpellCode:     SpellCode_WarlockShadowburn,
		SpellSchool:   core.SpellSchoolShadow,
		DefenseType:   core.DefenseTypeMagic,
		ProcMask:      core.ProcMaskSpellDamage,
		Flags:         core.SpellFlagAPL | core.SpellFlagResetAttackSwing | core.SpellFlagBinary | WarlockFlagDestruction,
		RequiredLevel: level,
		Rank:          rank,

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			CD: core.Cooldown{
				Timer:    warlock.NewTimer(),
				Duration: time.Second * time.Duration(15),
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: spellCoeff,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)
		},
	}
}

func (warlock *Warlock) registerShadowBurnSpell() {
	if !warlock.Talents.Shadowburn {
		return
	}

	warlock.Shadowburn = make([]*core.Spell, 0)
	for rank := 1; rank <= ShadowburnRanks; rank++ {
		config := warlock.registerShadowBurnBaseConfig(rank)

		if config.RequiredLevel <= int(warlock.Level) {
			warlock.Shadowburn = append(warlock.Shadowburn, warlock.GetOrRegisterSpell(config))
		}
	}
}
