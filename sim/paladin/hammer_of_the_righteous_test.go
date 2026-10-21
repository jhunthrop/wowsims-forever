package paladin

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core/spellconst"
)

// The client's text for Hammer of the Righteous (spell 407632) is "causing
// Holy damage to each equal to $s3 times the damage per second of your main
// hand weapon", and its third effect is 3. A weapon of any speed is 3 times
// its damage per second, so a slow weapon's hit is the swing damage over its
// speed times 3.
func TestHammerOfTheRighteousIsThreeTimesTheWeaponsDamagePerSecond(t *testing.T) {
	cases := []struct {
		weaponSpeed float64
		want        float64 // share of one swing's damage
	}{{1.5, 2.0}, {2.0, 1.5}, {2.5, 1.2}, {3.0, 1.0}, {3.6, 3.0 / 3.6}}
	for _, c := range cases {
		if got := hammerOfTheRighteousSwingShare(c.weaponSpeed); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("weapon speed %v: %v of a swing, want %v", c.weaponSpeed, got, c.want)
		}
	}
}

func TestHammerOfTheRighteousMultipleMatchesTheClient(t *testing.T) {
	client, err := spellconst.Load(clientPaladinSpellconst)
	if err != nil {
		t.Fatal(err)
	}
	spell, ok := client.ByID(hammerOfTheRighteousActionID)
	if !ok {
		t.Fatalf("spell %d is not in the client's spellconst", hammerOfTheRighteousActionID)
	}
	if got, want := float64(hammerOfTheRighteousDPSMultiple), spell.Effects[2].Amount; got != want {
		t.Errorf("damage per second multiple %v, client %v", got, want)
	}
	if got, want := hammerOfTheRighteousFlat, spell.Effects[0].Amount; got != want {
		t.Errorf("flat damage %v, client %v", got, want)
	}
}
