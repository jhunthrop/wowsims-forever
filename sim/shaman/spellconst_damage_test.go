package shaman

import (
	"testing"

	"github.com/wowsims/classic/sim/common/clientdamage/clientdamagetest"
)

const clientShamanSpellconst = "../core/testdata/conformance/client/shaman.json"

func TestLightningBoltDamageMatchesClient(t *testing.T) {
	class := clientdamagetest.Load(t, clientShamanSpellconst)
	clientdamagetest.AssertTable(t, class, clientdamagetest.Direct, "Lightning Bolt",
		LightningBoltSpellId[:], LightningBoltDamage[:], LightningBoltSpellCoef[:])
}

func TestChainLightningDamageMatchesClient(t *testing.T) {
	class := clientdamagetest.Load(t, clientShamanSpellconst)
	clientdamagetest.AssertTable(t, class, clientdamagetest.Direct, "Chain Lightning",
		ChainLightningSpellId[:], ChainLightningDamage[:], ChainLightningSpellCoef[:])
}

func TestEarthShockDamageMatchesClient(t *testing.T) {
	class := clientdamagetest.Load(t, clientShamanSpellconst)
	clientdamagetest.AssertTable(t, class, clientdamagetest.Direct, "Earth Shock",
		EarthShockSpellId[:], EarthShockDamage[:], EarthShockSpellCoef[:])
}

func TestFrostShockDamageMatchesClient(t *testing.T) {
	class := clientdamagetest.Load(t, clientShamanSpellconst)
	clientdamagetest.AssertTable(t, class, clientdamagetest.Direct, "Frost Shock",
		FrostShockSpellId[:], FrostShockDamage[:], FrostShockSpellCoef[:])
}

func TestFlameShockDamageMatchesClient(t *testing.T) {
	class := clientdamagetest.Load(t, clientShamanSpellconst)
	clientdamagetest.AssertTable(t, class, clientdamagetest.Direct, "Flame Shock",
		FlameShockSpellId[:], FlameShockDamage[:], FlameShockBaseSpellCoef[:])
	clientdamagetest.AssertTable(t, class, clientdamagetest.Periodic, "Flame Shock tick",
		FlameShockSpellId[:], FlameShockTickDamage[:], FlameShockDotSpellCoef[:])
}

func TestLavaBurstDamageMatchesClient(t *testing.T) {
	class := clientdamagetest.Load(t, clientShamanSpellconst)
	coefficients := make([]float64, LavaBurstRanks+1)
	for rank := 1; rank <= LavaBurstRanks; rank++ {
		coefficients[rank] = LavaBurstSpellCoefficient
	}
	clientdamagetest.AssertTable(t, class, clientdamagetest.Direct, "Lava Burst",
		LavaBurstSpellId[:], LavaBurstDamage[:], coefficients)
}

func TestFireTotemDamageMatchesClient(t *testing.T) {
	class := clientdamagetest.Load(t, clientShamanSpellconst)
	clientdamagetest.AssertTable(t, class, clientdamagetest.Direct, "Searing Totem bolt",
		SearingTotemAttackSpellId[:], SearingTotemDamage[:], SearingTotemSpellCoef[:])
	clientdamagetest.AssertTable(t, class, clientdamagetest.Direct, "Magma Totem",
		MagmaTotemAoeSpellId[:], MagmaTotemDamage[:], MagmaTotemSpellCoeff[:])
	clientdamagetest.AssertTable(t, class, clientdamagetest.Direct, "Fire Nova",
		FireNovaTotemAoeSpellId[:], FireNovaTotemDamage[:], FireNovaTotemSpellCoeff[:])
}
