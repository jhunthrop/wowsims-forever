package warlock

import (
	"testing"

	"github.com/wowsims/classic/sim/common/clientdamage/clientdamagetest"
)

// TestDemonArmorRankAtLevel pins the rank worn at each level to the
// client's five ranks (706, 1086, 11733, 11734, 11735 at levels
// 20/30/40/50/60), rank 2 included.
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
		{29, true, 706, 210.0, 3.0},
		{30, true, 1086, 300.0, 6.0},
		{39, true, 1086, 300.0, 6.0},
		{40, true, 11733, 390.0, 9.0},
		{50, true, 11734, 480.0, 12.0},
		{60, true, 11735, 570.0, 15.0},
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

// Every rank's armor and Shadow resistance are the client's effects 0
// and 1 of the rank's spell.
func TestDemonArmorRanksMatchTheClient(t *testing.T) {
	class := clientdamagetest.Load(t, clientWarlockSpellconst)
	for rank := 1; rank <= DemonArmorRanks; rank++ {
		spell, ok := class.ByID(DemonArmorSpellId[rank])
		if !ok {
			t.Fatalf("rank %d: spell %d is not in the client table", rank, DemonArmorSpellId[rank])
		}
		if spell.SpellLevel != DemonArmorLevel[rank] {
			t.Errorf("rank %d: learned at %d, client %d", rank, DemonArmorLevel[rank], spell.SpellLevel)
		}
		got, _ := demonArmorRankAtLevel(int32(DemonArmorLevel[rank]))
		if got.spellID != DemonArmorSpellId[rank] {
			t.Errorf("rank %d: worn spell %d at level %d", rank, got.spellID, DemonArmorLevel[rank])
		}
		if want := spell.Effects[0].Amount; got.armor != want {
			t.Errorf("rank %d: armor %v, client %v", rank, got.armor, want)
		}
		if want := spell.Effects[1].Amount; got.shadowRes != want {
			t.Errorf("rank %d: Shadow resistance %v, client %v", rank, got.shadowRes, want)
		}
	}
}
