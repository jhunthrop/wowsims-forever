package priest

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/common/clientdamage/clientdamagetest"
	"github.com/wowsims/classic/sim/core/spellconst"
)

// damageLadder is one spell's generated per-rank tables, as the abilities
// read them.
type damageLadder struct {
	name     string
	ranks    int
	spellID  []int32
	level    []int
	mana     []float64
	coeff    []float64
	baseDmg  [][]float64
	perLevel []float64
	maxLevel []int
}

var damageCheckLevels = []int{10, 20, 30, 40, 50, 60}

func damageLadders() []damageLadder {
	return []damageLadder{
		{"Mind Blast", MindBlastRanks, MindBlastSpellId[:], MindBlastLevel[:], MindBlastManaCost[:], MindBlastSpellCoeff[:], MindBlastBaseDamage[:], MindBlastPointsPerLevel[:], MindBlastMaxLevel[:]},
		{"Shadow Word: Death", ShadowWordDeathRanks, ShadowWordDeathSpellId[:], ShadowWordDeathLevel[:], ShadowWordDeathManaCost[:], ShadowWordDeathSpellCoeff[:], ShadowWordDeathBaseDamage[:], ShadowWordDeathPointsPerLevel[:], ShadowWordDeathMaxLevel[:]},
		{"Mind Flay", MindFlayRanks, MindFlaySpellId[:], MindFlayLevel[:], MindFlayManaCost[:], MindFlaySpellCoeff[:], MindFlayBaseDamage[:], MindFlayPointsPerLevel[:], MindFlayMaxLevel[:]},
		{"Devouring Plague", DevouringPlagueRanks, DevouringPlagueSpellId[:], DevouringPlagueLevel[:], DevouringPlagueManaCost[:], DevouringPlagueSpellCoeff[:], DevouringPlagueBaseDamage[:], DevouringPlaguePointsPerLevel[:], DevouringPlagueMaxLevel[:]},
		{"Smite", SmiteRanks, SmiteSpellId[:], SmiteLevel[:], SmiteManaCost[:], SmiteSpellCoeff[:], SmiteBaseDamage[:], SmitePointsPerLevel[:], SmiteMaxLevel[:]},
		{"Holy Fire", HolyFireRanks, HolyFireSpellId[:], HolyFireLevel[:], HolyFireManaCost[:], HolyFireSpellCoeff[:], HolyFireBaseDamage[:], HolyFirePointsPerLevel[:], HolyFireMaxLevel[:]},
		{"Starshards", StarshardsRanks, StarshardsSpellId[:], StarshardsLevel[:], StarshardsManaCost[:], StarshardsSpellCoeff[:], StarshardsBaseDamage[:], StarshardsPointsPerLevel[:], StarshardsMaxLevel[:]},
	}
}

func TestPriestDamageLaddersMatchTheClient(t *testing.T) {
	client := clientdamagetest.LoadClient(t, "..", "priest")
	for _, ladder := range damageLadders() {
		for rank := 1; rank <= ladder.ranks; rank++ {
			id := ladder.spellID[rank]
			spell, ok := client.ByID(id)
			if !ok {
				t.Fatalf("%s rank %d: id %d is not in the client table", ladder.name, rank, id)
			}
			if ladder.level[rank] != spell.SpellLevel {
				t.Errorf("%s rank %d (%d): level %d, client %d", ladder.name, rank, id, ladder.level[rank], spell.SpellLevel)
			}
			if math.Abs(ladder.mana[rank]-spell.Cost) > 0.5 {
				t.Errorf("%s rank %d (%d): mana %v, client %v", ladder.name, rank, id, ladder.mana[rank], spell.Cost)
			}
			effect := spell.Effects[0]
			if effect.CoefficientSource == "table" && math.Abs(ladder.coeff[rank]-effect.ResolvedSPCoefficient) > 0.002 {
				t.Errorf("%s rank %d (%d): coefficient %v, client %v", ladder.name, rank, id, ladder.coeff[rank], effect.ResolvedSPCoefficient)
			}
			for _, casterLevel := range damageCheckLevels {
				roll := clientdamage.Roll(ladder.baseDmg[rank], ladder.perLevel[rank], ladder.level[rank], ladder.maxLevel[rank], casterLevel)
				clientdamagetest.AssertRoll(t, client, ladder.name, id, effectIndexOf(t, spell), casterLevel, roll)
			}
		}
	}
}

// effectIndexOf is the effect the generated ladder describes: the school
// damage effect, else the first effect (the generator's primary effect).
func effectIndexOf(t *testing.T, spell spellconst.Spell) int {
	t.Helper()
	for _, effect := range spell.Effects {
		if effect.Effect == 2 {
			return effect.Index
		}
	}
	return spell.Effects[0].Index
}

func TestShadowWordPainTicksMatchTheClient(t *testing.T) {
	client := clientdamagetest.LoadClient(t, "..", "priest")
	for rank := 1; rank <= ShadowWordPainRanks; rank++ {
		table := shadowWordPainRankTable(rank)
		spell, ok := client.ByID(table.spellID)
		if !ok {
			t.Fatalf("rank %d: id %d is not in the client table", rank, table.spellID)
		}
		if table.level != spell.SpellLevel || math.Abs(table.manaCost-spell.Cost) > 0.5 {
			t.Errorf("rank %d (%d): level %d mana %v, client %d / %v", rank, table.spellID, table.level, table.manaCost, spell.SpellLevel, spell.Cost)
		}
		if math.Abs(table.coeff-spell.Effects[0].ResolvedSPCoefficient) > 0.002 {
			t.Errorf("rank %d: coefficient %v, client %v", rank, table.coeff, spell.Effects[0].ResolvedSPCoefficient)
		}
		for _, casterLevel := range damageCheckLevels {
			clientdamagetest.AssertRoll(t, client, "Shadow Word: Pain", table.spellID, 0, casterLevel, table.tickRoll(casterLevel))
		}
	}
}

func TestHolyFireDotTicksMatchTheClient(t *testing.T) {
	client := clientdamagetest.LoadClient(t, "..", "priest")
	for rank := 1; rank <= HolyFireRanks; rank++ {
		spell, _ := client.ByID(HolyFireSpellId[rank])
		effect := spell.Effects[1]
		if math.Abs(holyFireDotTickDamage[rank]-effect.Amount) > 0.5 {
			t.Errorf("rank %d: dot tick %v, client %v", rank, holyFireDotTickDamage[rank], effect.Amount)
		}
		if effect.PeriodMS != holyFireDotTickMS || spell.DurationMS/effect.PeriodMS != holyFireDotTicks {
			t.Errorf("rank %d: client period %d duration %d, engine %d ticks every %d ms", rank, effect.PeriodMS, spell.DurationMS, holyFireDotTicks, holyFireDotTickMS)
		}
	}
}

func TestDevouringPlagueTicksFollowTheClientDuration(t *testing.T) {
	client := clientdamagetest.LoadClient(t, "..", "priest")
	spell, _ := client.ByID(DevouringPlagueSpellId[DevouringPlagueRanks])
	if got := int(spell.DurationMS / spell.Effects[0].PeriodMS); got != devouringPlagueTicks {
		t.Errorf("Devouring Plague ticks %d, client duration/period %d", devouringPlagueTicks, got)
	}
}
