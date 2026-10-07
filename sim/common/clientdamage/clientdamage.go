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

import "github.com/wowsims/classic/sim/core"

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
