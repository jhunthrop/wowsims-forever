package mage

import "github.com/wowsims/classic/sim/common/clientdamage"

// clientRoll is the {min, max} the client has a spell roll for this
// mage's level, from the generated own-level roll, PointsPerLevel, the
// spell's own level and MaxLevel (see clientdamage.Roll). Spell power,
// talents and every other modifier apply on top of it.
func (mage *Mage) clientRoll(ownLevelRoll []float64, pointsPerLevel float64, spellLevel, maxLevel int) [2]float64 {
	return clientdamage.Roll(ownLevelRoll, pointsPerLevel, spellLevel, maxLevel, int(mage.Level))
}
