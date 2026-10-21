package core

import (
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// TestBuildStatWeightRequestsStepsHitByOnePoint pins the 2026-10-07
// ratings lane's replacement for the earlier x20 hit step.
//
// The x20 step existed because a +-1 step on a character with no other
// hit read bit-identical in both directions (the first point of hit is
// cancelled by HitSuppression) and tripped the hard-cap detector. But hit
// is not linear: PhysicalHitChance floors at max(hit - suppression, 0) and
// a special attack stops missing at the miss-table cap, so +-20 averaged a
// saturated response into a weight several times too small (0.46 per
// rating point for a Fury warrior). hitStepWindow now measures hit's
// marginal value at the character's actual hit; with no player to profile
// the step is the plain +-1, and Armor/BonusArmor/Mana keep their x20.
func TestBuildStatWeightRequestsStepsHitByOnePoint(t *testing.T) {
	scaledStats := []proto.Stat{proto.Stat_StatArmor, proto.Stat_StatBonusArmor, proto.Stat_StatMana}
	unscaledStats := []proto.Stat{proto.Stat_StatHit, proto.Stat_StatStamina, proto.Stat_StatCrit, proto.Stat_StatMeleeHaste}

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
			t.Errorf("%s: mod = (%v, %v), want (-20, 20)", stat, low, high)
		}
	}

	for _, stat := range unscaledStats {
		low, high, found := modFor(stat)
		if !found {
			t.Fatalf("%s: no stat sim request built", stat)
		}
		if high != 1 || low != -1 {
			t.Errorf("%s: mod = (%v, %v), want (-1, 1)", stat, low, high)
		}
	}
}

// TestBuildStatWeightRequestsScalesMeleePrimariesLikeArmor pins the
// 2026-10-07 fix for the warrior weights: with a ±1 step, a class whose
// resource is damage-driven (rage) breaks the sweep's paired random
// streams, and the level-60 Arms table published attack_power
// 1.00 ± 0.51 and strength 1.47 ± 0.34 against an engine conversion of
// exactly 2 AP per Strength, so the ranker preferred raw-AP items over
// Strength items. AttackPower, RangedAttackPower, FeralAttackPower,
// Strength and Agility now take the same x20 step Hit, Intellect,
// Armor and Mana already do; secondary ratings keep ±1.
func TestBuildStatWeightRequestsScalesMeleePrimariesLikeArmor(t *testing.T) {
	scaledStats := []proto.Stat{
		proto.Stat_StatAttackPower, proto.Stat_StatRangedAttackPower, proto.Stat_StatFeralAttackPower,
		proto.Stat_StatStrength, proto.Stat_StatAgility,
	}
	unscaledStats := []proto.Stat{proto.Stat_StatCrit, proto.Stat_StatMeleeHaste, proto.Stat_StatStamina, proto.Stat_StatSpirit}

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
			t.Errorf("%s: mod = (%v, %v), want (-20, 20) - a melee/ranged primary needs the wide step to survive broken pairing", stat, low, high)
		}
	}

	for _, stat := range unscaledStats {
		low, high, found := modFor(stat)
		if !found {
			t.Fatalf("%s: no stat sim request built", stat)
		}
		if high != 1 || low != -1 {
			t.Errorf("%s: mod = (%v, %v), want (-1, 1) - secondary ratings keep the default step", stat, low, high)
		}
	}
}

// TestBuildStatWeightRequestsScalesIntellectLikeArmor pins this lane's
// fix (bis-ranker-integrity-6, item 8): sim/cmd/leveling-bis's own
// nightly stat-weights sweep reported Intellect "insignificant or
// negative" in 32 of 70 caster band tables (mage-fire band 40's own
// measurement: -0.083 ± 0.167 DPS per point - a stdev roughly twice
// the mean, so the default ±1-point sweep cannot even recover the
// correct sign). Unlike Hit's hard floor (HitSuppression), Intellect
// is not capped at all here - its own true per-point DPS effect for a
// caster is simply too small for defaultStatMod=1 to resolve above
// this sweep's own sampling noise, the same shape Armor/BonusArmor/
// Mana's existing x20 scale already exists to fix. This test pins
// that Intellect now gets the identical treatment, and that unrelated
// stats are not swept up by it.
func TestBuildStatWeightRequestsScalesIntellectLikeArmor(t *testing.T) {
	scaledStats := []proto.Stat{proto.Stat_StatArmor, proto.Stat_StatBonusArmor, proto.Stat_StatMana, proto.Stat_StatIntellect}
	unscaledStats := []proto.Stat{proto.Stat_StatStamina, proto.Stat_StatCrit, proto.Stat_StatSpellPower, proto.Stat_StatSpirit}

	req := &proto.StatWeightsRequest{
		Player: &proto.Player{
			BonusStats: &proto.UnitStats{},
		},
		SimOptions:      &proto.SimOptions{Iterations: 2},
		EpReferenceStat: proto.Stat_StatSpellPower,
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
			t.Errorf("%s: mod = (%v, %v), want (-1, 1) - unrelated stats (including Spirit, not weighed by any DPS caster spec today - this lane's report) must not be swept up by the Intellect fix", stat, low, high)
		}
	}
}
