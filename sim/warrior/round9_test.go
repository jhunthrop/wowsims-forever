package warrior

import "testing"

// Round 9 (physical lane): a Fury warrior from 30 to 35 knows Berserker
// Stance but has no Berserker-only ability yet (Whirlwind is 36,
// Bloodthirst the 31-point talent), so Berserker Stance gave it 3% crit and
// took away Overpower. Opening in Battle Stance measured +3.9% at band 30
// bare (horde, 3,000 iterations, seeds 1 and 50001). Fury opens in
// Berserker Stance once Whirlwind is learned.
func TestFuryOpensInBattleStanceUntilWhirlwind(t *testing.T) {
	for _, tc := range []struct {
		level int32
		want  Stance
	}{{20, BattleStance}, {30, BattleStance}, {35, BattleStance}, {36, BerserkerStance}, {60, BerserkerStance}} {
		w := warriorAtLevel(tc.level)
		w.PrimaryTalentTree = FuryTree
		if got := w.defaultStance(); got != tc.want {
			t.Errorf("level %d Fury default stance = %v, want %v", tc.level, got, tc.want)
		}
	}
}
