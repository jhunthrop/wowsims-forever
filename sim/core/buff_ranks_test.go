package core

import (
	"testing"

	"github.com/wowsims/classic/sim/core/stats"
)

// The level-60 values are the existing BuffSpellValues entries, bit for
// bit: no golden moves.
func TestLevelSixtyBuffsMatchLegacyValues(t *testing.T) {
	if got, want := ArcaneIntellectStats(60), BuffSpellValues[ArcaneIntellect]; got != want {
		t.Fatalf("Arcane Intellect at 60 = %v, want %v", got, want)
	}
	if got, want := BlessingOfMightAttackPower(60), BuffSpellValues[BlessingOfMight][stats.AttackPower]; got != want {
		t.Fatalf("Blessing of Might at 60 = %v, want %v", got, want)
	}
	if got, want := MarkOfTheWildStats(60), BuffSpellValues[MarkOfTheWild]; got != want {
		t.Fatalf("Mark of the Wild at 60 = %v, want %v", got, want)
	}
}

func TestArcaneIntellectFollowsClientRanks(t *testing.T) {
	cases := []struct {
		level int
		want  float64
	}{{1, 2}, {13, 2}, {14, 7}, {28, 15}, {41, 15}, {42, 22}, {55, 22}, {56, 31}, {60, 31}}
	for _, c := range cases {
		if got := ArcaneIntellectStats(c.level)[stats.Intellect]; got != c.want {
			t.Errorf("Arcane Intellect at level %d = %v, want %v", c.level, got, c.want)
		}
	}
}

func TestBlessingOfMightFollowsClientRanks(t *testing.T) {
	if got := BlessingOfMightAttackPower(3); got != 0 {
		t.Errorf("Blessing of Might below level 4 = %v, want 0", got)
	}
	// Ranks keep the client's proportions, anchored on the level-60 value.
	prev := 0.0
	for _, level := range []int{4, 12, 22, 32, 42, 52, 60} {
		got := BlessingOfMightAttackPower(level)
		if got <= prev {
			t.Errorf("Blessing of Might at level %d = %v, not above the previous rank %v", level, got, prev)
		}
		prev = got
	}
	if BlessingOfMightAttackPower(51) != BlessingOfMightAttackPower(42) {
		t.Error("Blessing of Might should hold rank 5 from level 42 to 51")
	}
}

func TestMarkOfTheWildFollowsClientRanks(t *testing.T) {
	low := MarkOfTheWildStats(1)
	if low[stats.Strength] != 0 || low[stats.BonusArmor] <= 0 {
		t.Errorf("rank 1 should carry armor only, got %v", low)
	}
	mid := MarkOfTheWildStats(30)
	if mid[stats.Strength] <= 0 || mid[stats.NatureResistance] <= 0 {
		t.Errorf("rank 4 should carry stats and resistances, got %v", mid)
	}
	if mid[stats.BonusArmor] >= MarkOfTheWildStats(60)[stats.BonusArmor] {
		t.Error("armor should grow with rank")
	}
}
