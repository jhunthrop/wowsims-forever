package warlock

import "testing"

// TestDemonArmorRankAtLevel locks in the old SoD bracket map's values at
// level 60 (byte-identical) and exercises level 35, between Demon
// Armor's real rank-1 learn level (20) and rank-3 learn level (40) - the
// rank the old code cycled through as its "40" bracket.
func TestDemonArmorRankAtLevel(t *testing.T) {
	cases := []struct {
		level     int32
		wantOK    bool
		spellID   int32
		armor     float64
		shadowRes float64
	}{
		{19, false, 0, 0, 0},
		{20, true, 706, 210.0, 3.0},
		{35, true, 706, 210.0, 3.0},
		{40, true, 11733, 390.0, 9.0},
		{50, true, 11734, 480.0, 12.0},
		{60, true, 11735, 570.0, 15.0}, // old bracket map's level-60 entry
	}
	for _, c := range cases {
		r, ok := demonArmorRankAtLevel(c.level)
		if ok != c.wantOK {
			t.Fatalf("level %d: ok = %v, want %v", c.level, ok, c.wantOK)
		}
		if !ok {
			continue
		}
		if r.spellID != c.spellID || r.armor != c.armor || r.shadowRes != c.shadowRes {
			t.Errorf("level %d: got %+v, want {spellID:%d armor:%v shadowRes:%v}", c.level, r, c.spellID, c.armor, c.shadowRes)
		}
	}
}
