package druid

import (
	"math"
	"testing"
)

// Demoralizing Roar's threat is the client's effect 63 on every rank
// (spellconst/druid.json 99, 1735, 9490, 9747, 9898): 8, 16, 26, 34 and 42
// at the rank's level, plus 0.8 a level to its max level. Rank 5 at 60 is
// 42 + 8 x 0.8 = 48.4 a target. The engine had twice the rank's level
// (104 at rank 5), the convention it used where it read no threat column.
func TestDemoralizingRoarThreatMatchesClient(t *testing.T) {
	for _, c := range []struct {
		rank  int
		level int32
		want  float64
	}{{1, 10, 8}, {1, 20, 16}, {2, 25, 20}, {5, 52, 42}, {5, 60, 48.4}} {
		if got := demoralizingRoarThreat(c.rank, c.level); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("rank %d at %d: threat %v, want %v", c.rank, c.level, got, c.want)
		}
	}
}
