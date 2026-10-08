package item_sets

import (
	"fmt"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

// A rating bonus is the client's combat-rating aura: a rating for the
// melee, ranged and spell halves of one stat. The generic flat decoder in
// core reads only the expertise rating, so the Forever sets that state hit
// or crit as a rating are modelled here.

const clientAuraModRating int32 = 189

// The client's rating masks: one bit per combat rating (hit is melee 5,
// ranged 6, spell 7; crit is melee 8, ranged 9, spell 10).
const (
	clientRatingHit  int32 = 1<<5 | 1<<6 | 1<<7
	clientRatingCrit int32 = 1<<8 | 1<<9 | 1<<10
)

// Level 60 combat rating conversions: the ratings one percent of the stat
// takes. They agree with the names of the rows that use them ("Increased
// Hit Chance 00.5" is 5 hit rating, "Increased Critical 1.5 - All" is 21
// crit rating).
const (
	hitRatingPerPercent  = 10.0
	critRatingPerPercent = 14.0
)

type ratingKind struct {
	stat       stats.Stat
	perPercent float64
	engineUnit float64
}

var ratingKinds = map[int32]ratingKind{
	clientRatingHit:  {stat: stats.Hit, perPercent: hitRatingPerPercent, engineUnit: core.HitRatingPerHitChance},
	clientRatingCrit: {stat: stats.Crit, perPercent: critRatingPerPercent, engineUnit: core.CritRatingPerCritChance},
}

// ratingBonus raises hit or crit by the bonus's combat rating.
func ratingBonus(bonusID int32) core.ApplyEffect {
	effect := auraEffectOf(bonusID, core.MustClientSpellRow(bonusID), clientAuraModRating)
	kind, ok := ratingKinds[effect.Misc0]
	if !ok {
		panic(fmt.Sprintf("item_sets: client spell %d rates combat rating mask %d, which has no model", bonusID, effect.Misc0))
	}
	gain := effect.Points / kind.perPercent * kind.engineUnit
	return func(agent core.Agent) {
		agent.GetCharacter().AddStat(kind.stat, gain)
	}
}
