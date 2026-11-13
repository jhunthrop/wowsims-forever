package paladin

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
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

// The client's SpellEquippedItems row for 407632 (1.60.1.70888, hotfixed)
// is item class 2 with subclass mask 145: one-handed axes (0), one-handed
// maces (4) and one-handed swords (7). A two-hander, a dagger, a fist
// weapon or no weapon cannot cast it.
func TestHammerOfTheRighteousNeedsAOneHandedAxeMaceOrSword(t *testing.T) {
	cases := []struct {
		name string
		item *core.Item
		want bool
	}{
		{"one-handed mace", &core.Item{HandType: proto.HandType_HandTypeOneHand, WeaponType: proto.WeaponType_WeaponTypeMace}, true},
		{"main-hand sword", &core.Item{HandType: proto.HandType_HandTypeMainHand, WeaponType: proto.WeaponType_WeaponTypeSword}, true},
		{"one-handed axe", &core.Item{HandType: proto.HandType_HandTypeOneHand, WeaponType: proto.WeaponType_WeaponTypeAxe}, true},
		{"two-handed sword", &core.Item{HandType: proto.HandType_HandTypeTwoHand, WeaponType: proto.WeaponType_WeaponTypeSword}, false},
		{"two-handed mace", &core.Item{HandType: proto.HandType_HandTypeTwoHand, WeaponType: proto.WeaponType_WeaponTypeMace}, false},
		{"polearm", &core.Item{HandType: proto.HandType_HandTypeTwoHand, WeaponType: proto.WeaponType_WeaponTypePolearm}, false},
		{"dagger", &core.Item{HandType: proto.HandType_HandTypeOneHand, WeaponType: proto.WeaponType_WeaponTypeDagger}, false},
		{"fist weapon", &core.Item{HandType: proto.HandType_HandTypeOneHand, WeaponType: proto.WeaponType_WeaponTypeFist}, false},
		{"no weapon", nil, false},
	}
	for _, c := range cases {
		if got := hammerOfTheRighteousWeaponAllowed(c.item); got != c.want {
			t.Errorf("%s: allowed %v, want %v", c.name, got, c.want)
		}
	}
}
