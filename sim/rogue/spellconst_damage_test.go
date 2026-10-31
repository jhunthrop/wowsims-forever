package rogue

import (
	"testing"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/common/clientdamage/clientdamagetest"
	"github.com/wowsims/classic/sim/core"
)

const clientRogueSpellconst = "../core/testdata/conformance/client/rogue.json"

// assertTable checks one rank table against the client's effect of the
// given kind, on the ids the engine registers (or, for a spell whose
// damage the client keeps in a separate spell, that spell's ids).
func assertTable(t *testing.T, kind clientdamagetest.Kind, name string, ids []int32, effects []clientdamage.Effect) {
	t.Helper()
	clientdamagetest.AssertTable(t, clientdamagetest.Load(t, clientRogueSpellconst), kind, name, ids, effects, nil)
}

func TestSinisterStrikeDamageMatchesClient(t *testing.T) {
	assertTable(t, clientdamagetest.NormalizedWeapon, "Sinister Strike", sinisterStrikeSpellID[:], SinisterStrikeDamage)
}

func TestBackstabDamageMatchesClient(t *testing.T) {
	assertTable(t, clientdamagetest.NormalizedWeapon, "Backstab", backstabSpellID[:], BackstabDamage)
}

func TestAmbushDamageMatchesClient(t *testing.T) {
	assertTable(t, clientdamagetest.NormalizedWeapon, "Ambush", ambushSpellID[:], AmbushDamage)
}

func TestMutilateDamageMatchesClient(t *testing.T) {
	assertTable(t, clientdamagetest.NormalizedWeapon, "Mutilate", mutilateMHSpellID[:], MutilateDamage[:])
	assertTable(t, clientdamagetest.NormalizedWeapon, "Mutilate off hand", mutilateOHSpellID[:], MutilateDamage[:])
}

func TestEviscerateDamageMatchesClient(t *testing.T) {
	assertTable(t, clientdamagetest.Direct, "Eviscerate", eviscerateSpellID[:], EviscerateDamage)
}

func TestGarroteDamageMatchesClient(t *testing.T) {
	assertTable(t, clientdamagetest.Periodic, "Garrote", garroteSpellID[:], GarroteTickDamage)
}

func TestGougeDamageMatchesClient(t *testing.T) {
	assertTable(t, clientdamagetest.Direct, "Gouge", GougeSpellId[:], GougeDamage)
}

func TestKickDamageMatchesClient(t *testing.T) {
	assertTable(t, clientdamagetest.Direct, "Kick", KickSpellId[:], KickDamage)
}

func TestRuptureDamageMatchesClient(t *testing.T) {
	assertTable(t, clientdamagetest.Periodic, "Rupture", ruptureSpellID[:], RuptureTickDamage)
}

func TestInstantPoisonDamageMatchesClient(t *testing.T) {
	assertTable(t, clientdamagetest.Direct, "Instant Poison",
		[]int32{0, 8680, 8685, 8689, 11335, 11336, 11337}, InstantPoisonDamage[:])
}

// Forever's Deadly Poison enchants (SpellItemEnchantment 7, 8, 626, 627,
// 2630) trigger 2818, 2819, 11353, 11354 and 25349, which tick for 6, 9,
// 14, 18 and 23 every 3 s (SpellEffect aura 3, client 1.60.1.70658). The
// 434312-434316 family (9 to 34) is Season of Discovery's and no Forever
// enchant applies it; the conformance client file keeps only that family,
// so the Forever amounts are pinned here.
func TestDeadlyPoisonDamageMatchesClient(t *testing.T) {
	want := []float64{0, 6, 9, 14, 18, core.TernaryFloat64(core.IncludeAQ, 23, 18)}
	for rank := 1; rank < len(want); rank++ {
		if got := DeadlyPoisonTickDamage[rank].Amount; got != want[rank] {
			t.Errorf("Deadly Poison rank %d ticks for %v, want the enchant spell's %v", rank, got, want[rank])
		}
	}
}
