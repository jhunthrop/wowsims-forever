package warlock

import (
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

const ShadowburnRanks = 6

// ShadowburnDamage is spellconst/warlock.json's own roll for ids 17877
// through 18871 (rank 6 rolls 258.3-288.1 at level 60: a centre of 266 at its
// own level, growing 1.8 a level, 0.112 wide; sp_coefficient 0.429), where
// the classic tooltip roll was {462, 514} - see shadowbolt.go's comment.
var ShadowburnDamage = [ShadowburnRanks + 1]clientdamage.Effect{
	{},
	{Amount: 66, Variance: 0.129032, PerLevel: 0.9, SpellLevel: 20, MaxLevel: 24},
	{Amount: 80, Variance: 0.130081, PerLevel: 1, SpellLevel: 24, MaxLevel: 30},
	{Amount: 118, Variance: 0.121212, PerLevel: 1.3, SpellLevel: 32, MaxLevel: 38},
	{Amount: 148, Variance: 0.115523, PerLevel: 1.3, SpellLevel: 40, MaxLevel: 46},
	{Amount: 203, Variance: 0.113208, PerLevel: 1.6, SpellLevel: 48, MaxLevel: 54},
	{Amount: 266, Variance: 0.109244, PerLevel: 1.8, SpellLevel: 56, MaxLevel: 62},
}

func (warlock *Warlock) registerShadowBurnBaseConfig(rank int) core.SpellConfig {
	spellId := [ShadowburnRanks + 1]int32{0, 17877, 18867, 18868, 18869, 18870, 18871}[rank]
	damage := ShadowburnDamage[rank]
	casterLevel := int(warlock.Level)
	manaCost := [ShadowburnRanks + 1]float64{0, 105, 130, 190, 245, 305, 365}[rank]
	// Rank 1 (17877) is learned at level 20 in the client's own data
	// (1.60.1.70009), not 15.
	level := [ShadowburnRanks + 1]int{0, 20, 24, 32, 40, 48, 56}[rank]

	spellCoeff := 0.429

	return core.SpellConfig{
		ActionID:         core.ActionID{SpellID: spellId},
		SpellCode:        SpellCode_WarlockShadowburn,
		SpellSchool:      core.SpellSchoolShadow,
		DefenseType:      core.DefenseTypeMagic,
		ProcMask:         core.ProcMaskSpellDamage,
		Flags:            core.SpellFlagAPL | core.SpellFlagResetAttackSwing | core.SpellFlagBinary | WarlockFlagDestruction,
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
			CD: core.Cooldown{
				Timer:    warlock.NewTimer(),
				Duration: time.Second * time.Duration(15),
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: spellCoeff,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcAndDealDamage(sim, target, damage.Roll(sim, casterLevel), spell.OutcomeMagicHitAndCrit)

			if result.Landed() {
				// Shadow and Flame (talents.go): hitting with
				// Shadowburn buffs the warlock's own Fire damage. The
				// talent's other half - a chance to instantly refund a
				// Soul Shard - is not modelled: this fork's Shadowburn
				// costs mana, not Soul Shards (see soul_fire.go's own
				// comment on the same point), and no Soul Shard
				// resource exists anywhere in this package.
				warlock.triggerShadowAndFlame(sim, SpellCode_WarlockShadowburn)
			}
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
