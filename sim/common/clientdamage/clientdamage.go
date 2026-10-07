// Package clientdamage carries the client's damage roll for one spell
// rank, in the same convention as spellconst.Spell.DamageRange: a centre
// (the effect's EffectBasePointsF at the spell's own level) that grows by
// PerLevel for every caster level above the spell's own (counting levels
// only up to MaxLevel, 0 meaning no cap), spread by Variance as the whole
// width of the roll, centre x (1 - Variance/2) through centre x
// (1 + Variance/2).
//
// Ability files keep one Effect per rank and ask it for the roll at the
// caster's level, so a hand table never restates a number the client
// already states. The tables are checked rank by rank against the
// vendored client file by each class package's spellconst_damage_test.go.
package clientdamage

import (
	"fmt"

	"github.com/wowsims/classic/sim/core"
)

// Effect is one rank's damage effect as the client states it.
type Effect struct {
	// Amount is the centre at the spell's own level.
	Amount float64
	// Variance is the whole width of the roll as a share of the centre.
	Variance float64
	// PerLevel is added to the centre per caster level above SpellLevel.
	PerLevel float64
	// SpellLevel is the level the spell is learned at.
	SpellLevel int
	// MaxLevel caps the levels that count towards PerLevel; 0 is no cap.
	MaxLevel int
}

// Center is the centre of the roll for a caster of the given level.
func (e Effect) Center(casterLevel int) float64 {
	if e.MaxLevel > 0 {
		casterLevel = min(casterLevel, e.MaxLevel)
	}
	center := e.Amount + e.PerLevel*float64(max(casterLevel-e.SpellLevel, 0))
	return max(center, 0)
}

// Range is the {min, max} of the roll for a caster of the given level,
// the form core.SpellConfig.ClientBaseDamage declares.
func (e Effect) Range(casterLevel int) [2]float64 {
	center := e.Center(casterLevel)
	return [2]float64{center * (1 - e.Variance/2), center * (1 + e.Variance/2)}
}

// Roll draws one hit from the roll. A flat effect (Variance 0) draws
// nothing from the sim's random stream.
func (e Effect) Roll(sim *core.Simulation, casterLevel int) float64 {
	bounds := e.Range(casterLevel)
	if bounds[0] == bounds[1] {
		return bounds[0]
	}
	return sim.Roll(bounds[0], bounds[1])
}

// FromRoll builds an Effect from a generated constants_auto_gen.go row:
// <Spell>BaseDamage[rank] is the client's {min, max} at the spell's own
// level, <Spell>PointsPerLevel[rank] and <Spell>MaxLevel[rank] its growth
// and cap, and <Spell>Level[rank] the spell's own level. A row that is not
// a two-ended roll is a programming error and panics at registration.
func FromRoll(roll []float64, perLevel float64, spellLevel, maxLevel int) Effect {
	if len(roll) != 2 {
		panic(fmt.Sprintf("clientdamage: a roll needs {min, max}, got %v", roll))
	}
	center := (roll[0] + roll[1]) / 2
	variance := 0.0
	if center > 0 {
		variance = (roll[1] - roll[0]) / center
	}
	return Effect{Amount: center, Variance: variance, PerLevel: perLevel, SpellLevel: spellLevel, MaxLevel: maxLevel}
}

// FromTable is FromRoll over a whole generated rank table: index i of the
// result is the Effect of rank i, so a spell whose damage the generator
// already carries reads one Effect per rank from its own constants.
func FromTable(rolls [][]float64, perLevel []float64, spellLevels, maxLevels []int) []Effect {
	if len(perLevel) != len(rolls) || len(spellLevels) != len(rolls) || len(maxLevels) != len(rolls) {
		panic("clientdamage: generated damage columns differ in length")
	}
	effects := make([]Effect, len(rolls))
	for rank := range rolls {
		effects[rank] = FromRoll(rolls[rank], perLevel[rank], spellLevels[rank], maxLevels[rank])
	}
	return effects
}
