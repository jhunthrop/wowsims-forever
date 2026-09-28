package warlock

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// ConflagrateRanks was 4 (the client's ranks 3-6, all classic ids,
// their baseDamage a known Classic tooltip roll rather than this
// pipeline's flat spellconst "amount" - Conflagrate's real mechanic
// consumes a percentage of the target's remaining Immolate damage, a
// server-scripted behavior this package does not model at all, so its
// flat roll was already a "keep Classic's known behavior" stand-in
// before this comment). spellranks.json's own "Conflagrate" chain
// (build 1.60.1.70009) carries two MORE ranks below that: 1293817
// (rank 1, level 25) and 1293818 (rank 2, level 32) - Forever ids with
// no Classic precedent, so there is no known tooltip roll to fall back
// on for them. Registered here as a flat (non-rolled) base damage
// taken directly from spellconst/warlock.json's own "amount" (95 and
// 122), the same convention sim/warlock/incinerate.go uses for a
// Forever-original ability with a single scalar effect - until a
// Forever combat log gives a real min/max to replace it with.
const ConflagrateRanks = 6

func (warlock *Warlock) getConflagrateConfig(rank int) core.SpellConfig {
	spellId := [ConflagrateRanks + 1]int32{0, 1293817, 1293818, 17962, 18930, 18931, 18932}[rank]
	baseDamageMin := [ConflagrateRanks + 1]float64{0, 95, 122, 249, 319, 395, 447}[rank]
	baseDamageMax := [ConflagrateRanks + 1]float64{0, 95, 122, 316, 400, 491, 557}[rank]
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
			baseDamage := sim.Roll(baseDamageMin, baseDamageMax)

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
