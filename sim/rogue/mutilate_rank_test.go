package rogue

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

// TestMutilateRankAtLevel pins mutilateLearnLevels/mutilateSpellID
// against spellranks.json's Mutilate chain (1310707@30, 399956@40,
// 1241582@50, 1241584@60): a rank the ladder's rewrite of a rotation's
// castSpell resolves to must be the same rank this file registers, at
// every level the ladder measures.
func TestMutilateRankAtLevel(t *testing.T) {
	cases := []struct {
		level     int32
		wantRank  int
		wantSpell int32
	}{
		{level: 10, wantRank: 0, wantSpell: 0}, // not learned yet
		{level: 29, wantRank: 0, wantSpell: 0}, // one below rank 1's floor
		{level: 30, wantRank: 1, wantSpell: 1310707},
		{level: 39, wantRank: 1, wantSpell: 1310707},
		{level: 40, wantRank: 2, wantSpell: 399956},
		{level: 50, wantRank: 3, wantSpell: 1241582},
		{level: 60, wantRank: 4, wantSpell: 1241584},
	}
	for _, c := range cases {
		rank := core.HighestRankAtLevel(mutilateLearnLevels, c.level)
		if rank != c.wantRank {
			t.Errorf("level %d: rank = %d, want %d", c.level, rank, c.wantRank)
			continue
		}
		if rank == 0 {
			continue
		}
		if got := mutilateSpellID[rank]; got != c.wantSpell {
			t.Errorf("level %d: mutilateSpellID[%d] = %d, want %d", c.level, rank, got, c.wantSpell)
		}
	}
}
