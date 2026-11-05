package druid

import (
	"math"
	"testing"
)

// Blizzard's Forever notes of 3 November 2026: "Rip: Base points and attack
// power scaling increased by about 25%" and the same for Ferocious Bite.
// The base half is in the client (Rip rank 6 15 to 19 a tick and 25.5 to
// 31.9 a point; Bite 145 to 181 a point, push 113403). The attack power
// half is server-side: Era's 6% of attack power a combo point over Rip's
// duration (four points at most) and 3% a point on Ferocious Bite become
// 7.5% and 3.75%.
func TestRipAndBiteAttackPowerSharesAreRaisedAQuarter(t *testing.T) {
	for _, c := range []struct {
		name string
		got  float64
		want float64
	}{
		{"Rip, 1 point, all six ticks", float64(RipNumberOfTicks) * ripTickAttackPowerShare(1), 0.075},
		{"Rip, 4 points, all six ticks", float64(RipNumberOfTicks) * ripTickAttackPowerShare(4), 0.30},
		{"Rip, 5 points, all six ticks", float64(RipNumberOfTicks) * ripTickAttackPowerShare(5), 0.30},
		{"Ferocious Bite, 5 points", ferociousBiteAttackPowerShare(5), 0.1875},
	} {
		if math.Abs(c.got-c.want) > 1e-9 {
			t.Errorf("%s takes %v of attack power, want %v", c.name, c.got, c.want)
		}
	}
}
