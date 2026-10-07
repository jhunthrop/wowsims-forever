// Package clientdamage turns the client's own-level damage roll, as the
// generated constants_auto_gen.go files carry it, into the roll a caster
// of a given level makes. It is the one place the client's growth rule
// lives: the effect's centre grows by EffectRealPointsPerLevel for every
// caster level above the spell's own, counting levels only up to
// SpellLevels.MaxLevel (0 meaning no cap), and the roll keeps the width
// the client's Variance gives it. spellconst.Spell.DamageRange states the
// same rule against the raw client table; the tests pin the two together.
package clientdamage

// Roll returns the {min, max} a caster at casterLevel rolls for a spell
// whose roll at its own level is ownLevelRoll, before spell power,
// talents and any other modifier. A caster at or below the spell's level
// gets the own-level roll.
func Roll(ownLevelRoll []float64, pointsPerLevel float64, spellLevel, maxLevel, casterLevel int) [2]float64 {
	low, high := ownLevelRoll[0], ownLevelRoll[1]
	centre := (low + high) / 2
	if centre == 0 {
		return [2]float64{low, high}
	}
	grown := centre + pointsPerLevel*float64(levelsAboveSpell(spellLevel, maxLevel, casterLevel))
	scale := grown / centre
	return [2]float64{low * scale, high * scale}
}

// levelsAboveSpell is how many caster levels past the spell's own count
// towards the per-level growth.
func levelsAboveSpell(spellLevel, maxLevel, casterLevel int) int {
	if maxLevel > 0 && casterLevel > maxLevel {
		casterLevel = maxLevel
	}
	if casterLevel <= spellLevel {
		return 0
	}
	return casterLevel - spellLevel
}
