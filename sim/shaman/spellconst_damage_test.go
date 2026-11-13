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

// The overload rows are client 1.60.1.70888's, native in the vendored 70938
// file but with half points rounded away (36.5 reads 36), so they are pinned
// exactly by TestLightningOverloadRowsAreThe70888Client (hotfix_1110_test.go).

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

// TestNukeCastsMatchClient pins Lightning Bolt's and Chain Lightning's
// mana cost, cast time and level against the vendored client table, so a
// push that reprices or re-times a rank (1.60.1.70755 moved every Chain
// Lightning rank to CastingTimeIndex 19, 2500 ms, and raised its cost)
// fails here instead of drifting.
func TestNukeCastsMatchClient(t *testing.T) {
	class := clientdamagetest.Load(t, clientShamanSpellconst)
	check := func(name string, rank int, id int32, cost float64, castTime time.Duration, level int) {
		t.Helper()
		spell, ok := class.ByID(id)
		if !ok {
			t.Fatalf("%s rank %d (%d) is not in the client table", name, rank, id)
		}
		if spell.Cost != cost {
			t.Errorf("%s rank %d cost = %v, client %v", name, rank, cost, spell.Cost)
		}
		if got := time.Duration(spell.CastTimeMS) * time.Millisecond; got != castTime {
			t.Errorf("%s rank %d cast time = %v, client %v", name, rank, castTime, got)
		}
		if spell.SpellLevel != level {
			t.Errorf("%s rank %d level = %d, client %d", name, rank, level, spell.SpellLevel)
		}
	}
	for rank := 1; rank <= LightningBoltRanks; rank++ {
		check("Lightning Bolt", rank, LightningBoltSpellId[rank], LightningBoltManaCost[rank],
			time.Duration(LightningBoltCastTime[rank])*time.Millisecond, LightningBoltLevel[rank])
	}
	for rank := 1; rank <= ChainLightningRanks; rank++ {
		check("Chain Lightning", rank, ChainLightningSpellId[rank], ChainLightningManaCost[rank],
			ChainLightningCastTime, ChainLightningLevel[rank])
		spell, _ := class.ByID(ChainLightningSpellId[rank])
		if got := time.Duration(spell.EffectiveCooldownMS()) * time.Millisecond; got != ChainLightningCooldown {
			t.Errorf("Chain Lightning rank %d cooldown = %v, client %v", rank, ChainLightningCooldown, got)
		}
	}
}
