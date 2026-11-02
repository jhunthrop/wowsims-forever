package core

import (
	"testing"

	"github.com/wowsims/classic/sim/core/stats"
)

// Round 8 (auras lane): the raid's debuffs are cast at the raid's level, so
// each takes the rank that level has learned (1.60.1.70707 SpellEffect with
// the hotfix push, SpellLevels). The engine applied every one at its top
// rank whatever the level.

// newDebuffTarget is a target unit whose raid is one player of level.
func newDebuffTarget(level int32) *Unit {
	return &Unit{
		Type:        EnemyUnit,
		Level:       63,
		auraTracker: newAuraTracker(),
		Env: &Environment{
			MeasuringStats: true,
			Raid:           &Raid{AllPlayerUnits: []*Unit{{Type: PlayerUnit, Level: level}}},
		},
	}
}

func TestRaidDebuffsAreCastAtTheRaidsLevel(t *testing.T) {
	if got := raidDebuffLevel(newDebuffTarget(34)); got != 34 {
		t.Errorf("raid debuff level = %d, want the raid's 34", got)
	}
	if got := raidDebuffLevel(&Unit{}); got != CharacterMaxLevel {
		t.Errorf("raid debuff level with no raid = %d, want %d", got, CharacterMaxLevel)
	}
}

// Sunder Armor 7386, 7405, 8380, 11596, 11597 (levels 10, 22, 34, 46, 58):
// 90, 180, 270, 360 and 450 armor a stack (aura 674, misc 1).
// Faerie Fire 770, 778, 9749, 9907 (18, 30, 42, 54): 175, 285, 395, 505.
// Curse of Recklessness 704, 7658, 7659, 11717 (14, 28, 42, 56): 175, 285,
// 395, 505 armor, and no attack power (its other effect is aura 92, the
// no-flee). Curse of the Elements 440892, 1311676, 1311677, 1311680 (20,
// 30, 40, 50): 4, 6, 8, 10% magic damage taken and 30, 45, 60, 75
// resistance. Hunter's Mark 1130, 14323, 14324, 14325 (6, 22, 40, 58): 26,
// 59, 98, 143 ranged attack power. Judgement of the Crusader 21183, 20188,
// 20300, 20301, 20302, 20303 (Seal levels 6, 12, 22, 32, 42, 52): 23, 35,
// 58, 92, 127, 161 Holy damage taken.
func TestRaidDebuffsFollowTheClientRanks(t *testing.T) {
	checkRanks(t, "Sunder Armor", SunderArmorRanks, []levelWant{{9, 0}, {10, 90}, {22, 180}, {34, 270}, {46, 360}, {57, 360}, {58, 450}, {60, 450}})
	checkRanks(t, "Faerie Fire", FaerieFireRanks, []levelWant{{17, 0}, {18, 175}, {30, 285}, {42, 395}, {53, 395}, {54, 505}, {60, 505}})
	checkRanks(t, "Curse of Recklessness", CurseOfRecklessnessRanks, []levelWant{{13, 0}, {14, 175}, {28, 285}, {42, 395}, {55, 395}, {56, 505}, {60, 505}})
	checkRanks(t, "Curse of the Elements damage", CurseOfTheElementsDamageRanks, []levelWant{{19, 0}, {20, 4}, {30, 6}, {40, 8}, {50, 10}, {60, 10}})
	checkRanks(t, "Curse of the Elements resistance", CurseOfTheElementsResistanceRanks, []levelWant{{19, 0}, {20, 30}, {30, 45}, {40, 60}, {50, 75}, {60, 75}})
	checkRanks(t, "Hunter's Mark", HuntersMarkRanks, []levelWant{{5, 0}, {6, 26}, {22, 59}, {40, 98}, {57, 98}, {58, 143}, {60, 143}})
	checkRanks(t, "Judgement of the Crusader", JudgementOfTheCrusaderRanks, []levelWant{{5, 0}, {6, 23}, {12, 35}, {22, 58}, {32, 92}, {42, 127}, {51, 127}, {52, 161}, {60, 161}})
	if top := HuntersMarkRanks[len(HuntersMarkRanks)-1]; top.SpellID != HuntersMarkTopRankSpellID || top.Amount != HuntersMarkTopRankRangedAttackPower {
		t.Errorf("Hunter's Mark top rank = %+v, want %d at %v", top, HuntersMarkTopRankSpellID, HuntersMarkTopRankRangedAttackPower)
	}
	if top := JudgementOfTheCrusaderRanks[len(JudgementOfTheCrusaderRanks)-1]; top.Amount != JudgementOfTheCrusaderRaidBonus {
		t.Errorf("Judgement of the Crusader top rank = %v, want %v", top.Amount, float64(JudgementOfTheCrusaderRaidBonus))
	}
}

// Curse of Recklessness takes armor only: Forever's rows carry no attack
// power effect (Era's gave the target 90).
func TestCurseOfRecklessnessTakesTheRanksArmorOnly(t *testing.T) {
	for _, c := range []struct {
		level int32
		armor float64
	}{{60, 505}, {30, 285}} {
		target := newDebuffTarget(c.level)
		aura := CurseOfRecklessnessAura(target)
		aura.Activate(&Simulation{})
		if got := target.GetStat(stats.Armor); got != -c.armor {
			t.Errorf("level %d: armor after Curse of Recklessness = %v, want %v", c.level, got, -c.armor)
		}
		if got := target.GetStat(stats.AttackPower); got != 0 {
			t.Errorf("level %d: Curse of Recklessness gave the target %v attack power, want 0", c.level, got)
		}
	}
}
