package warlock

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
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
// is an orc, so Command's 5% pet damage adds in as well. Command's 70753
// spell modifier on Firebolt and Lash of Pain is 0 since the 6 Nov hotfix,
// so neither bolt takes a second 5%.
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
	for _, spell := range []*core.Spell{warlock.Succubus.primaryAbility, warlock.Imp.primaryAbility} {
		if got := spell.DamageMultiplier; got != 1 {
			t.Errorf("%v multiplier = %.3f, want 1 (Unholy Power and Command are pet-wide)", spell.ActionID, got)
		}
	}
}

// TestFireboltLowRanksReadThe70888Push pins the 10 Nov hotfix ("Imp base
// damage of the first 4 Ranks of Imp's Firebolt increased"), push 113766 on
// 1.60.1.70888 and native in the 1.60.1.70938 tables, on SpellEffect 681913 and 685570 to 685572: ranks 1 to 4 are centres of 6, 11,
// 15 and 18 (were 4, 7, 12 and 17), 0.3333, 0.1818, 0.2353 and 0.2222 wide,
// growing 0.3, 0.4, 0.4 and 0.4 a level (were 0.1, 0.2, 0.3 and 0.3).
// Ranks 5 to 7 are unchanged.
func TestFireboltLowRanksReadThe70888Push(t *testing.T) {
	for _, c := range []struct {
		rank, level int
		lo, hi      float64
	}{
		{1, 5, 6.0, 8.4},      // centre 6 + 0.3 x 4 = 7.2, 0.3333 wide (cap 5)
		{2, 13, 11.82, 14.18}, // 11 + 0.4 x 5 = 13, 0.1818 wide (cap 13)
		{3, 23, 15.0, 19.0},   // 15 + 0.4 x 5 = 17, 0.2353 wide (cap 23)
		{4, 33, 17.78, 22.22}, // 18 + 0.4 x 5 = 20, 0.2222 wide (cap 33)
		{7, 60, 42.63, 47.77}, // unchanged
	} {
		got := FireboltDamage[c.rank].Range(c.level)
		if math.Abs(got[0]-c.lo) > 0.01 || math.Abs(got[1]-c.hi) > 0.01 {
			t.Errorf("Firebolt rank %d at %d = %.2f-%.2f, want %.2f-%.2f", c.rank, c.level, got[0], got[1], c.lo, c.hi)
		}
	}
}
