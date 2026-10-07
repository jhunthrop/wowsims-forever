package warrior

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

// Battle Shout grants the client's amount for the rank the warrior has
// learned (spells 6673 .. 25289), not the vanilla AQ-era 232 at every
// level. Inside a rank it grows by the client's points per level.
func TestBattleShoutGrantFollowsTheClientRanks(t *testing.T) {
	cases := []struct {
		level  int32
		rank   int32
		spell  int32
		attack float64
	}{
		{1, 1, 6673, 9},
		{12, 2, 5242, 21},
		{22, 3, 6192, 33},
		{32, 4, 11549, 51},
		{40, 4, 11549, 55},
		{42, 5, 11550, 78},
		{52, 6, 11551, 111},
	}
	// Rank 7 is an Ahn'Qiraj book rank: a level-60 warrior without
	// IncludeAQ casts rank 6, which keeps growing 0.6 a level to 61.
	if core.IncludeAQ {
		cases = append(cases, struct {
			level  int32
			rank   int32
			spell  int32
			attack float64
		}{60, 7, 25289, 139})
	} else {
		cases = append(cases, struct {
			level  int32
			rank   int32
			spell  int32
			attack float64
		}{60, 6, 11551, 115})
	}
	for _, c := range cases {
		rank, spell, attack := battleShoutGrant(c.level)
		if rank != c.rank || spell != c.spell || attack != c.attack {
			t.Errorf("level %d: rank %d spell %d attack power %v, want rank %d spell %d attack power %v",
				c.level, rank, spell, attack, c.rank, c.spell, c.attack)
		}
	}
	if core.BattleShoutAttackPower(60) == 232 {
		t.Error("level 60 Battle Shout still grants the vanilla 232")
	}
}
