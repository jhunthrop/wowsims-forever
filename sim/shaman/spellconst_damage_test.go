package shaman

import (
	"testing"
	"time"

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
		FireNovaDamageSpellId[:], FireNovaDamage[:], FireNovaDamageCoeff[:])
}

func TestFireNovaCastMatchesClient(t *testing.T) {
	class := clientdamagetest.Load(t, clientShamanSpellconst)
	for rank := 1; rank <= FireNovaLearnRanks; rank++ {
		id := FireNovaLearnSpellId[rank]
		spell, ok := class.ByID(id)
		if !ok {
			t.Fatalf("Fire Nova rank %d (%d) is not in the client table", rank, id)
		}
		if spell.Cost != FireNovaLearnManaCost[rank] {
			t.Errorf("rank %d cost = %v, client %v", rank, FireNovaLearnManaCost[rank], spell.Cost)
		}
		if spell.SpellLevel != FireNovaLearnLevel[rank] {
			t.Errorf("rank %d level = %d, client %d", rank, FireNovaLearnLevel[rank], spell.SpellLevel)
		}
		if got := time.Duration(spell.EffectiveCooldownMS()) * time.Millisecond; got != FireNovaCooldown {
			t.Errorf("rank %d cooldown = %v, client %v", rank, FireNovaCooldown, got)
		}
	}
}
