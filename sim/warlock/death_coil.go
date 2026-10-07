package warlock

import (
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

const DeathCoilRanks = 3

// DeathCoilDamage is spellconst/warlock.json's own health-leech amount
// for ids 6789, 17925 and 17926 (rank 3: 454 at level 58, growing 3 a level to
// level 64), where the table it replaces carried the Classic 301/375/476.
var DeathCoilDamage = [DeathCoilRanks + 1]clientdamage.Effect{
	{},
	{Amount: 272, PerLevel: 2.2, SpellLevel: 42, MaxLevel: 48},
	{Amount: 359, PerLevel: 2.6, SpellLevel: 50, MaxLevel: 56},
	{Amount: 454, PerLevel: 3, SpellLevel: 58, MaxLevel: 64},
}

func (warlock *Warlock) getDeathCoilBaseConfig(rank int) core.SpellConfig {
	spellId := [DeathCoilRanks + 1]int32{0, 6789, 17925, 17926}[rank]
	damage := DeathCoilDamage[rank]
	casterLevel := int(warlock.Level)
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

	shadowMastery := 1 + warlock.shadowMasteryBonus()

	healingSpell := warlock.GetOrRegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: spellId}.WithTag(1),
		SpellSchool: core.SpellSchoolPhysical,
		ProcMask:    core.ProcMaskSpellHealing,
		Flags:       core.SpellFlagPassiveSpell | core.SpellFlagHelpful,

		DamageMultiplier: 1,
		ThreatMultiplier: 0,
	})

	return core.SpellConfig{
		SpellCode:        SpellCode_WarlockDeathCoil,
		ActionID:         core.ActionID{SpellID: spellId},
		SpellSchool:      core.SpellSchoolShadow,
		DefenseType:      core.DefenseTypeMagic,
		ProcMask:         core.ProcMaskSpellDamage,
		Flags:            core.SpellFlagAPL | core.SpellFlagResetAttackSwing | core.SpellFlagBinary | WarlockFlagAffliction,
		RequiredLevel:    level,
		Rank:             rank,
		ClientBaseDamage: damage.Range(casterLevel),
		MissileSpeed:     24,

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
			results := spell.CalcDamage(sim, target, damage.Roll(sim, casterLevel)*shadowMastery, spell.OutcomeMagicHitAndCrit)

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
