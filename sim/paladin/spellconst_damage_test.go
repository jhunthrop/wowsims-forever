package paladin

import (
	"testing"

	"github.com/wowsims/classic/sim/common/clientdamage/clientdamagetest"
)

const clientPaladinSpellconst = "../core/testdata/conformance/client/paladin.json"

// The tables are checked against the client rows of the ids the engine
// registers each rank under.

func TestExorcismDamageMatchesClient(t *testing.T) {
	clientdamagetest.AssertTable(t, clientdamagetest.Load(t, clientPaladinSpellconst), clientdamagetest.Direct, "Exorcism",
		[]int32{0, 879, 5614, 5615, 10312, 10313, 10314}, ExorcismDamage[:], nil)
}

func TestHammerOfWrathDamageMatchesClient(t *testing.T) {
	clientdamagetest.AssertTable(t, clientdamagetest.Load(t, clientPaladinSpellconst), clientdamagetest.Direct, "Hammer of Wrath",
		[]int32{0, 24275, 24274, 24239}, HammerOfWrathDamage[:], nil)
}

func TestHolyWrathDamageMatchesClient(t *testing.T) {
	clientdamagetest.AssertTable(t, clientdamagetest.Load(t, clientPaladinSpellconst), clientdamagetest.Direct, "Holy Wrath",
		[]int32{0, 2812, 10318}, HolyWrathDamage[:], nil)
}

func TestJudgementOfRighteousnessDamageMatchesClient(t *testing.T) {
	coefficients := make([]float64, judgementOfRighteousnessRanks+1)
	for rank := 1; rank <= judgementOfRighteousnessRanks; rank++ {
		coefficients[rank] = judgementOfRighteousnessCoefficient
	}
	clientdamagetest.AssertTable(t, clientdamagetest.Load(t, clientPaladinSpellconst), clientdamagetest.Direct, "Judgement of Righteousness",
		[]int32{0, 20187, 20280, 20281, 20282, 20283, 20284, 20285, 20286}, JudgementOfRighteousnessDamage[:], coefficients)
}

func TestJudgementOfCommandDamageMatchesClient(t *testing.T) {
	clientdamagetest.AssertTable(t, clientdamagetest.Load(t, clientPaladinSpellconst), clientdamagetest.Direct, "Judgement of Command",
		[]int32{0, 20467, 20963, 20964, 20965, 20966}, JudgementOfCommandDamage[:], nil)
}

func TestHolyShockDamageMatchesClient(t *testing.T) {
	clientdamagetest.AssertTable(t, clientdamagetest.Load(t, clientPaladinSpellconst), clientdamagetest.Direct, "Holy Shock",
		[]int32{0, 25912, 25911, 25902}, HolyShockDamage[:], nil)
}

func TestHolyStrikeFlatDamageMatchesClient(t *testing.T) {
	clientdamagetest.AssertTable(t, clientdamagetest.Load(t, clientPaladinSpellconst), clientdamagetest.NormalizedWeapon, "Holy Strike",
		[]int32{0, 679, 678, 1866, 680, 2495, 5569, 10332, 10333}, HolyStrikeFlatDamage[:], nil)
}
