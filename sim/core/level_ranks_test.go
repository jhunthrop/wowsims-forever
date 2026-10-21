package core

import "testing"

func TestHighestRankAtLevel(t *testing.T) {
	// Mongoose Bite: ranks 1-4 at 16, 30, 44, 58.
	levels := []int{16, 30, 44, 58}
	cases := []struct {
		level int32
		want  int
	}{
		{1, 0}, {15, 0}, {16, 1}, {29, 1}, {30, 2}, {43, 2}, {44, 3}, {57, 3}, {58, 4}, {60, 4},
	}
	for _, c := range cases {
		if got := HighestRankAtLevel(levels, c.level); got != c.want {
			t.Errorf("level %d: got rank %d, want %d", c.level, got, c.want)
		}
	}
	if got := HighestRankAtLevel(nil, 60); got != 0 {
		t.Errorf("no ranks: got %d, want 0", got)
	}
}
