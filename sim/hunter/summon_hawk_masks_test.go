package hunter

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
)

// Round 10 (physical lane): Summon Hawk (1293241) carries class mask 2 bit
// 8192. Mortal Shots (19485, aura 108 crit damage, 424449 / 2147483648 /
// 268435968 / 8388608) does not name that bit; Ferocity (19598, aura 107
// crit chance, 1073741824 / 0 / 8192) does, and its text reads "your pets
// and hawks", as does Unleashed Fury's ("damage done by your pets and
// hawks"). The engine gave the hawk Mortal Shots and neither of the two
// hawk talents.
func TestSummonHawkTakesTheHawkTalentsNotMortalShots(t *testing.T) {
	talents := buildTalentsStringFromRanks(t, map[string]int{
		"summon_hawk": 1, "mortal_shots": 5, "ferocity": 5, "unleashed_fury": 5,
	})
	hunter := buildHunterWithTalents(t, 60, talents, proto.Hunter_Options_Cat)
	hawk := hunter.SummonHawk
	if hawk == nil {
		t.Fatal("Summon Hawk is not registered")
	}
	if hawk.CritDamageBonus != 1 {
		t.Errorf("Summon Hawk crit damage bonus = %v, want 1 (Mortal Shots does not name the hawk)", hawk.CritDamageBonus)
	}
	if want := ferocityCritPerRank * 5; math.Abs(hawk.BonusCritRating-want) > 1e-9 {
		t.Errorf("Summon Hawk bonus crit = %v, want Ferocity's %v", hawk.BonusCritRating, want)
	}
	if want := 1 + unleashedFuryPerRank*5; math.Abs(hawk.DamageMultiplier-want) > 1e-9 {
		t.Errorf("Summon Hawk damage multiplier = %v, want Unleashed Fury's %v", hawk.DamageMultiplier, want)
	}
}
