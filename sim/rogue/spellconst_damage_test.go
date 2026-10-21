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

func TestDeadlyPoisonDamageMatchesClient(t *testing.T) {
	ids := []int32{0, 434312, 434313, 434314, 434315, 434316}
	top := core.MaxTrainerRank(len(ids) - 1)
	assertTable(t, clientdamagetest.Periodic, "Deadly Poison", ids[:top+1], DeadlyPoisonTickDamage[:top+1])
}
