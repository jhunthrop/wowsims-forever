package core

import (
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// TestBuildStatWeightRequestsScalesHitLikeArmor pins this lane's fix
// (Forever's own leveling-bis ranker was publishing "hit" as weight 0,
// error 0 - "no effect" - for every physical spec: hunter-beast-
// mastery, hunter-marksmanship, warrior-arms, warrior-fury, druid-
// feral and rogue-assassination, at every band).
//
// The root cause is not a real hard cap: HitRatingPerHitChance is 1,
// so one raw point of Hit is exactly 1% hit chance
// (spell_result.go's PhysicalHitChance), and NewAttackTable
// (target.go) sets HitSuppression to (targetDefense-weaponSkill-10)*
// 0.002, which comes out to exactly 0.01 (1%) for the ordinary
// "3 levels higher" gap this package's default encounter fights. A
// character with no other source of Hit has its first raw point
// entirely cancelled by that suppression - the low direction
// (-defaultStatMod) can't push an already-floored hit chance any
// lower, and the high direction (+defaultStatMod) lands exactly back
// on the floor too, so runStatWeights measures bit-identical DPS in
// both directions and computeStatWeights' hard-cap detector
// (modPlayerHigh.Dps.Avg == baselinePlayer.Dps.Avg) reads that as a
// real cap - confirmed directly for hunter-marksmanship band 60 and
// warrior-fury band 60 (see this lane's report): +1 hit point reads
// bit-identical to +0, but +2 already shows a real, positive DPS
// gain.
//
// Armor, BonusArmor and Mana already get a x20 statMod for the
// analogous reason (their own per-point DPS effect is too small for
// the default mod to resolve); this test pins that Hit now gets the
// same treatment, scaled the same way, so the sweep's own perturbation
// clears the suppression floor with room to spare instead of reading
// back a false 0/0.
func TestBuildStatWeightRequestsScalesHitLikeArmor(t *testing.T) {
	scaledStats := []proto.Stat{proto.Stat_StatArmor, proto.Stat_StatBonusArmor, proto.Stat_StatMana, proto.Stat_StatHit}
	unscaledStats := []proto.Stat{proto.Stat_StatAgility, proto.Stat_StatCrit, proto.Stat_StatMeleeHaste}

	req := &proto.StatWeightsRequest{
		Player: &proto.Player{
			BonusStats: &proto.UnitStats{},
		},
		SimOptions:      &proto.SimOptions{Iterations: 2},
		EpReferenceStat: proto.Stat_StatAttackPower,
		StatsToWeigh:    append(append([]proto.Stat{}, scaledStats...), unscaledStats...),
	}

	data := buildStatWeightRequests(req)

	modFor := func(stat proto.Stat) (low, high float64, found bool) {
		unitStat := stats.UnitStatFromStat(stats.Stat(stat))
		for _, sr := range data.StatSimRequests {
			if stats.UnitStatFromIdx(int(sr.StatData.UnitStat)) == unitStat {
				return sr.StatData.ModLow, sr.StatData.ModHigh, true
			}
		}
		return 0, 0, false
	}

	for _, stat := range scaledStats {
		low, high, found := modFor(stat)
		if !found {
			t.Fatalf("%s: no stat sim request built", stat)
		}
		if high != 20 || low != -20 {
			t.Errorf("%s: mod = (%v, %v), want (-20, 20) - the same x20 scale Armor/BonusArmor/Mana already get", stat, low, high)
		}
	}

	for _, stat := range unscaledStats {
		low, high, found := modFor(stat)
		if !found {
			t.Fatalf("%s: no stat sim request built", stat)
		}
		if high != 1 || low != -1 {
			t.Errorf("%s: mod = (%v, %v), want (-1, 1) - unrelated stats must not be swept up by the Hit fix", stat, low, high)
		}
	}
}
