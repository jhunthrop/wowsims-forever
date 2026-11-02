package shaman

import (
	"math"
	"testing"
)

// Each Windfury Weapon rank's extra attack power is the attack power aura
// (aura 99) of its buff spell, which grows by EffectRealPointsPerLevel from
// the rank's level to its MaxLevel (1.60.1.70707 SpellEffect and
// SpellLevels): 8233 46 + 7.2 a level from 30 to 38, 8236 119 + 12.8 from
// 40 to 48, 10484 249 + 8.3 from 50 to 58, 16361 333 + 12.5 from 60 to 68.
// Wowhead Forever's tooltips render each rank at its MaxLevel: 104, 221,
// 315 and 433.
func TestWindfuryWeaponAttackPowerGrowsWithLevel(t *testing.T) {
	cases := []struct {
		rank, level int32
		want        float64
	}{
		{1, 30, 46},
		{1, 34, 46 + 7.2*4},
		{1, 39, 46 + 7.2*8},
		{2, 40, 119},
		{2, 45, 119 + 12.8*5},
		{3, 50, 249},
		{3, 55, 249 + 8.3*5},
		{4, 60, 333},
	}
	for _, c := range cases {
		if got := windfuryWeaponBonusAttackPower(c.rank, 0, c.level); math.Abs(got-c.want) > 1e-6 {
			t.Errorf("Windfury Weapon rank %d at level %d adds %v attack power, want %v", c.rank, c.level, got, c.want)
		}
	}
}

// Each Rockbiter Weapon rank's attack power is its enchant spell's aura 99
// (1.60.1.70707: 10400, 15567, 15568, 15569, 16311, 16312, 16313), growing
// a level to the rank's MaxLevel.
func TestRockbiterWeaponAttackPowerGrowsWithLevel(t *testing.T) {
	cases := []struct {
		rank, level int32
		want        float64
	}{
		{1, 1, 29},
		{1, 6, 29 + 4.1*5},
		{2, 10, 58 + 3.5*2},
		{3, 20, 88 + 5*4},
		{4, 24, 129},
		{4, 30, 129 + 8.1*6},
		{5, 40, 211 + 18*6},
		{6, 50, 393 + 16.1*6},
		{7, 60, 554 + 16.5*6},
	}
	for _, c := range cases {
		if got := rockbiterWeaponBonusAttackPower(c.rank, 0, c.level); math.Abs(got-c.want) > 1e-6 {
			t.Errorf("Rockbiter Weapon rank %d at level %d adds %v attack power, want %v", c.rank, c.level, got, c.want)
		}
	}
	if got, want := rockbiterWeaponBonusAttackPower(3, 3, 20), (88+5*4)*1.20; math.Abs(got-want) > 1e-6 {
		t.Errorf("Rockbiter rank 3 at 20 with Elemental Weapons 3/3 adds %v, want %v", got, want)
	}
}
