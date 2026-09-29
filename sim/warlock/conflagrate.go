package warlock

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// ConflagrateRanks was 4 (the client's ranks 3-6, all classic ids,
// their baseDamage a known Classic tooltip roll {249,316}/{319,400}/
// {395,491}/{447,557} - Conflagrate's real mechanic consumes a
// percentage of the target's remaining Immolate damage, a
// server-scripted behavior this package does not model at all).
// Rotation-accuracy audit: spellconst/warlock.json's own "amount" for
// 17962/18930/18931/18932 is a single flat 146/194/239/282 (rank 6,
// 18932, corroborated by wowhead's Forever page showing a single
// "Value: 283" with no min-max tooltip), not that classic roll's
// ~280-500 average - the same halving shadowbolt.go's comment
// documents across the rest of the kit. spellranks.json's own
// "Conflagrate" chain (build 1.60.1.70009) carries two MORE ranks
// below that: 1293817 (rank 1, level 25) and 1293818 (rank 2, level
// 32) - Forever ids with no Classic precedent, registered flat from
// spellconst's "amount" (95 and 122) since day one. All six ranks are
// now the same flat-amount convention sim/warlock/incinerate.go uses.
const ConflagrateRanks = 6

func (warlock *Warlock) getConflagrateConfig(rank int) core.SpellConfig {
	spellId := [ConflagrateRanks + 1]int32{0, 1293817, 1293818, 17962, 18930, 18931, 18932}[rank]
	baseDamage := [ConflagrateRanks + 1]float64{0, 95, 122, 146, 194, 239, 282}[rank]
	manaCost := [ConflagrateRanks + 1]float64{0, 100, 130, 165, 200, 230, 255}[rank]
	// Ranks 3-6 here are the client's ranks 3-6 (17962/18930/18931/18932);
	// rank 3 is learned at level 40 in the client's own data
	// (1.60.1.70009), not 0.
	level := [ConflagrateRanks + 1]int{0, 25, 32, 40, 48, 54, 60}[rank]

	spCoeff := 0.429

	return core.SpellConfig{
		SpellCode:     SpellCode_WarlockConflagrate,
		ActionID:      core.ActionID{SpellID: spellId},
		SpellSchool:   core.SpellSchoolFire,
		DefenseType:   core.DefenseTypeMagic,
		ProcMask:      core.ProcMaskSpellDamage,
		Flags:         core.SpellFlagAPL | WarlockFlagDestruction,
		Rank:          rank,
		RequiredLevel: level,

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
			spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)

			immoSpell := warlock.getActiveImmolateSpell(target)
			if immoSpell != nil {
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
