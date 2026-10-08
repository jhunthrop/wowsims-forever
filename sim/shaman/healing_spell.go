package shaman

import (
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

// healingRank is one learnable rank of a direct heal, read from the
// generated client tables (constants_auto_gen.go) except where an ability
// file says its table is wrong.
type healingRank struct {
	rank        int
	spellID     int32
	level       int
	manaCost    float64
	castTime    time.Duration
	coefficient float64
	healing     clientdamage.Effect
}

// castTimeFromMS turns a generated cast time column into a duration.
func castTimeFromMS(ms int32) time.Duration {
	return time.Duration(ms) * time.Millisecond
}

// RegisterHealingSpells registers every healing spell the shaman has
// learned at its level. Only the Restoration spec casts them, so the spec
// calls this after Initialize rather than every shaman paying for them.
func (shaman *Shaman) RegisterHealingSpells() {
	shaman.registerHealingWaveSpell()
	shaman.registerLesserHealingWaveSpell()
	shaman.registerChainHealSpell()
	shaman.registerRiptideSpell()
	shaman.registerManaTideTotemSpell()
}

// newHealingSpellConfig is what every direct heal shares: a Nature spell
// that costs flat mana, takes the global cooldown, can crit, and heals its
// target for the client's roll plus its spell-power coefficient.
func (shaman *Shaman) newHealingSpellConfig(rank healingRank, classMask uint64) core.SpellConfig {
	casterLevel := int(shaman.Level)

	return core.SpellConfig{
		ActionID:       core.ActionID{SpellID: rank.spellID},
		SpellSchool:    core.SpellSchoolNature,
		DefenseType:    core.DefenseTypeMagic,
		ProcMask:       core.ProcMaskSpellHealing,
		Flags:          SpellFlagShaman | core.SpellFlagHelpful | core.SpellFlagAPL,
		ClassSpellMask: classMask,
		RequiredLevel:  rank.level,
		Rank:           rank.rank,

		ManaCost: core.ManaCostOptions{
			FlatCost:   rank.manaCost,
			Multiplier: 100,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD:      core.GCDDefault,
				CastTime: rank.castTime,
			},
		},

		BonusCoefficient: rank.coefficient,
		ClientBaseDamage: rank.healing.Range(casterLevel),

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.CalcAndDealHealing(sim, target, rank.healing.Roll(sim, casterLevel), spell.OutcomeHealingCrit)
		},
	}
}

// registerHealingRanks registers each rank the shaman's level has learned
// into a slice indexed by rank, like the damage spells' slices.
func (shaman *Shaman) registerHealingRanks(ranks []healingRank, newConfig func(healingRank) core.SpellConfig) []*core.Spell {
	spells := make([]*core.Spell, len(ranks)+1)
	for _, rank := range ranks {
		if rank.level <= int(shaman.Level) {
			spells[rank.rank] = shaman.RegisterSpell(newConfig(rank))
		}
	}
	return spells
}
