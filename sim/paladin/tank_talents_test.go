package paladin

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// The tiers and prerequisites are read from the UI tree the talent
// generator wrote from the same client tables as the proto, not retyped.
const generatedPaladinTrees = "../../ui/core/talents/trees/paladin.json"

type treeLocation struct {
	RowIdx int `json:"rowIdx"`
	ColIdx int `json:"colIdx"`
}

type treeTalent struct {
	FieldName      string       `json:"fieldName"`
	Location       treeLocation `json:"location"`
	MaxPoints      int          `json:"maxPoints"`
	PrereqLocation treeLocation `json:"prereqLocation"`
	PrereqRank     int          `json:"prereqRank"`
}

type talentTree struct {
	Name    string       `json:"name"`
	Talents []treeTalent `json:"talents"`
}

func loadPaladinTrees(t *testing.T) []talentTree {
	t.Helper()
	raw, err := os.ReadFile(generatedPaladinTrees)
	if err != nil {
		t.Fatalf("reading the generated tree: %v", err)
	}
	var trees []talentTree
	if err := json.Unmarshal(raw, &trees); err != nil {
		t.Fatalf("parsing the generated tree: %v", err)
	}
	if len(trees) != len(TalentTreeSizes) {
		t.Fatalf("the generated tree has %d trees, TalentTreeSizes has %d", len(trees), len(TalentTreeSizes))
	}
	for i, tree := range trees {
		if len(tree.Talents) != TalentTreeSizes[i] {
			t.Fatalf("%s has %d talents, TalentTreeSizes says %d", tree.Name, len(tree.Talents), TalentTreeSizes[i])
		}
	}
	return trees
}

// A build the client would refuse is a number nobody can reach: every
// segment is its tree's width, the points fit each talent's ranks, each
// tier is open (5 points spent above it) and each prerequisite is met.
func TestForeverProtectionTalentsAreLegal(t *testing.T) {
	segments := strings.Split(ForeverProtectionTalents, "-")
	if len(segments) != len(TalentTreeSizes) {
		t.Fatalf("ForeverProtectionTalents has %d segments, want %d", len(segments), len(TalentTreeSizes))
	}

	var total int
	spent := make([]int, len(segments))
	for i, tree := range loadPaladinTrees(t) {
		if len(segments[i]) != TalentTreeSizes[i] {
			t.Fatalf("%s segment is %d characters, want %d", tree.Name, len(segments[i]), TalentTreeSizes[i])
		}
		byLocation := map[treeLocation]int{}
		for j, talent := range tree.Talents {
			byLocation[talent.Location] = j
			if j > 0 && talent.Location.RowIdx < tree.Talents[j-1].Location.RowIdx {
				t.Fatalf("%s is not sorted by tier, so the cumulative gate below is wrong", tree.Name)
			}
		}

		for j, talent := range tree.Talents {
			points := int(segments[i][j] - '0')
			if points == 0 {
				continue
			}
			if points > talent.MaxPoints {
				t.Errorf("%s: %d points, the client's max rank is %d", talent.FieldName, points, talent.MaxPoints)
			}
			if gate := 5 * talent.Location.RowIdx; spent[i] < gate {
				t.Errorf("%s is tier %d and needs %d points above it, the build has %d", talent.FieldName, talent.Location.RowIdx, gate, spent[i])
			}
			if talent.PrereqRank > 0 {
				prereq := byLocation[talent.PrereqLocation]
				if got := int(segments[i][prereq] - '0'); got < talent.PrereqRank {
					t.Errorf("%s needs %s at rank %d, the build has %d", talent.FieldName, tree.Talents[prereq].FieldName, talent.PrereqRank, got)
				}
			}
			spent[i] += points
		}
		total += spent[i]
	}

	if total != 51 {
		t.Errorf("the build spends %d points, want 51", total)
	}
	if spent[1] != 31 {
		t.Errorf("the build spends %d points in Protection, want 31 to reach Holy Shield", spent[1])
	}
}

// Every Protection talent the reference build names exists in the tree
// the engine parses: a renamed field would silently drop out of the build.
func TestForeverProtectionTalentsTakeTheTalentsThisSpecModels(t *testing.T) {
	for _, field := range []string{"holy_shield", "templars_bulwark", "swift_judgement", "improved_seal_of_fury", "shield_specialization", "redoubt"} {
		if _, ok := TalentNodeIDs[field]; !ok {
			t.Errorf("%s is not in the generated talent table", field)
		}
	}
}

// Redoubt is 4% block a rank, 20% at rank 5 (the live text and Blizzard's
// 1 October 2026 notes, "Redoubt 4-20%").
func TestRedoubtBlockPerRank(t *testing.T) {
	if redoubtBlockChancePerRank != 4.0 {
		t.Errorf("redoubtBlockChancePerRank = %v, want 4", redoubtBlockChancePerRank)
	}
	if got := redoubtBlockChancePerRank * 5; got != 20 {
		t.Errorf("rank 5 Redoubt = %v%% block, want 20%%", got)
	}
}

// The rank texts as numbers: each is the client's own line, so a weekly
// number change is one line here and one in tank_talents.go.
func TestProtectionRankTextsAsNumbers(t *testing.T) {
	for _, tc := range []struct {
		name string
		got  float64
		want float64
	}{
		{"Redoubt proc chance", redoubtProcChance, 0.10},
		{"Reckoning, block, per rank", reckoningBlockChancePerRank * 5, 0.40},
		{"Reckoning, crit, per rank", reckoningCritChancePerRank * 5, 1.00},
		{"Shield Specialization rank 1 mana chance", shieldSpecializationManaChance[1], 0.33},
		{"Shield Specialization rank 2 mana chance", shieldSpecializationManaChance[2], 0.66},
		{"Shield Specialization rank 3 mana chance", shieldSpecializationManaChance[3], 1.00},
		{"Shield Specialization mana", shieldSpecializationManaFraction, 0.06},
		{"Shield Specialization absorb rank 3", shieldSpecializationAbsorbPerRank * 3, 0.30},
		{"One-Handed Weapon Specialization rank 1", oneHandedWeaponSpecializationPct[1], 0.03},
		{"One-Handed Weapon Specialization rank 2", oneHandedWeaponSpecializationPct[2], 0.07},
		{"One-Handed Weapon Specialization rank 3", oneHandedWeaponSpecializationPct[3], 0.10},
		{"Anticipation rank 5 defense", anticipationDefensePerRank * 5, 20},
		{"Toughness rank 5", toughnessArmorPctPerRank * 5, 0.10},
		{"Sacred Duty rank 2 stamina", sacredDutyStaminaPerRank * 2, 0.04},
		{"Iron Creed rank 5 threat", ironCreedThreatPerRank * 5, 0.25},
		{"Iron Creed rank 5 damage taken", ironCreedDamageTakenPerRank * 5, 0.10},
		{"Improved Righteous Fury rank 3", improvedRighteousFuryDamageTakenPerRank * 3, 0.06},
		{"Instrument of Law rank 2", instrumentOfLawThreatReductionPerRank * 2, 0.20},
		{"Righteous Fury Holy threat", righteousFuryHolyThreatBonus, 0.60},
	} {
		if d := tc.got - tc.want; d > 1e-9 || d < -1e-9 {
			t.Errorf("%s = %v, want %v", tc.name, tc.got, tc.want)
		}
	}
}
