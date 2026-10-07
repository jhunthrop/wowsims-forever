package warlock

import (
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

// ConflagrateRanks is six: Forever's ids 1293817 (rank 1, level 25) and
// 1293818 (rank 2, level 32) below the client's ranks 3-6 (17962, 18930,
// 18931, 18932; rank 3 is learned at level 40 in the client's own data,
// 1.60.1.70009, not 0). Conflagrate's real mechanic consumes a percentage of
// the target's remaining Immolate damage, a server-scripted behaviour this
// package does not model at all. ConflagrateDamage is spellconst/
// warlock.json's own roll for those ids (rank 6 rolls 251.1-312.9 at level
// 60: a centre of 282 at its own level, growing 1.3 a level, 0.2188 wide),
// where the Classic tooltip roll was {447,557} - see shadowbolt.go's comment.
var ConflagrateDamage = [ConflagrateRanks + 1]clientdamage.Effect{
	{},
	{Amount: 95, Variance: 0.241758, PerLevel: 0.9, SpellLevel: 25, MaxLevel: 30},
	{Amount: 122, Variance: 0.241758, PerLevel: 0.9, SpellLevel: 32, MaxLevel: 38},
	{Amount: 146, Variance: 0.241758, PerLevel: 1, SpellLevel: 40, MaxLevel: 46},
	{Amount: 194, Variance: 0.224719, PerLevel: 1.1, SpellLevel: 48, MaxLevel: 54},
	{Amount: 239, Variance: 0.222738, PerLevel: 1.2, SpellLevel: 54, MaxLevel: 60},
	{Amount: 282, Variance: 0.219124, PerLevel: 1.3, SpellLevel: 60, MaxLevel: 66},
}

const ConflagrateRanks = 6

func (warlock *Warlock) getConflagrateConfig(rank int) core.SpellConfig {
	spellId := [ConflagrateRanks + 1]int32{0, 1293817, 1293818, 17962, 18930, 18931, 18932}[rank]
	damage := ConflagrateDamage[rank]
	casterLevel := int(warlock.Level)
	manaCost := [ConflagrateRanks + 1]float64{0, 100, 130, 165, 200, 230, 255}[rank]
	// Ranks 3-6 here are the client's ranks 3-6 (17962/18930/18931/18932);
	// rank 3 is learned at level 40 in the client's own data
	// (1.60.1.70009), not 0.
	level := [ConflagrateRanks + 1]int{0, 25, 32, 40, 48, 54, 60}[rank]

	spCoeff := 0.429

	return core.SpellConfig{
		SpellCode:        SpellCode_WarlockConflagrate,
		ActionID:         core.ActionID{SpellID: spellId},
		SpellSchool:      core.SpellSchoolFire,
		DefenseType:      core.DefenseTypeMagic,
		ProcMask:         core.ProcMaskSpellDamage,
		Flags:            core.SpellFlagAPL | WarlockFlagDestruction,
		Rank:             rank,
		ClientBaseDamage: damage.Range(casterLevel),
		RequiredLevel:    level,

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			CD: core.Cooldown{
				Timer:    warlock.NewTimer(),
				Duration: time.Second * 10,
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return warlock.getActiveImmolateSpell(target) != nil
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: spCoeff,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcAndDealDamage(sim, target, damage.Roll(sim, casterLevel), spell.OutcomeMagicHitAndCrit)

			if result.Landed() {
				// Shadow and Flame (talents.go): hitting with
				// Conflagrate buffs the warlock's own Shadow damage.
				warlock.triggerShadowAndFlame(sim, SpellCode_WarlockConflagrate)
			}

			immoSpell := warlock.getActiveImmolateSpell(target)
			if immoSpell != nil && !warlock.shadowAndFlamePreservesImmolate(sim) {
				immoSpell.Dot(target).Deactivate(sim)
			}
		},
	}
}

func (warlock *Warlock) registerConflagrateSpell() {
	if !warlock.Talents.Conflagrate {
		return
	}

	warlock.Conflagrate = make([]*core.Spell, 0)
	for rank := 1; rank <= ConflagrateRanks; rank++ {
		config := warlock.getConflagrateConfig(rank)

		if config.RequiredLevel <= int(warlock.Level) {
			warlock.Conflagrate = append(warlock.Conflagrate, warlock.GetOrRegisterSpell(config))
		}
	}
}
