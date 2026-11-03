package warlock

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
)

// Lash of Pain is the client's Forever amount: rank 6 (11780) deals a flat 50
// (SpellEffect base 50, sp coefficient 0.429; Wowhead Forever "causing 50
// Shadow damage"), half of Era's 99.
func TestLashOfPainUsesForeversBaseDamage(t *testing.T) {
	_, warlock := newPetInheritanceTestWarlock(t, proto.WarlockOptions_Succubus)
	lash := warlock.Succubus.primaryAbility
	if got := lash.ClientBaseDamage; got != [2]float64{50, 50} {
		t.Errorf("Lash of Pain rank 6 base = %v, want the client's flat 50", got)
	}
}

// Firebolt rank 7 (11763) rolls the client's centre 44, 0.1136 wide, plus
// 0.6 a level above 58: a centre of 45.2, so 42.63 to 47.77 at 60.
func TestFireboltUsesTheClientRoll(t *testing.T) {
	_, warlock := newPetInheritanceTestWarlock(t, proto.WarlockOptions_Imp)
	got := warlock.Imp.primaryAbility.ClientBaseDamage
	if math.Abs(got[0]-42.63) > 0.01 || math.Abs(got[1]-47.77) > 0.01 {
		t.Errorf("Firebolt rank 7 base = %v, want 42.63-47.77", got)
	}
}

// Unholy Power raises "all damage done by your Imp, Voidwalker, Succubus,
// Incubus, and Felhunter pets" by 2% a rank, and the Firebolt tooltip applies
// it, so it is a pet-wide multiplier: the Imp's Firebolt gains it, and the
// Succubus' melee and Lash of Pain gain it once, not twice. The test warlock
// is an orc, so Command's 5% pet damage multiplies in as well, and Command's
// 70753 spell modifier on class mask 0x3000 gives Lash of Pain its own 5%.
func TestUnholyPowerRaisesAllDemonDamageOnce(t *testing.T) {
	_, warlock := newPetInheritanceTestWarlock(t, proto.WarlockOptions_Imp)
	// Both add points to the demon's Tamed Pet Passive: 1 + 0.10 + 0.05.
	const want = 1.15
	for _, pet := range []*WarlockPet{warlock.Imp, warlock.Succubus} {
		if got := pet.PseudoStats.DamageDealtMultiplier; math.Abs(got-want) > 1e-9 {
			t.Errorf("%s damage dealt multiplier = %.4f, want Unholy Power's 10%% plus Command's 5%%", pet.Name, got)
		}
	}
	if got := warlock.Succubus.AutoAttacks.MHConfig().DamageMultiplier; got != 1 {
		t.Errorf("Succubus melee multiplier = %.3f, want 1 (Unholy Power is pet-wide)", got)
	}
	if got := warlock.Succubus.primaryAbility.DamageMultiplier; math.Abs(got-1.05) > 1e-9 {
		t.Errorf("Lash of Pain multiplier = %.3f, want Command's 1.05 alone (Unholy Power is pet-wide)", got)
	}
}
