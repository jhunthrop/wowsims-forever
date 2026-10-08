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
		[]int32{0, 1311604, 25912, 25911, 25902}, HolyShockDamage[:], nil)
}

func TestHolyStrikeFlatDamageMatchesClient(t *testing.T) {
	clientdamagetest.AssertTable(t, clientdamagetest.Load(t, clientPaladinSpellconst), clientdamagetest.NormalizedWeapon, "Holy Strike",
		[]int32{0, 679, 678, 1866, 680, 2495, 5569, 10332, 10333}, HolyStrikeFlatDamage[:], nil)
}

func TestSealOfFuryProcDamageMatchesClient(t *testing.T) {
	coefficients := make([]float64, sealOfFuryRankCount+1)
	for rank := 1; rank <= sealOfFuryRankCount; rank++ {
		coefficients[rank] = sealOfFuryProcCoefficient
	}
	clientdamagetest.AssertTable(t, clientdamagetest.Load(t, clientPaladinSpellconst), clientdamagetest.Direct, "Seal of Fury proc",
		sealOfFuryIDs(func(rank sealOfFuryRank) int32 { return rank.procID }), sealOfFuryProcDamage[:], coefficients)
}

func TestJudgementOfFuryDamageMatchesClient(t *testing.T) {
	coefficients := make([]float64, sealOfFuryRankCount+1)
	for rank := 1; rank <= sealOfFuryRankCount; rank++ {
		coefficients[rank] = judgementOfFuryCoefficient
	}
	clientdamagetest.AssertTable(t, clientdamagetest.Load(t, clientPaladinSpellconst), clientdamagetest.Direct, "Judgement of Fury",
		sealOfFuryIDs(func(rank sealOfFuryRank) int32 { return rank.judgeSpell }), JudgementOfFuryDamage[:], coefficients)
}

// sealOfFuryIDs is one id per rank, index 0 unused, as AssertTable reads.
func sealOfFuryIDs(id func(sealOfFuryRank) int32) []int32 {
	ids := make([]int32, sealOfFuryRankCount+1)
	for i, rank := range sealOfFuryRanks {
		ids[i+1] = id(rank)
	}
	return ids
}

// The seal's own row carries the cost and the level the ranks are learned
// at; a mistyped rank would silently shift the whole progression.
func TestSealOfFuryRanksMatchTheClientSeals(t *testing.T) {
	client := clientdamagetest.Load(t, clientPaladinSpellconst)
	for i, rank := range sealOfFuryRanks {
		spell, ok := client.ByID(rank.spellID)
		if !ok {
			t.Fatalf("rank %d: seal %d is not in the client table", i+1, rank.spellID)
		}
		if spell.SpellLevel != int(rank.level) || spell.Cost != rank.manaCost {
			t.Errorf("rank %d: engine level %d cost %v, client level %d cost %v", i+1, rank.level, rank.manaCost, spell.SpellLevel, spell.Cost)
		}
	}
}

// Hammer of the Righteous states a flat 1 Holy damage at every level;
// the rest of its damage is the script's (hammer_of_the_righteous.go).
func TestHammerOfTheRighteousFlatDamageMatchesClient(t *testing.T) {
	client := clientdamagetest.Load(t, clientPaladinSpellconst)
	for _, level := range []int{40, 50, 60} {
		clientdamagetest.AssertRoll(t, client, "Hammer of the Righteous", hammerOfTheRighteousActionID, 0, level,
			[2]float64{hammerOfTheRighteousFlat, hammerOfTheRighteousFlat})
	}
}

// Holy Shield states the damage of a block on its trigger-damage aura
// (110, 153, 221 by rank, 0.08 spell power), not on a spell of its own.
func TestHolyShieldDamageMatchesClient(t *testing.T) {
	const procDamageAura = 43
	client := clientdamagetest.Load(t, clientPaladinSpellconst)
	for _, rank := range HolyShieldValues {
		spell, ok := client.ByID(rank.spellID)
		if !ok {
			t.Fatalf("Holy Shield %d is not in the client table", rank.spellID)
		}
		var found bool
		for _, effect := range spell.Effects {
			if effect.Aura != procDamageAura {
				continue
			}
			found = true
			if effect.Amount != rank.damage {
				t.Errorf("Holy Shield %d: engine damage %v, client %v", rank.spellID, rank.damage, effect.Amount)
			}
			if effect.SPCoefficient != 0 && (effect.SPCoefficient-holyShieldCoefficient > 0.005 || holyShieldCoefficient-effect.SPCoefficient > 0.005) {
				t.Errorf("Holy Shield %d: engine coefficient %v, client %v", rank.spellID, holyShieldCoefficient, effect.SPCoefficient)
			}
		}
		if !found {
			t.Errorf("Holy Shield %d has no trigger-damage effect in the client table", rank.spellID)
		}
	}
}
