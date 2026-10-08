package paladin

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// blessingDuration is the client duration (3600000 ms) of every blessing
// below, Greater versions included.
const blessingDuration = time.Hour

// blessingSpell is one castable blessing as the client teaches it.
type blessingSpell struct {
	spellID int32
	level   int
	// manaCost is the flat mana cost; zero for a blessing priced as a
	// percentage of base mana.
	manaCost float64
	// baseManaPct is the cost as a percentage of base mana (client
	// cost_pct); zero for a flat-cost blessing.
	baseManaPct float64
	// amount is the effect's base points: attack power for Might, mana per
	// five seconds for Wisdom, the stat percentage for Kings and the threat
	// percentage for Salvation.
	amount float64
}

func (spell blessingSpell) manaCostOptions() core.ManaCostOptions {
	if spell.baseManaPct > 0 {
		return core.ManaCostOptions{BaseCost: spell.baseManaPct / 100}
	}
	return core.ManaCostOptions{FlatCost: spell.manaCost}
}

// rankedBlessingSpells pairs the core rank table (spell id, level and
// amount, shared with the raid-buff option) with the client's mana costs.
func rankedBlessingSpells(ranks core.BuffRanks, manaCosts []float64) []blessingSpell {
	spells := make([]blessingSpell, len(ranks))
	for i, rank := range ranks {
		spells[i] = blessingSpell{spellID: rank.SpellID, level: rank.Level, manaCost: manaCosts[i], amount: rank.Amount}
	}
	return spells
}

// greaterBlessingSpell is a Greater rank: it takes the amount of the
// single-target rank (index into ranks) that the client states for it.
func greaterBlessingSpell(spellID int32, ranks core.BuffRanks, rankIndex int, manaCost float64) blessingSpell {
	rank := ranks[rankIndex]
	return blessingSpell{spellID: spellID, level: rank.Level, manaCost: manaCost, amount: rank.Amount}
}

const (
	// blessingOfKingsStatBonus is Blessing of Kings' effect 0 (spell 20217,
	// aura 137 "modify total stat percentage", amount 10).
	blessingOfKingsStatBonus = 10
	// blessingOfSalvationThreatReduction is Blessing of Salvation's effect 0
	// (spell 1038, aura 10 "modify threat", amount -30), as a positive
	// percentage.
	blessingOfSalvationThreatReduction = 30
)

// Trainables "Blessing of Might" (19740...25291), mana costs 20 to 130.
var blessingOfMightSpells = rankedBlessingSpells(core.BlessingOfMightRanks,
	[]float64{20, 30, 45, 60, 85, 110, 130})

// Trainables "Greater Blessing of Might": 25782 (level 52, 220 mana) and
// 25916 (level 60, 260 mana) carry the amounts of ranks 6 and 7.
var greaterBlessingOfMightSpells = []blessingSpell{
	greaterBlessingSpell(25782, core.BlessingOfMightRanks, 5, 220),
	greaterBlessingSpell(25916, core.BlessingOfMightRanks, 6, 260),
}

// Trainables "Blessing of Wisdom" (19742...25290), mana costs 30 to 125.
var blessingOfWisdomSpells = rankedBlessingSpells(core.BlessingOfWisdomRanks,
	[]float64{30, 45, 65, 90, 115, 125})

// Trainables "Greater Blessing of Wisdom": 25894 (level 54, 230 mana) and
// 25918 (level 60, 250 mana) carry the amounts of ranks 5 and 6.
var greaterBlessingOfWisdomSpells = []blessingSpell{
	greaterBlessingSpell(25894, core.BlessingOfWisdomRanks, 4, 230),
	greaterBlessingSpell(25918, core.BlessingOfWisdomRanks, 5, 250),
}

// Blessing of Kings: 20217 (level 20, 8% of base mana) and Greater 25898
// (level 60, 150 mana).
var (
	blessingOfKingsSpells = []blessingSpell{
		{spellID: 20217, level: 20, baseManaPct: 8, amount: blessingOfKingsStatBonus},
	}
	greaterBlessingOfKingsSpells = []blessingSpell{
		{spellID: 25898, level: 60, manaCost: 150, amount: blessingOfKingsStatBonus},
	}
)

// Blessing of Salvation: 1038 (level 26, 8% of base mana) and Greater 25895
// (level 60, 16% of base mana).
var (
	blessingOfSalvationSpells = []blessingSpell{
		{spellID: 1038, level: 26, baseManaPct: 8, amount: blessingOfSalvationThreatReduction},
	}
	greaterBlessingOfSalvationSpells = []blessingSpell{
		{spellID: 25895, level: 60, baseManaPct: 16, amount: blessingOfSalvationThreatReduction},
	}
)
