package druid

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core/spellconst"
)

// clientDruidSpells loads the vendored copy of the site's
// spellconst/druid.json (the file the conformance report reads), the
// single source these damage tables must agree with.
func clientDruidSpells(t *testing.T) spellconst.Class {
	t.Helper()
	class, err := spellconst.Load("../core/testdata/conformance/client/druid.json")
	if err != nil {
		t.Fatalf("loading the client's druid spellconst: %v", err)
	}
	return class
}

func effectOf(t *testing.T, class spellconst.Class, spellID int32, match func(spellconst.Effect) bool) spellconst.Effect {
	t.Helper()
	spell, ok := class.ByID(spellID)
	if !ok {
		t.Fatalf("spell %d is not in the client's druid spellconst", spellID)
	}
	for _, effect := range spell.Effects {
		if match(effect) {
			return effect
		}
	}
	t.Fatalf("spell %d (%s) has no matching effect", spellID, spell.Name)
	return spellconst.Effect{}
}

func isDirectDamage(effect spellconst.Effect) bool { return effect.Effect == 2 }

func isPeriodicDamage(effect spellconst.Effect) bool { return effect.Effect == 6 && effect.Aura == 3 }

// assertRankTable checks one effect of every rank of one spell: the
// registered flat amount and spell-power coefficient against the client's.
func assertRankTable(t *testing.T, class spellconst.Class, name string, ranks int, spellID func(rank int) int32, match func(spellconst.Effect) bool, amounts, coeffs []float64) {
	t.Helper()
	for rank := 1; rank <= ranks; rank++ {
		client := effectOf(t, class, spellID(rank), match)
		if got := amounts[rank]; math.Abs(got-client.Amount) > 0.5 {
			t.Errorf("%s rank %d (%d): engine damage %v, client amount %v", name, rank, spellID(rank), got, client.Amount)
		}
		if got := coeffs[rank]; math.Abs(got-client.SPCoefficient) > 0.002 {
			t.Errorf("%s rank %d (%d): engine coefficient %v, client %v", name, rank, spellID(rank), got, client.SPCoefficient)
		}
	}
}

func TestWrathDamageMatchesClient(t *testing.T) {
	assertRankTable(t, clientDruidSpells(t), "Wrath", WrathRanks,
		func(rank int) int32 { return WrathSpellId[rank] }, isDirectDamage,
		WrathBaseDamage[:], WrathSpellCoeff[:])
}

func TestStarfireDamageMatchesClient(t *testing.T) {
	assertRankTable(t, clientDruidSpells(t), "Starfire", StarfireRanks,
		func(rank int) int32 { return StarfireSpellId[rank] }, isDirectDamage,
		StarfireBaseDamage[:], StarfireSpellCoeff[:])
}

func TestMoonfireDamageMatchesClient(t *testing.T) {
	class := clientDruidSpells(t)
	assertRankTable(t, class, "Moonfire", MoonfireRanks,
		func(rank int) int32 { return MoonfireSpellId[rank] }, isDirectDamage,
		MoonfireBaseDamage[:], MoonfireSpellCoeff[:])
	assertRankTable(t, class, "Moonfire DoT", MoonfireRanks,
		func(rank int) int32 { return MoonfireSpellId[rank] }, isPeriodicDamage,
		MoonfireTickDamage[:], MoonfireDotSpellCoeff[:])
}

func TestInsectSwarmDamageMatchesClient(t *testing.T) {
	assertRankTable(t, clientDruidSpells(t), "Insect Swarm", InsectSwarmRanks,
		func(rank int) int32 { return InsectSwarmSpellId[rank] }, isPeriodicDamage,
		InsectSwarmTickDamage[:], InsectSwarmTickSpellCoeff[:])
}
