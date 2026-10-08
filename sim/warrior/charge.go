package warrior

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// chargeRank is one trainer rank of Charge: the client's trainable ids
// (100, 6178, 11578), not the generator's ChargeSpellId, whose dedup kept
// the 1240287-1240289 reissues of ranks 1-3 and an unrelated rank 0 (the
// level 25 Season of Discovery rune 411684).
type chargeRank struct {
	spellID int32
	level   int
	// rage is effect 1 (SPELL_EFFECT_ENERGIZE, rage) in whole points: the
	// client states it in tenths, 90, 120 and 150.
	rage float64
}

var chargeRanks = [...]chargeRank{
	{spellID: 100, level: 4, rage: 9},
	{spellID: 6178, level: 26, rage: 12},
	{spellID: 11578, level: 46, rage: 15},
}

// chargeCooldown is the client's category cooldown, 15 seconds, on every
// rank (cooldown_ms 0, category_cooldown_ms 15000).
const chargeCooldown = 15 * time.Second

// registerChargeSpell registers the highest Charge rank the warrior has
// learned. This sim has no movement primitive, so the gap-closer itself
// is not modelled: the cast grants the rage and nothing else, and can only
// be made before the fight starts, as the game allows it only out of
// combat. Improved Charge's extra rage is not added here because
// applyImprovedCharge already grants it once at the pull, standing in for
// this same opening Charge.
func (warrior *Warrior) registerChargeSpell() {
	var rank chargeRank
	rankNumber := 0
	for i, candidate := range chargeRanks {
		if int(warrior.Level) >= candidate.level {
			rank, rankNumber = candidate, i+1
		}
	}
	if rankNumber == 0 {
		return
	}

	rageMetrics := warrior.NewRageMetrics(core.ActionID{SpellID: rank.spellID})

	warrior.Charge = warrior.RegisterSpell(BattleStance, core.SpellConfig{
		ActionID:    core.ActionID{SpellID: rank.spellID},
		SpellSchool: core.SpellSchoolPhysical,
		Flags:       core.SpellFlagAPL | core.SpellFlagNoOnCastComplete,

		RequiredLevel: rank.level,
		Rank:          rankNumber,

		Cast: core.CastConfig{
			CD: core.Cooldown{
				Timer:    warrior.NewTimer(),
				Duration: chargeCooldown,
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, _ *core.Unit) bool {
			return sim.CurrentTime <= 0
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			warrior.AddRage(sim, rank.rage, rageMetrics)
		},
	})
}
