package druid

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/common/clientdamage/clientdamagetest"
	"github.com/wowsims/classic/sim/core/spellconst"
)

// Client effect numbers of a weapon special: 58 is the flat weapon damage
// bonus, 31 the weapon damage percentage.
const (
	clientEffectWeaponFlat    = 58
	clientEffectWeaponPercent = 31
)

func clientWeaponEffect(t *testing.T, spell spellconst.Spell, effect int32) float64 {
	t.Helper()
	for _, e := range spell.Effects {
		if e.Effect == effect {
			return e.Amount
		}
	}
	t.Fatalf("%s (%d): no effect %d", spell.Name, spell.ID, effect)
	return 0
}

// Forever's cat specials keep the client's weapon percentage and flat bonus
// on every rank. Shred is 155% in Forever (Era's 225%; Wowhead Forever,
// spell 9830: "causing 155% damage plus 80"), and Claw is 110%.
func TestCatWeaponSpecialsMatchTheClient(t *testing.T) {
	class := clientdamagetest.Load(t, clientDruidSpellconst)
	cases := []struct {
		name    string
		ids     []int32
		flat    []float64
		percent float64
	}{
		{"Shred", shredSpellID[:], shredFlatDamageBonus[:], shredWeaponDamageMultiplier},
		{"Claw", clawSpellID[:], clawFlatDamageBonus[:], clawWeaponDamageMultiplier},
		{"Ravage", ravageSpellID[:], ravageFlatDamageBonus[:], ravageDamageMultiplier},
	}
	for _, c := range cases {
		for rank := 1; rank < len(c.ids); rank++ {
			spell, ok := class.ByID(c.ids[rank])
			if !ok {
				t.Fatalf("%s rank %d: %d is not in the client's spellconst", c.name, rank, c.ids[rank])
			}
			if got, want := c.flat[rank], clientWeaponEffect(t, spell, clientEffectWeaponFlat); got != want {
				t.Errorf("%s rank %d (%d): flat bonus %v, client %v", c.name, rank, c.ids[rank], got, want)
			}
			if got, want := c.percent, clientWeaponEffect(t, spell, clientEffectWeaponPercent)/100; math.Abs(got-want) > 1e-9 {
				t.Errorf("%s rank %d (%d): weapon damage x%v, client x%v", c.name, rank, c.ids[rank], got, want)
			}
		}
	}
}
