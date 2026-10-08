package restoration

import (
	"fmt"
	"testing"

	"github.com/wowsims/classic/sim/core"
)

// learnedRanks counts the ranks a family has registered.
func learnedRanks(spells []*core.Spell) int {
	var count int
	for _, spell := range spells {
		if spell != nil {
			count++
		}
	}
	return count
}

// TestHealingRanksFollowTheCharactersLevel pins how many ranks of each
// healing spell a character has at each level, from the learn levels the
// client's trainables state (Healing Wave 1, 6, 12, 18, 24, 32, 40, 48, 56,
// 60; Lesser Healing Wave 20, 28, 36, 44, 52, 60; Chain Heal 40, 46, 54;
// Riptide 40, 50, 60 and Mana Tide Totem 40, 48, 58, both with their
// talent).
func TestHealingRanksFollowTheCharactersLevel(t *testing.T) {
	talents := restoTalents(map[int]int{posRiptide: 1, posManaTideTotem: 1})

	for _, tc := range []struct {
		level                                         int32
		healingWave, lesser, chain, riptide, manaTide int
	}{
		{1, 1, 0, 0, 0, 0},
		{10, 2, 0, 0, 0, 0},
		{20, 4, 1, 0, 0, 0},
		{30, 5, 2, 0, 0, 1},
		{38, 6, 3, 0, 0, 1},
		{40, 7, 3, 1, 1, 1},
		{50, 8, 4, 2, 2, 2},
		{60, 10, 6, 3, 3, 3},
	} {
		t.Run(fmt.Sprintf("L%d", tc.level), func(t *testing.T) {
			_, healer := newHealer(t, tc.level, talents)

			for name, got := range map[string][2]int{
				"Healing Wave":        {learnedRanks(healer.HealingWave), tc.healingWave},
				"Lesser Healing Wave": {learnedRanks(healer.LesserHealingWave), tc.lesser},
				"Chain Heal":          {learnedRanks(healer.ChainHeal), tc.chain},
				"Riptide":             {learnedRanks(healer.Riptide), tc.riptide},
				"Mana Tide Totem":     {learnedRanks(healer.ManaTideTotem), tc.manaTide},
			} {
				if got[0] != got[1] {
					t.Errorf("%s: %d ranks, want %d", name, got[0], got[1])
				}
			}
		})
	}
}

func TestEveryRegisteredRankIsOneTheLevelHasLearned(t *testing.T) {
	_, healer := newHealer(t, 38, restoTalents(map[int]int{posRiptide: 1, posManaTideTotem: 1}))

	for _, family := range [][]*core.Spell{healer.HealingWave, healer.LesserHealingWave, healer.ChainHeal, healer.Riptide, healer.ManaTideTotem} {
		for _, spell := range family {
			if spell != nil && spell.RequiredLevel > int(healer.Level) {
				t.Errorf("%s needs level %d but the character is %d", spell.ActionID, spell.RequiredLevel, healer.Level)
			}
		}
	}
}
