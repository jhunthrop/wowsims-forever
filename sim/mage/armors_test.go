package mage

import "testing"

// TestFrostIceArmorRankAtLevel locks in the old SoD bracket map's values
// at level 60 (byte-identical) and exercises a level between two of the
// real client learn levels (38, Ice Armor rank 1 at 30 until rank 2 at 40) to show the aura now scales continuously instead
// of only at 25/40/50/60.
func TestFrostIceArmorRankAtLevel(t *testing.T) {
	cases := []struct {
		level    int32
		wantOK   bool
		spellID  int32
		armor    float64
		frostRes float64
	}{
		{0, false, 0, 0, 0},
		{1, true, 168, 30, 0},
		{10, true, 7300, 110, 0},
		{19, true, 7300, 110, 0},
		{20, true, 7301, 200, 0},
		{30, true, 7302, 290, 6},
		{38, true, 7302, 290, 6},
		{40, true, 7320, 380, 9},
		{50, true, 10219, 470, 12},
		{60, true, 10220, 560, 15}, // old bracket map's level-60 entry
	}
	for _, c := range cases {
		r, ok := frostIceArmorRankAtLevel(c.level)
		if ok != c.wantOK {
			t.Fatalf("level %d: ok = %v, want %v", c.level, ok, c.wantOK)
		}
		if !ok {
			continue
		}
		if r.spellID != c.spellID || r.armor != c.armor || r.frostRes != c.frostRes {
			t.Errorf("level %d: got %+v, want {spellID:%d armor:%v frostRes:%v}", c.level, r, c.spellID, c.armor, c.frostRes)
		}
	}
}

// TestMageArmorRankAtLevel locks in the old SoD bracket map's values at
// level 60 (byte-identical) and exercises level 38, between Mage Armor's
// real rank-1 learn level (34) and rank-2 learn level (46).
// TestMageArmorKeepsHalfOfRegenWhileCasting pins the client's aura 134
// amount (50) on every Mage Armor rank.
func TestMageArmorKeepsHalfOfRegenWhileCasting(t *testing.T) {
	if mageArmorCastingRegen != 0.5 {
		t.Errorf("mageArmorCastingRegen = %v, want 0.5 (client aura 134 amount 50)", mageArmorCastingRegen)
	}
}

func TestMageArmorRankAtLevel(t *testing.T) {
	cases := []struct {
		level    int32
		wantOK   bool
		spellID  int32
		spellRes float64
	}{
		{33, false, 0, 0},
		{34, true, 6117, 5},
		{38, true, 6117, 5},
		{46, true, 22782, 10},
		{58, true, 22783, 15},
		{60, true, 22783, 15}, // old bracket map's level-60 entry
	}
	for _, c := range cases {
		r, ok := mageArmorRankAtLevel(c.level)
		if ok != c.wantOK {
			t.Fatalf("level %d: ok = %v, want %v", c.level, ok, c.wantOK)
		}
		if !ok {
			continue
		}
		if r.spellID != c.spellID || r.spellRes != c.spellRes {
			t.Errorf("level %d: got %+v, want {spellID:%d spellRes:%v}", c.level, r, c.spellID, c.spellRes)
		}
	}
}
