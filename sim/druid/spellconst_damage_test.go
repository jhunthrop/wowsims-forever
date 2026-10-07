package druid

import (
	"testing"

	"github.com/wowsims/classic/sim/common/clientdamage/clientdamagetest"
)

const clientDruidSpellconst = "../core/testdata/conformance/client/druid.json"

func TestWrathDamageMatchesClient(t *testing.T) {
	clientdamagetest.AssertTable(t, clientdamagetest.Load(t, clientDruidSpellconst), clientdamagetest.Direct, "Wrath",
		WrathSpellId[:], WrathDamage[:], WrathSpellCoeff[:])
}

func TestStarfireDamageMatchesClient(t *testing.T) {
	clientdamagetest.AssertTable(t, clientdamagetest.Load(t, clientDruidSpellconst), clientdamagetest.Direct, "Starfire",
		StarfireSpellId[:], StarfireDamage[:], StarfireSpellCoeff[:])
}

func TestMoonfireDamageMatchesClient(t *testing.T) {
	class := clientdamagetest.Load(t, clientDruidSpellconst)
	clientdamagetest.AssertTable(t, class, clientdamagetest.Direct, "Moonfire",
		MoonfireSpellId[:], MoonfireDamage[:], MoonfireSpellCoeff[:])
	clientdamagetest.AssertTable(t, class, clientdamagetest.Periodic, "Moonfire DoT",
		MoonfireSpellId[:], MoonfireTickDamage[:], MoonfireDotSpellCoeff[:])
}

func TestInsectSwarmDamageMatchesClient(t *testing.T) {
	clientdamagetest.AssertTable(t, clientdamagetest.Load(t, clientDruidSpellconst), clientdamagetest.Periodic, "Insect Swarm",
		InsectSwarmSpellId[:], InsectSwarmTickDamage[:], InsectSwarmTickSpellCoeff[:])
}

// The Rake, Rip and Ferocious Bite ladders read constants_auto_gen.go's
// own rows; these check that the ranks the engine registers (1 and up)
// still line up with the client's, rank 0 being a different spell.
func TestRakeDamageMatchesClient(t *testing.T) {
	class := clientdamagetest.Load(t, clientDruidSpellconst)
	clientdamagetest.AssertTable(t, class, clientdamagetest.Direct, "Rake",
		RakeSpellId[:], RakeInitialDamage[:], nil)
	clientdamagetest.AssertTable(t, class, clientdamagetest.Periodic, "Rake DoT",
		RakeSpellId[:], RakeTickDamage[:], nil)
}

func TestRipDamageMatchesClient(t *testing.T) {
	clientdamagetest.AssertTable(t, clientdamagetest.Load(t, clientDruidSpellconst), clientdamagetest.Periodic, "Rip",
		RipSpellId[:], RipTickDamage[:], nil)
}

func TestFerociousBiteDamageMatchesClient(t *testing.T) {
	clientdamagetest.AssertTable(t, clientdamagetest.Load(t, clientDruidSpellconst), clientdamagetest.Direct, "Ferocious Bite",
		FerociousBiteSpellId[:], FerociousBiteDamage[:], nil)
}

// The client states Ferocious Bite's damage per point of energy spent as the
// amount (in hundredths) of the spell's dummy effect 1: 100/150/200/250/270
// for ranks 1-5. Neither Rip nor Ferocious Bite carries a per-combo-point
// step anywhere in its effect rows (Rip's second effect is a zero-amount
// dummy), so ripTickPerComboPoint and ferociousBiteDamagePerComboPoint
// remain Era figures.
func TestFerociousBiteDamagePerEnergyMatchesClient(t *testing.T) {
	const dummyEffectIndex = 1
	const clientAmountPerEnergyPoint = 100.0
	class := clientdamagetest.Load(t, clientDruidSpellconst)
	for rank := 1; rank <= FerociousBiteRanks; rank++ {
		spell, ok := class.ByID(FerociousBiteSpellId[rank])
		if !ok {
			t.Fatalf("Ferocious Bite rank %d (%d) is not in the client table", rank, FerociousBiteSpellId[rank])
		}
		if got, want := ferociousBiteDamagePerEnergy[rank], spell.Effects[dummyEffectIndex].Amount/clientAmountPerEnergyPoint; got != want {
			t.Errorf("Ferocious Bite rank %d: %v damage per energy, client dummy effect states %v", rank, got, want)
		}
	}
}
