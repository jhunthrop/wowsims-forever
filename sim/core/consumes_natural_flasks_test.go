package core

import (
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// The Natural flasks (items 274273-274276, spells 1293740-1293743) as the
// 1.60.1.70790 server push 113570 leaves them: 30 Stamina (aura 29, misc 2)
// everywhere and a 2 (the dummy aura 4 row) of the flask's own stat in
// Mount Hyjal, Hyjal Summit and the Barrow Deeps, which the raid preset
// fights in. The shipped rows read 60 and 5 (4 for Aggression); Wowhead's
// tooltip still renders those.
func TestNaturalFlasksMatchThePush(t *testing.T) {
	want := map[proto.Flask]stats.Stats{
		proto.Flask_FlaskOfNaturalAccuracy:   {stats.Stamina: 30, stats.Hit: 2 * HitRatingPerHitChance},
		proto.Flask_FlaskOfNaturalAggression: {stats.Stamina: 30, stats.Crit: 2 * CritRatingPerCritChance},
		proto.Flask_FlaskOfNaturalPrecision:  {stats.Stamina: 30, stats.Expertise: 2 * ExpertiseRatingPerExpertiseChance},
		proto.Flask_FlaskOfNaturalSwiftness: {
			stats.Stamina:    30,
			stats.MeleeHaste: 2 * HasteRatingPerHastePercent,
			stats.SpellHaste: 2 * HasteRatingPerHastePercent,
		},
	}
	for flask, expected := range want {
		if got := naturalFlaskStats[flask]; got != expected {
			t.Errorf("%v gives %v, want %v", flask, got, expected)
		}
	}
}

// A Natural flask reaches the character through its Consumes message.
func TestNaturalFlaskApplies(t *testing.T) {
	character := newGnomeEurekaTestCaster(proto.Class_ClassPriest)
	before := character.GetStats()
	applyFlaskConsumes(character, &proto.Consumes{Flask: proto.Flask_FlaskOfNaturalAggression})
	after := character.GetStats()
	if got := after[stats.Crit] - before[stats.Crit]; got != 2*CritRatingPerCritChance {
		t.Errorf("Flask of Natural Aggression adds %v crit, want %v", got, 2*CritRatingPerCritChance)
	}
}
