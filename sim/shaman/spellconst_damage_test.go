package shaman

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core/spellconst"
)

// clientShamanSpells loads the vendored copy of the site's
// spellconst/shaman.json (the same file the conformance report reads),
// the single source these damage tables must agree with.
func clientShamanSpells(t *testing.T) spellconst.Class {
	t.Helper()
	class, err := spellconst.Load("../core/testdata/conformance/client/shaman.json")
	if err != nil {
		t.Fatalf("loading the client's shaman spellconst: %v", err)
	}
	return class
}

// directEffect is the spell's school-damage effect (effect 2): its flat
// per-rank "amount" and "sp_coefficient" are what the engine registers.
func directEffect(t *testing.T, class spellconst.Class, spellID int32) spellconst.Effect {
	t.Helper()
	spell, ok := class.ByID(spellID)
	if !ok {
		t.Fatalf("spell %d is not in the client's shaman spellconst", spellID)
	}
	for _, effect := range spell.Effects {
		if effect.Effect == 2 {
			return effect
		}
	}
	t.Fatalf("spell %d (%s) has no school-damage effect", spellID, spell.Name)
	return spellconst.Effect{}
}

// periodicEffect is the spell's periodic-damage effect (apply aura 3).
func periodicEffect(t *testing.T, class spellconst.Class, spellID int32) spellconst.Effect {
	t.Helper()
	spell, ok := class.ByID(spellID)
	if !ok {
		t.Fatalf("spell %d is not in the client's shaman spellconst", spellID)
	}
	for _, effect := range spell.Effects {
		if effect.Effect == 6 && effect.Aura == 3 {
			return effect
		}
	}
	t.Fatalf("spell %d (%s) has no periodic-damage effect", spellID, spell.Name)
	return spellconst.Effect{}
}

type directDamageTable struct {
	name    string
	ids     []int32
	damage  []float64
	coeffs  []float64
	ranks   int
	idAtRnk func(rank int) int32
}

// assertDirectTable checks every rank of one spell's registered base
// damage and spell-power coefficient against the client.
func assertDirectTable(t *testing.T, class spellconst.Class, table directDamageTable) {
	t.Helper()
	for rank := 1; rank <= table.ranks; rank++ {
		id := table.idAtRnk(rank)
		client := directEffect(t, class, id)
		if got := table.damage[rank]; math.Abs(got-client.Amount) > 0.5 {
			t.Errorf("%s rank %d (%d): engine base damage %v, client amount %v", table.name, rank, id, got, client.Amount)
		}
		if got := table.coeffs[rank]; math.Abs(got-client.SPCoefficient) > 0.002 {
			t.Errorf("%s rank %d (%d): engine spell-power coefficient %v, client %v", table.name, rank, id, got, client.SPCoefficient)
		}
	}
}

func TestLightningBoltDamageMatchesClient(t *testing.T) {
	class := clientShamanSpells(t)
	assertDirectTable(t, class, directDamageTable{
		name: "Lightning Bolt", ranks: LightningBoltRanks,
		damage: LightningBoltBaseDamage[:], coeffs: LightningBoltSpellCoef[:],
		idAtRnk: func(rank int) int32 { return LightningBoltSpellId[rank] },
	})
}

func TestChainLightningDamageMatchesClient(t *testing.T) {
	class := clientShamanSpells(t)
	assertDirectTable(t, class, directDamageTable{
		name: "Chain Lightning", ranks: ChainLightningRanks,
		damage: ChainLightningBaseDamage[:], coeffs: ChainLightningSpellCoef[:],
		idAtRnk: func(rank int) int32 { return ChainLightningSpellId[rank] },
	})
}

func TestEarthShockDamageMatchesClient(t *testing.T) {
	class := clientShamanSpells(t)
	assertDirectTable(t, class, directDamageTable{
		name: "Earth Shock", ranks: EarthShockRanks,
		damage: EarthShockBaseDamage[:], coeffs: EarthShockSpellCoef[:],
		idAtRnk: func(rank int) int32 { return EarthShockSpellId[rank] },
	})
}

func TestFrostShockDamageMatchesClient(t *testing.T) {
	class := clientShamanSpells(t)
	assertDirectTable(t, class, directDamageTable{
		name: "Frost Shock", ranks: FrostShockRanks,
		damage: FrostShockBaseDamage[:], coeffs: FrostShockSpellCoef[:],
		idAtRnk: func(rank int) int32 { return FrostShockSpellId[rank] },
	})
}

func TestFlameShockDamageMatchesClient(t *testing.T) {
	class := clientShamanSpells(t)
	assertDirectTable(t, class, directDamageTable{
		name: "Flame Shock", ranks: FlameShockRanks,
		damage: FlameShockBaseDamage[:], coeffs: FlameShockBaseSpellCoef[:],
		idAtRnk: func(rank int) int32 { return FlameShockSpellId[rank] },
	})

	for rank := 1; rank <= FlameShockRanks; rank++ {
		client := periodicEffect(t, class, FlameShockSpellId[rank])
		if got := FlameShockTickDamage[rank]; math.Abs(got-client.Amount) > 0.5 {
			t.Errorf("Flame Shock rank %d: engine damage per tick %v, client %v", rank, got, client.Amount)
		}
		if got := FlameShockDotSpellCoef[rank]; math.Abs(got-client.SPCoefficient) > 0.002 {
			t.Errorf("Flame Shock rank %d: engine per-tick coefficient %v, client %v", rank, got, client.SPCoefficient)
		}
	}
}

func TestFireTotemDamageMatchesClient(t *testing.T) {
	class := clientShamanSpells(t)
	assertDirectTable(t, class, directDamageTable{
		name: "Searing Totem bolt", ranks: SearingTotemRanks,
		damage: SearingTotemBaseDamage[:], coeffs: SearingTotemSpellCoef[:],
		idAtRnk: func(rank int) int32 { return SearingTotemAttackSpellId[rank] },
	})
	assertDirectTable(t, class, directDamageTable{
		name: "Magma Totem", ranks: MagmaTotemRanks,
		damage: MagmaTotemBaseDamage[:], coeffs: MagmaTotemSpellCoeff[:],
		idAtRnk: func(rank int) int32 { return MagmaTotemAoeSpellId[rank] },
	})
	assertDirectTable(t, class, directDamageTable{
		name: "Fire Nova", ranks: FireNovaTotemRanks,
		damage: FireNovaTotemBaseDamage[:], coeffs: FireNovaTotemSpellCoeff[:],
		idAtRnk: func(rank int) int32 { return FireNovaTotemAoeSpellId[rank] },
	})
}
