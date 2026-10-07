package priest

import (
	"github.com/wowsims/classic/sim/common/clientdamage"
)

// clientRoll is the {min, max} base damage this priest rolls for a spell
// whose own-level roll, growth per level, level and level cap are the
// generated tables' entries (constants_auto_gen.go): the client's amount
// at the priest's level, before spell power and talents.
func (priest *Priest) clientRoll(ownLevelRoll []float64, perLevel float64, spellLevel, maxLevel int) [2]float64 {
	return clientdamage.Roll(ownLevelRoll, perLevel, spellLevel, maxLevel, int(priest.Level))
}

// periodicTick is the amount a periodic effect deals per tick: the
// client states no variance for one, so both ends of the roll agree.
func periodicTick(roll [2]float64) float64 {
	return (roll[0] + roll[1]) / 2
}

// shadowWordPainRankTable is what one rank of Shadow Word: Pain reads.
// Ranks 1-7 are the generated ladder. Rank 8 cannot be: the client
// lists three spells under (Shadow Word: Pain, 8) - 10894, the player
// cast the rotations use (127 per tick, coefficient 0.2), 27605, and
// 1226589, a school-damage variant (142, coefficient 0.167) that the
// generator's tie-break (same spell level, higher id) picks for
// ShadowWordPain*[8]. The ranked ids stay the engine's, so rank 8 keeps
// 10894's own numbers here; TestShadowWordPainTicksMatchTheClient pins
// every rank to the client row at the engine's id.
type shadowWordPainTable struct {
	spellID  int32
	level    int
	manaCost float64
	coeff    float64
	tick     float64 // per-tick amount; the client states no growth or variance
}

const (
	shadowWordPainTopRank       = 8
	shadowWordPainTopRankSpell  = 10894
	shadowWordPainTopRankMana   = 470
	shadowWordPainTopRankTick   = 127
	shadowWordPainTopRankCoeff  = 0.2
	shadowWordPainTopRankLevel  = 58
	shadowWordPainBaseTickCount = 6 // 18 s at one tick per 3 s
)

func shadowWordPainRankTable(rank int) shadowWordPainTable {
	if rank == shadowWordPainTopRank {
		return shadowWordPainTable{
			spellID:  shadowWordPainTopRankSpell,
			level:    shadowWordPainTopRankLevel,
			manaCost: shadowWordPainTopRankMana,
			coeff:    shadowWordPainTopRankCoeff,
			tick:     shadowWordPainTopRankTick,
		}
	}
	return shadowWordPainTable{
		spellID:  ShadowWordPainSpellId[rank],
		level:    ShadowWordPainLevel[rank],
		manaCost: ShadowWordPainManaCost[rank],
		coeff:    ShadowWordPainSpellCoeff[rank],
		tick:     ShadowWordPainBaseDamage[rank][0],
	}
}

// tickRoll is the per-tick roll at any caster level (it does not grow).
func (table shadowWordPainTable) tickRoll(_ int) [2]float64 {
	return [2]float64{table.tick, table.tick}
}

// holyFireDotTickDamage is Holy Fire's periodic effect (effect 1 of the
// client row), a secondary effect the generated ladder does not carry:
// the amount per tick at every caster level (no growth, no variance).
// TestHolyFireDotTicksMatchTheClient pins it to the client.
var holyFireDotTickDamage = [HolyFireRanks + 1]float64{0, 4, 5, 6, 7, 13, 10, 13, 15}

const (
	holyFireDotCoefficient = 0.05
	holyFireDotTickMS      = 2000
	holyFireDotTicks       = 5 // 10 s at one tick per 2 s
)

// devouringPlagueTicks is the client's 24 s duration at one tick per 3 s.
const devouringPlagueTicks = 8
