package hunter

import (
	"testing"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/common/clientdamage/clientdamagetest"
)

const clientHunterSpellconst = "../core/testdata/conformance/client/hunter.json"

// assertTable checks one rank table against the client's effect of the
// given kind, on the ids the table is read for.
func assertTable(t *testing.T, kind clientdamagetest.Kind, name string, ids []int32, effects []clientdamage.Effect) {
	t.Helper()
	clientdamagetest.AssertTable(t, clientdamagetest.Load(t, clientHunterSpellconst), kind, name, ids, effects, nil)
}

func TestArcaneShotDamageMatchesClient(t *testing.T) {
	assertTable(t, clientdamagetest.Direct, "Arcane Shot", ArcaneShotSpellId[:], ArcaneShotDamage)
}

func TestSerpentStingDamageMatchesClient(t *testing.T) {
	assertTable(t, clientdamagetest.Periodic, "Serpent Sting",
		[]int32{0, 1978, 13549, 13550, 13551, 13552, 13553, 13554, 13555, 25295}, SerpentStingDamage[:])
}

func TestAimedShotDamageMatchesClient(t *testing.T) {
	assertTable(t, clientdamagetest.NormalizedWeapon, "Aimed Shot",
		[]int32{0, 19434, 20900, 20901, 20902, 20903, 20904}, AimedShotDamage[:])
}

func TestRaptorStrikeDamageMatchesClient(t *testing.T) {
	assertTable(t, clientdamagetest.WeaponDamage, "Raptor Strike", RaptorStrikeSpellId[:], RaptorStrikeDamage[:])
}

func TestMongooseBiteDamageMatchesClient(t *testing.T) {
	assertTable(t, clientdamagetest.NormalizedWeapon, "Mongoose Bite", MongooseBiteSpellId[:], MongooseBiteDamage)
}

func TestCounterattackDamageMatchesClient(t *testing.T) {
	assertTable(t, clientdamagetest.NormalizedWeapon, "Counterattack", CounterattackSpellId[:], CounterattackDamage)
}

func TestWingClipDamageMatchesClient(t *testing.T) {
	assertTable(t, clientdamagetest.Direct, "Wing Clip", WingClipSpellId[:], WingClipDamage)
}

func TestSniperShotDamageMatchesClient(t *testing.T) {
	assertTable(t, clientdamagetest.NormalizedWeapon, "Sniper Shot", SniperShotSpellId[:], SniperShotDamage)
}

func TestVolleyDamageMatchesClient(t *testing.T) {
	assertTable(t, clientdamagetest.Direct, "Volley", VolleySpellId[:], VolleyDamage)
}

func TestImmolationTrapDamageMatchesClient(t *testing.T) {
	assertTable(t, clientdamagetest.Periodic, "Immolation Trap",
		[]int32{0, 13797, 14298, 14299, 14300, 14301}, ImmolationTrapTickDamage[:])
}

func TestExplosiveTrapDamageMatchesClient(t *testing.T) {
	ids := []int32{0, 13812, 14314, 14315}
	assertTable(t, clientdamagetest.Direct, "Explosive Trap", ids, ExplosiveTrapDamage[:])
	assertTable(t, clientdamagetest.AreaPeriodic, "Explosive Trap burn", ids, ExplosiveTrapTickDamage[:])
}

func TestSummonHawkDamageMatchesClient(t *testing.T) {
	assertTable(t, clientdamagetest.Direct, "Summon Hawk", SummonHawkSpellId[:], SummonHawkDamage)
}

func TestLacerateDamageMatchesClient(t *testing.T) {
	assertTable(t, clientdamagetest.Periodic, "Lacerate", LacerateSpellId[:], LacerateDamage)
}
