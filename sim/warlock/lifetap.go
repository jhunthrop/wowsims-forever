package warlock

import (
	"github.com/wowsims/classic/sim/core"
)

const LifeTapRanks = 6

var LifeTapSpellId = [LifeTapRanks + 1]int32{0, 1454, 1455, 1456, 11687, 11688, 11689}

// LifeTapBaseDamage was a slightly-stale set of classic tooltip values
// ({30, 75, 140, 220, 310, 424}) until the rotation-accuracy audit
// compared it against spellconst/warlock.json's own per-rank flat
// "amount" (rank 6, 11689, amount 420 unchanged) - a small drift, not
// the ~2x halving the rest of the kit's nukes carried, but still a
// mismatch against the client's own number.
var LifeTapBaseDamage = [LifeTapRanks + 1]float64{0, 20, 65, 130, 210, 300, 420}

func (warlock *Warlock) getLifeTapBaseConfig(rank int) core.SpellConfig {
	spellId := LifeTapSpellId[rank]
	baseDamage := LifeTapBaseDamage[rank]
	spellCoef := [LifeTapRanks + 1]float64{0, 0.68, 0.8, 0.8, 0.8, 0.8, 0.8}[rank]

	level := [LifeTapRanks + 1]int{0, 6, 16, 26, 36, 46, 56}[rank]

	actionID := core.ActionID{SpellID: spellId}

	manaMetrics := warlock.NewManaMetrics(actionID)
	for _, pet := range warlock.BasePets {
		pet.LifeTapManaMetrics = pet.NewManaMetrics(actionID)
	}

	return core.SpellConfig{
		ActionID:      actionID,
		SpellSchool:   core.SpellSchoolShadow,
		SpellCode:     SpellCode_WarlockLifeTap,
		DefenseType:   core.DefenseTypeMagic,
		ProcMask:      core.ProcMaskSpellDamage,
		Flags:         core.SpellFlagAPL | core.SpellFlagResetAttackSwing | core.SpellFlagBinary | WarlockFlagAffliction,
		RequiredLevel: level,
		Rank:          rank,

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
		},

		BonusCoefficient: spellCoef,

		DamageMultiplier: 1 + 0.1*float64(warlock.Talents.ImprovedLifeTap),
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcDamage(sim, spell.Unit, baseDamage, spell.OutcomeAlwaysHit)
			restore := result.Damage

			if warlock.IsTanking() {
				spell.DealDamage(sim, result)
			}

			warlock.AddMana(sim, restore, manaMetrics)
		},
	}
}

func (warlock *Warlock) registerLifeTapSpell() {
	warlock.LifeTap = make([]*core.Spell, 0)
	for i := 1; i <= LifeTapRanks; i++ {
		config := warlock.getLifeTapBaseConfig(i)

		if config.RequiredLevel <= int(warlock.Level) {
			warlock.LifeTap = append(warlock.LifeTap, warlock.GetOrRegisterSpell(config))
		}
	}
}
