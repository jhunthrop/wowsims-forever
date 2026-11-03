package core

import (
	"testing"

	"github.com/wowsims/classic/sim/core/stats"
)

// The rows themselves are pinned against the client CSVs by the site's
// sim/leveling/buff_ranks_client_test.go; these tests pin the lookup.

func TestLevelSixtyBuffsAreTheClientTopRanks(t *testing.T) {
	if got := ArcaneIntellectStats(60)[stats.Intellect]; got != 31 {
		t.Errorf("Arcane Intellect at 60 = %v, want 31", got)
	}
	// Rank 7 of both is an Ahn'Qiraj book rank, so the level-60 value
	// follows IncludeAQ: the book rank with it on, rank 6 (grown to 60 for
	// Battle Shout) with it off.
	if got, want := BlessingOfMightAttackPower(60), TernaryFloat64(IncludeAQ, 133, 112); got != want {
		t.Errorf("Blessing of Might at 60 = %v, want %v", got, want)
	}
	if got, want := BattleShoutAttackPower(60), TernaryFloat64(IncludeAQ, 139, 115); got != want {
		t.Errorf("Battle Shout at 60 = %v, want %v", got, want)
	}
	mark := MarkOfTheWildStats(60)
	if mark[stats.BonusArmor] != 385 || mark[stats.Strength] != 16 || mark[stats.FrostResistance] != 27 {
		t.Errorf("Mark of the Wild at 60 = %v, want armor 385, stats 16, resistances 27", mark)
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
	cases := []struct {
		level int
		want  float64
	}{{4, 14}, {11, 14}, {12, 25}, {22, 40}, {32, 61}, {42, 83}, {51, 83}, {52, 112}, {60, TernaryFloat64(IncludeAQ, 133, 112)}}
	for _, c := range cases {
		if got := BlessingOfMightAttackPower(c.level); got != c.want {
			t.Errorf("Blessing of Might at level %d = %v, want %v", c.level, got, c.want)
		}
	}
}

func TestMarkOfTheWildFollowsClientRanks(t *testing.T) {
	low := MarkOfTheWildStats(1)
	if low[stats.Strength] != 0 || low[stats.BonusArmor] != 34 {
		t.Errorf("rank 1 should carry 34 armor only, got %v", low)
	}
	mid := MarkOfTheWildStats(30)
	if mid[stats.BonusArmor] != 203 || mid[stats.Strength] != 8 || mid[stats.NatureResistance] != 7 {
		t.Errorf("rank 4 = %v, want armor 203, stats 8, resistances 7", mid)
	}
}

// Battle Shout grows by the client's points per level inside a rank, capped
// at the rank's max level.
func TestBattleShoutGrowsWithinARank(t *testing.T) {
	cases := []struct {
		level int
		want  float64
	}{{1, 9}, {11, 12}, {12, 21}, {32, 51}, {41, 56}, {52, 111}, {60, TernaryFloat64(IncludeAQ, 139, 115)}, {70, TernaryFloat64(IncludeAQ, 139, 116)}}
	for _, c := range cases {
		if got := BattleShoutAttackPower(c.level); got != c.want {
			t.Errorf("Battle Shout at level %d = %v, want %v", c.level, got, c.want)
		}
	}
}

func TestBattleShoutArraysAgreeWithTheRankTable(t *testing.T) {
	for i, rank := range BattleShoutRankTable {
		if BattleShoutSpellId[i+1] != rank.SpellID || BattleShoutLevel[i+1] != rank.Level {
			t.Errorf("rank %d: arrays say %d at %d, table says %d at %d", i+1, BattleShoutSpellId[i+1], BattleShoutLevel[i+1], rank.SpellID, rank.Level)
		}
	}
}

// Rank 7 of Blessing of Might and Battle Shout are Ahn'Qiraj book ranks:
// without IncludeAQ a level-60 character resolves to the top trainer rank
// (rank 6, learned at 52); with it, to rank 7. Mark of the Wild rank 7
// (9885) and Arcane Intellect rank 5 are trainer ranks either way.
func TestAhnQirajBookRanksFollowIncludeAQ(t *testing.T) {
	if got := BlessingOfMightRanks.at(60, false); got != 112 {
		t.Errorf("Blessing of Might at 60 without AQ = %v, want 112", got)
	}
	if got := BlessingOfMightRanks.at(60, true); got != 133 {
		t.Errorf("Blessing of Might at 60 with AQ = %v, want 133", got)
	}
	rank, _ := BattleShoutRankTable.learned(60, false)
	if rank.SpellID != 11551 || rank.Amount != 111 {
		t.Errorf("Battle Shout at 60 without AQ = rank %d amount %v, want 11551 / 111", rank.SpellID, rank.Amount)
	}
	if rank, _ := BattleShoutRankTable.learned(60, true); rank.SpellID != 25289 || BattleShoutRankTable.at(60, true) != 139 {
		t.Errorf("Battle Shout at 60 with AQ = rank %d, want 25289 / 139", rank.SpellID)
	}
	for _, includeAQ := range []bool{false, true} {
		if got := MarkOfTheWildArmorRanks.at(60, includeAQ); got != 385 {
			t.Errorf("Mark of the Wild armor at 60 (AQ %v) = %v, want 385", includeAQ, got)
		}
		if got := ArcaneIntellectRanks.at(60, includeAQ); got != 31 {
			t.Errorf("Arcane Intellect at 60 (AQ %v) = %v, want 31", includeAQ, got)
		}
	}
}

// Trueshot Aura since the server's 70753 hotfix push (2026-11-03):
// 20/35/50/75/100 ranged attack power (70707 had cut it to 15/22/30/40/50),
// so level 60 takes rank 5; before level 25 there is no rank at all.
func TestTrueshotAuraFollowsThe70753Push(t *testing.T) {
	cases := []struct {
		level int
		want  float64
	}{{24, 0}, {25, 20}, {32, 35}, {40, 50}, {50, 75}, {59, 75}, {60, 100}}
	for _, c := range cases {
		if got := TrueshotAuraRanks.At(c.level); got != c.want {
			t.Errorf("Trueshot Aura at level %d = %v, want %v", c.level, got, c.want)
		}
	}
	if rank, _ := TrueshotAuraRanks.Learned(60); rank.SpellID != 20906 {
		t.Errorf("Trueshot Aura learned at 60 = %d, want 20906 (rank 5)", rank.SpellID)
	}
}

// Devotion Aura is a paladin's own armor aura. The client states seven
// ranks (spells 465 to 10293), 55 armor at level 1 to 735 at level 60,
// and the engine used to give every level the level-60 amount.
func TestDevotionAuraFollowsClientRanks(t *testing.T) {
	cases := []struct {
		level int
		want  float64
	}{{1, 55}, {9, 55}, {10, 160}, {19, 160}, {20, 275}, {30, 390}, {40, 505}, {50, 620}, {59, 620}, {60, 735}}
	for _, c := range cases {
		if got := DevotionAuraArmor(c.level); got != c.want {
			t.Errorf("Devotion Aura at level %d = %v armor, want %v", c.level, got, c.want)
		}
	}
}

// Blessing of Wisdom restores the client's mana per five seconds for the
// rank a character has learned: the client states 12, 18, 24, 30 and 36
// for the trainer ranks and 40 for the Ahn'Qiraj book rank, where the
// engine used to give every level the vanilla 30.
func TestBlessingOfWisdomFollowsTheClientRanks(t *testing.T) {
	cases := []struct {
		level int
		want  float64
	}{{13, 0}, {14, 12}, {24, 18}, {34, 24}, {44, 30}, {54, 36}, {60, 36}}
	for _, c := range cases {
		if got := BlessingOfWisdomMP5(c.level); got != c.want {
			t.Errorf("Blessing of Wisdom at level %d = %v, want %v", c.level, got, c.want)
		}
	}
	if got := BlessingOfWisdomRanks.at(60, true); got != 40 {
		t.Errorf("Blessing of Wisdom at 60 with the Ahn'Qiraj book = %v, want 40", got)
	}
}

// The client scales from SpellLevels.BaseLevel (0 for 409583), not its
// SpellLevel (1): Wowhead Forever's buff reads 280 at 60 (ppl0:60:40:400).
func TestHeartOfTheLionAttackPowerFollowsClientRule(t *testing.T) {
	cases := []struct {
		level int
		want  float64
	}{{1, 44}, {2, 48}, {30, 160}, {59, 276}, {60, 280}, {61, 280}}
	for _, c := range cases {
		if got := HeartOfTheLionRanks.At(c.level); got != c.want {
			t.Errorf("Heart of the Lion attack power at level %d = %v, want %v", c.level, got, c.want)
		}
	}
}

// Strength of Earth and Grace of Air Totem: Forever retuned both. The
// totems' auras state Strength 7/14/25/42 at levels 10/24/38/52 and 53 at
// 60 (8076, 8162, 8163, 10441, 25362), and Agility 49/77 at 42/56 and 89 at
// 60 (8836, 10626, 25360); the level-60 ranks are the Ahn'Qiraj books
// (25361, 25359). Wowhead Forever's 10442 reads "by 42" and 10627 "by 77".
// The engine gave every level Era's 61 Strength and 67 Agility.
func TestEarthAndAirTotemsFollowTheClientRanks(t *testing.T) {
	strength := []struct {
		level int
		want  float64
	}{{9, 0}, {10, 7}, {24, 14}, {38, 25}, {51, 25}, {52, 42}, {60, TernaryFloat64(IncludeAQ, 53, 42)}}
	for _, c := range strength {
		if got := StrengthOfEarthRanks.At(c.level); got != c.want {
			t.Errorf("Strength of Earth at level %d = %v, want %v", c.level, got, c.want)
		}
	}
	agility := []struct {
		level int
		want  float64
	}{{41, 0}, {42, 49}, {55, 49}, {56, 77}, {60, TernaryFloat64(IncludeAQ, 89, 77)}}
	for _, c := range agility {
		if got := GraceOfAirRanks.At(c.level); got != c.want {
			t.Errorf("Grace of Air at level %d = %v, want %v", c.level, got, c.want)
		}
	}
}

// Mana Spring Totem (5675, 10495, 10496, 10497 at levels 26, 36, 46, 56;
// their auras 5677, 10491, 10493, 10494 restore 4, 6, 8 and 10 mana every
// 2 s): 10, 15, 20 and 25 mana per 5 s. There is no Ahn'Qiraj rank. The
// engine gave every level rank 4's 25.
func TestManaSpringFollowsTheClientRanks(t *testing.T) {
	for _, c := range []struct {
		level int
		want  float64
	}{{25, 0}, {26, 10}, {35, 10}, {36, 15}, {46, 20}, {55, 20}, {56, 25}, {60, 25}} {
		if got := ManaSpringMP5(c.level); got != c.want {
			t.Errorf("Mana Spring at level %d = %v mp5, want %v", c.level, got, c.want)
		}
	}
}
