package warrior

import (
	"testing"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/common/clientdamage/clientdamagetest"
)

const clientWarriorSpellconst = "../core/testdata/conformance/client/warrior.json"

// assertGenerated checks one generated rank table against the client,
// reading it the way client_damage.go does.
func assertGenerated(t *testing.T, kind clientdamagetest.Kind, name string, ids []int32, effects []clientdamage.Effect) {
	t.Helper()
	clientdamagetest.AssertTable(t, clientdamagetest.Load(t, clientWarriorSpellconst), kind, name, ids, effects, nil)
}

func TestHamstringDamageMatchesClient(t *testing.T) {
	assertGenerated(t, clientdamagetest.Direct, "Hamstring", HamstringSpellId[:], HamstringDamage[:])
}

func TestPummelDamageMatchesClient(t *testing.T) {
	assertGenerated(t, clientdamagetest.Direct, "Pummel", PummelSpellId[:], PummelDamage[:])
}

func TestThunderClapDamageMatchesClient(t *testing.T) {
	assertGenerated(t, clientdamagetest.Direct, "Thunder Clap", ThunderClapSpellId[:], ThunderClapDamage[:])
}

func TestRendDamageMatchesClient(t *testing.T) {
	assertGenerated(t, clientdamagetest.Periodic, "Rend", RendSpellId[:], RendDamage[:])
}

func TestHeroicStrikeDamageMatchesClient(t *testing.T) {
	assertGenerated(t, clientdamagetest.WeaponDamageNoSchool, "Heroic Strike", HeroicStrikeSpellId[:], HeroicStrikeDamage[:])
}

func TestCleaveDamageMatchesClient(t *testing.T) {
	assertGenerated(t, clientdamagetest.WeaponDamageNoSchool, "Cleave", CleaveSpellId[:], CleaveDamage[:])
}

func TestSlamDamageMatchesClient(t *testing.T) {
	assertGenerated(t, clientdamagetest.WeaponDamageNoSchool, "Slam", slamRankSpellID[:], SlamDamage[:])
}

func TestExecuteDamageMatchesClient(t *testing.T) {
	assertGenerated(t, clientdamagetest.Dummy, "Execute", ExecuteSpellId[:], ExecuteDamage[:])
}

func TestOverpowerDamageMatchesClient(t *testing.T) {
	assertGenerated(t, clientdamagetest.NormalizedWeapon, "Overpower", OverpowerSpellId[:], OverpowerDamage[:])
}

func TestMortalStrikeDamageMatchesClient(t *testing.T) {
	assertGenerated(t, clientdamagetest.NormalizedWeapon, "Mortal Strike", MortalStrikeSpellId[:], MortalStrikeDamage[:])
}

func TestBloodthirstDamageMatchesClient(t *testing.T) {
	assertGenerated(t, clientdamagetest.Direct, "Bloodthirst", BloodthirstSpellId[:], BloodthirstDamage[:])
}

func TestShieldSlamDamageMatchesClient(t *testing.T) {
	assertGenerated(t, clientdamagetest.Direct, "Shield Slam", ShieldSlamSpellId[:], ShieldSlamDamage[:])
}

func TestRevengeDamageMatchesClient(t *testing.T) {
	assertGenerated(t, clientdamagetest.Direct, "Revenge", RevengeSpellId[:], RevengeDamage[:])
}
