package warrior

import (
	"math"
	"testing"
)

// Thunder Clap scales with attack power in Forever (Blizzard's 30
// September 2026 deep dive); the share is the logs' 0.0625, read off
// eleven Protection warriors' Thunder Clap hits against their Revenge
// hits (see thunderClapAttackPowerCoefficient).
func TestThunderClapAddsASixteenthOfAttackPower(t *testing.T) {
	if got, want := thunderClapBaseDamage(103, 1600), 103+100.0; math.Abs(got-want) > 1e-9 {
		t.Errorf("Thunder Clap at 1,600 attack power = %v, want %v", got, want)
	}
	if got := thunderClapBaseDamage(37, 0); got != 37 {
		t.Errorf("Thunder Clap with no attack power = %v, want the rank's 37", got)
	}
}
