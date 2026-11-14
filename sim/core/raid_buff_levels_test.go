package core

import "testing"

// Round 8 (auras lane): the raid buffs the engine held at one value for
// every level. Each row is the client's rank for that level (1.60.1.70707
// SpellEffect with the hotfix push, SpellLevels; Wowhead Forever's tooltips
// agree): the rows are pinned against the CSVs by the site's
// sim/leveling/buff_ranks_client_test.go.

type levelWant struct {
	level int
	want  float64
}

func checkRanks(t *testing.T, name string, ranks BuffRanks, cases []levelWant) {
	t.Helper()
	for _, c := range cases {
		if got := ranks.At(c.level); got != c.want {
			t.Errorf("%s at level %d = %v, want %v", name, c.level, got, c.want)
		}
	}
}

// Power Word: Fortitude 1243, 1244, 1245, 2791, 10937 and 10938 (levels 1,
// 12, 24, 36, 48, 60): 4, 10, 26, 42, 56 and 70 Stamina (Wowhead Forever:
// 2791 "by 42", 10938 "by 70"). Forever folded
// Improved Power Word: Fortitude's 30% into the base ranks; the engine gave
// Era's 54 at every level.
func TestPowerWordFortitudeFollowsTheClientRanks(t *testing.T) {
	checkRanks(t, "Power Word: Fortitude", PowerWordFortitudeRanks, []levelWant{
		{1, 4}, {11, 4}, {12, 10}, {24, 26}, {36, 42}, {48, 56}, {59, 56}, {60, 70},
	})
}

// Divine Spirit 14752, 14818, 14819, 27841 (levels 30, 40, 50, 60): 17,
// 23, 33 and 40 Spirit, and nothing before 30. The engine gave 40 from
// level 1, to every priest's own sim too (priest.AddRaidBuffs).
func TestDivineSpiritFollowsTheClientRanks(t *testing.T) {
	checkRanks(t, "Divine Spirit", DivineSpiritRanks, []levelWant{
		{20, 0}, {29, 0}, {30, 17}, {40, 23}, {50, 33}, {59, 33}, {60, 40},
	})
}

// Blood Pact 6307, 7804, 7805, 11766, 11767: base points plus points per
// level from the rank's level to its SpellLevels MaxLevel. Rank 5 reads 54
// at 60 (Wowhead Forever 11767: "Increases party members' Stamina by 54");
// the engine gave Era's 42 at every level.
func TestBloodPactFollowsTheClientRanks(t *testing.T) {
	checkRanks(t, "Blood Pact", BloodPactRanks, []levelWant{
		{3, 0}, {4, 3}, {13, 3}, {14, 9}, {20, 10}, {26, 21}, {30, 22}, {38, 35}, {49, 39}, {50, 49}, {60, 54},
	})
}

// Windfury Totem 8512, 10613, 10614 are learned at 32, 42 and 52; their
// strikes 8516, 10608, 10610 carry 95, 179 and 246 attack power. Below 32 a
// raid has no Windfury Totem. The engine gave rank 3 at every level.
func TestWindfuryTotemFollowsTheClientRanks(t *testing.T) {
	for _, c := range []struct {
		level int32
		rank  int32
		ap    float64
	}{{20, 0, 0}, {31, 0, 0}, {32, 1, 95}, {41, 1, 95}, {42, 2, 179}, {51, 2, 179}, {52, 3, 246}, {60, 3, 246}} {
		rank := WindfuryTotemRankAtLevel(c.level)
		if rank != c.rank {
			t.Errorf("Windfury Totem at level %d = rank %d, want %d", c.level, rank, c.rank)
			continue
		}
		if rank > 0 && WindfuryBonusAP[rank] != c.ap {
			t.Errorf("Windfury Totem rank %d = %v attack power, want %v", rank, WindfuryBonusAP[rank], c.ap)
		}
	}
}

// Leader of the Pack (17007) is tier 4 of the Feral tree and Moonkin Form
// (24858, the Moonkin Aura 24907) tier 6 of Balance: a druid takes them at
// 30 and 40 at the earliest. Below that a raid has neither aura.
func TestTalentRaidAurasWaitForTheirTalentLevel(t *testing.T) {
	for _, c := range []struct {
		level         int32
		pack, moonkin bool
	}{{20, false, false}, {29, false, false}, {30, true, false}, {39, true, false}, {40, true, true}, {60, true, true}} {
		if got := LeaderOfThePackReaches(c.level); got != c.pack {
			t.Errorf("Leader of the Pack at level %d = %v, want %v", c.level, got, c.pack)
		}
		if got := MoonkinAuraReaches(c.level); got != c.moonkin {
			t.Errorf("Moonkin Aura at level %d = %v, want %v", c.level, got, c.moonkin)
		}
	}
}
