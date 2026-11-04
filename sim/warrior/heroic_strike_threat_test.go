package warrior

import "testing"

// Heroic Strike's bonus threat by rank. The client carries no threat
// effect for it, so the figures are the Classic server's spell_threat rows
// (mangoszero/database World/Setup/FullDB/spell_threat.sql; the same rows
// in mangosArchives databaseZero_Rel19 and tortoise-wow): 20, 39, 59, 78,
// 98, 118, 137, 145 and 175. The engine's level-60 145 is rank 8's row;
// before this test every lower rank a levelling warrior casts got it too.
func TestHeroicStrikeThreatIsPerRank(t *testing.T) {
	want := map[int]float64{1: 20, 2: 39, 3: 59, 4: 78, 5: 98, 6: 118, 7: 137, 8: 145, 9: 175}
	for rank := 1; rank <= HeroicStrikeRanks; rank++ {
		if got := heroicStrikeThreat(rank); got != want[rank] {
			t.Errorf("Heroic Strike rank %d (spell %d): threat %v, want %v", rank, heroicStrikeRankSpellID(rank), got, want[rank])
		}
	}
}
