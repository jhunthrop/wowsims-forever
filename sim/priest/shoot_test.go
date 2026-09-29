package priest

import (
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
)

// wandSpecializationMultiplier is the class-specific half of core.RegisterShootSpell's wand
// damage math; core/wand_test.go covers the shared Shoot spell config directly (school, zero
// cost, cast time from a known wand's speed), and mage/shoot_test.go's
// TestShootDealsWandDamageAgainstARealTarget covers the same core.RegisterShootSpell path
// end-to-end against a live target, so this only needs the talent-rank table.
func TestWandSpecializationMultiplier(t *testing.T) {
	cases := []struct {
		rank int32
		want float64
	}{
		{rank: 0, want: 1},
		{rank: 1, want: 1.13},
		{rank: 2, want: 1.25},
	}

	for _, c := range cases {
		priest := &Priest{Talents: &proto.PriestTalents{WandSpecialization: c.rank}}
		if got := priest.wandSpecializationMultiplier(); got != c.want {
			t.Errorf("wandSpecializationMultiplier() at rank %d = %v, want %v", c.rank, got, c.want)
		}
	}
}
