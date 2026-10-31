package druid

import (
	"math"
	"testing"
)

// Lacerate's hit since client 1.60.1.70707: "10% weapon damage per existing
// application of Lacerate on the target". The first application lands no
// weapon damage; the fifth lands 40%. Before the hotfix every hit was a flat
// 20% weapon damage.
func TestLacerateHitScalesWithExistingStacks(t *testing.T) {
	const weaponDamage = 200.0
	for stacks, want := range []float64{0, 20, 40, 60, 80} {
		if got := lacerateHitDamage(int32(stacks), weaponDamage); math.Abs(got-want) > 1e-9 {
			t.Errorf("hit with %d stacks up = %v, want %v", stacks, got, want)
		}
	}
}

// The live ladder's low ranks total 20 and 30 over the five ticks.
func TestLacerateLowRanksTotalOverFifteenSeconds(t *testing.T) {
	for _, c := range []struct {
		level int32
		total float64
	}{{25, 20}, {34, 30}, {42, 50}, {50, 60}, {58, 75}} {
		rank, number := lacerateRankAt(c.level)
		if number == 0 {
			t.Fatalf("level %d: no Lacerate rank", c.level)
		}
		if got := rank.tick.Center(int(c.level)) * lacerateTicks; got != c.total {
			t.Errorf("level %d: Lacerate totals %v over 15 s, want %v", c.level, got, c.total)
		}
	}
	if _, number := lacerateRankAt(24); number != 0 {
		t.Errorf("level 24 knows Lacerate rank %d, want none", number)
	}
}

// Primal Bite's flat amount per live rank: 42, 62, 77 (26, 38, 77 in the
// client tables at those levels).
func TestPrimalBiteLiveFlatDamage(t *testing.T) {
	for _, c := range []struct {
		level int32
		flat  float64
	}{{40, 42}, {49, 42}, {50, 62}, {60, 77}} {
		rank, ok := primalBiteRankAt(c.level)
		if !ok {
			t.Fatalf("level %d: no Primal Bite rank", c.level)
		}
		if rank.flat != c.flat {
			t.Errorf("level %d: Primal Bite flat = %v, want %v", c.level, rank.flat, c.flat)
		}
	}
}
