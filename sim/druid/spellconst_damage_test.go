package druid

import (
	"math"
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

// The Bear Form ladder: Swipe's school damage, Lacerate's tick, and the
// weapon-damage flats of Maul and Primal Bite.
func TestSwipeDamageMatchesClient(t *testing.T) {
	clientdamagetest.AssertTable(t, clientdamagetest.Load(t, clientDruidSpellconst), clientdamagetest.Direct, "Swipe",
		SwipeSpellId[:], SwipeDamage[:], nil)
}

func TestLacerateTickMatchesClient(t *testing.T) {
	clientdamagetest.AssertTable(t, clientdamagetest.Load(t, clientDruidSpellconst), clientdamagetest.Periodic, "Lacerate",
		LacerateSpellId[:], LacerateTickDamage[:], nil)
}

func TestMaulFlatDamageMatchesClient(t *testing.T) {
	clientdamagetest.AssertTable(t, clientdamagetest.Load(t, clientDruidSpellconst), clientdamagetest.WeaponDamage, "Maul",
		MaulSpellId[:], MaulFlatDamage, nil)
}

func TestPrimalBiteFlatDamageMatchesClient(t *testing.T) {
	clientdamagetest.AssertTable(t, clientdamagetest.Load(t, clientDruidSpellconst), clientdamagetest.WeaponDamage, "Primal Bite",
		PrimalBiteSpellId[:], PrimalBiteFlatDamage, nil)
}

// Demoralizing Roar's attack power reduction is the client's aura 99, the
// amount negative and growing with level.
func TestDemoralizingRoarReductionMatchesClient(t *testing.T) {
	class := clientdamagetest.Load(t, clientDruidSpellconst)
	for rank := 1; rank <= DemoralizingRoarRanks; rank++ {
		spell, ok := class.ByID(DemoralizingRoarSpellId[rank])
		if !ok {
			t.Fatalf("rank %d (%d) is not in the client table", rank, DemoralizingRoarSpellId[rank])
		}
		for _, level := range []int{DemoralizingRoarLevel[rank], 50, 60} {
			if level < DemoralizingRoarLevel[rank] {
				continue
			}
			effect := spell.Effects[0]
			want := effect.Amount + effect.PointsPerLevel*float64(min(level, spell.MaxLevel)-spell.SpellLevel)
			if got := DemoralizingRoarReduction[rank].Center(level); math.Abs(got+want) > 0.01 {
				t.Errorf("rank %d at level %d: reduction %v, client amount %v", rank, level, got, want)
			}
		}
	}
}

// Each bear form tier's passive states its attack power (aura 99, effect 3),
// health (aura 230, effect 2) and armor multiplier (aura 142, effect 1).
func TestBearFormTiersMatchClient(t *testing.T) {
	class := clientdamagetest.Load(t, clientDruidSpellconst)
	const (
		healthEffect      = 2
		attackPowerEffect = 3
	)
	for _, tier := range bearFormTiers {
		for _, level := range []int{tier.LearnLevel, 50, 60} {
			if level < tier.LearnLevel {
				continue
			}
			clientdamagetest.AssertRoll(t, class, "attack power", tier.PassiveSpellID, attackPowerEffect, level,
				[2]float64{tier.AttackPower.Center(level), tier.AttackPower.Center(level)})
			clientdamagetest.AssertRoll(t, class, "health", tier.PassiveSpellID, healthEffect, level,
				[2]float64{tier.Health.Center(level), tier.Health.Center(level)})
		}
		spell, ok := class.ByID(tier.PassiveSpellID)
		if !ok {
			t.Fatalf("passive %d is not in the client table", tier.PassiveSpellID)
		}
		if got := spell.Effects[1].Amount; got != tier.ArmorFromItemsPercent {
			t.Errorf("passive %d armor percent: engine %v, client %v", tier.PassiveSpellID, tier.ArmorFromItemsPercent, got)
		}
	}
}
