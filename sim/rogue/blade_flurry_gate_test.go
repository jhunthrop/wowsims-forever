package rogue

import (
	"testing"
	"time"
)

// The last Blade Flurry of a fight waits for Slice and Dice, but never
// past the point where its own duration would run beyond the fight: a
// rotation that lets Slice and Dice drop must still get the cast.
func TestLastBladeFlurryFiresBeforeTheFightEnds(t *testing.T) {
	const cooldown, duration = 2 * time.Minute, 15 * time.Second
	cases := []struct {
		name      string
		remaining time.Duration
		snd       time.Duration
		want      bool
	}{
		{"another one fits", 3 * time.Minute, 0, true},
		{"last one, Slice and Dice up", 60 * time.Second, 10 * time.Second, true},
		{"last one, Slice and Dice down, time left", 60 * time.Second, 0, false},
		{"last one, Slice and Dice down, its duration left", duration, 0, true},
	}
	for _, c := range cases {
		if got := bladeFlurryShouldActivate(c.remaining, cooldown, duration, c.snd); got != c.want {
			t.Errorf("%s: activate = %v, want %v", c.name, got, c.want)
		}
	}
}
