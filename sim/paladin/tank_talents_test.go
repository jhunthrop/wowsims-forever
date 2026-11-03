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
	if spent[1] < 31 {
		t.Errorf("the build spends %d points in Protection, want at least 31 to reach Holy Shield", spent[1])
	}
}

// The points past the Protection core go where a tank gets something for
// them: Deflection (the only parry talent a paladin has, in the
// Retribution tree), Sacred Duty, Iron Creed and the Shieldsworn capstone.
// The reference build once spent them on the Holy tree's healing talents,
// and the same boss killed it many times more often than it kills this
// build.
func TestForeverProtectionTalentsSpendThePointsPastHolyShieldOnMitigation(t *testing.T) {
	trees := loadPaladinTrees(t)
	segments := strings.Split(ForeverProtectionTalents, "-")
	rank := func(treeIndex int, field string) int {
		for j, talent := range trees[treeIndex].Talents {
			if talent.FieldName == field {
				return int(segments[treeIndex][j] - '0')
			}
		}
		t.Fatalf("%s has no %s", trees[treeIndex].Name, field)
		return 0
	}
	for _, want := range []struct {
		tree  int
		field string
		rank  int
	}{{2, "deflection", 5}, {1, "sacredDuty", 2}, {1, "ironCreed", 3}, {1, "shieldsworn", 1}} {
		if got := rank(want.tree, want.field); got != want.rank {
			t.Errorf("%s is at rank %d, want %d", want.field, got, want.rank)
		}
	}
}

// Every Protection talent the reference build names exists in the tree
// the engine parses: a renamed field would silently drop out of the build.
func TestForeverProtectionTalentsTakeTheTalentsThisSpecModels(t *testing.T) {
	for _, field := range []string{"holy_shield", "templars_bulwark", "swift_judgement", "shieldsworn", "shield_specialization", "redoubt", "deflection", "sacred_duty", "iron_creed"} {
		if _, ok := TalentNodeIDs[field]; !ok {
			t.Errorf("%s is not in the generated talent table", field)
		}
	}
}

// Redoubt is 6% block a rank, 30% at rank 5 (client 1.60.1.70707).
func TestRedoubtBlockPerRank(t *testing.T) {
	if redoubtBlockChancePerRank != 6.0 {
		t.Errorf("redoubtBlockChancePerRank = %v, want 6", redoubtBlockChancePerRank)
	}
	if got := redoubtBlockChancePerRank * 5; got != 30 {
		t.Errorf("rank 5 Redoubt = %v%% block, want 30%%", got)
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
		{"Shield Specialization rank 1 mana chance", shieldSpecializationManaChance[1], 0.50},
		{"Shield Specialization rank 2 mana chance", shieldSpecializationManaChance[2], 1.00},
		{"Shield Specialization mana", shieldSpecializationManaFraction, 0.04},
		{"Shield Specialization absorb rank 2", shieldSpecializationAbsorbPerRank * 2, 0.30},
		{"Anticipation rank 5 defense", anticipationDefensePerRank * 5, 20},
		{"Toughness rank 5", toughnessArmorPctPerRank * 5, 0.10},
		{"Sacred Duty rank 2 Forbearance", (sacredDutyForbearanceReductionPerRank * 2).Seconds(), 10},
		{"Iron Creed rank 3 threat", ironCreedThreatPerRank * 3, 0.30},
		{"Iron Creed rank 3 damage taken", ironCreedDamageTakenPerRank * 3, 0.06},
		{"Improved Righteous Fury rank 3", improvedRighteousFuryDamageTakenPerRank * 3, 0.09},
		{"Righteous Fury Holy threat", righteousFuryHolyThreatBonus, 0.90},
		{"Righteous Fury all threat", righteousFuryAllThreatBonus, 0.33},
	} {
		if d := tc.got - tc.want; d > 1e-9 || d < -1e-9 {
			t.Errorf("%s = %v, want %v", tc.name, tc.got, tc.want)
		}
	}
}

// Holy Shield's ranks cost 100, 150 and 195 mana: the 70707 hotfix push of
// 31 October (SpellPower 185892, 186383, 184913; shipped 150, 195, 240) and
// Wowhead Forever's live tooltips agree. Client 1.60.1.70753 adds rank 4
// (1239570) at level 60 for 240.
func TestHolyShieldRanksCostTheHotfixedMana(t *testing.T) {
	want := map[int32]float64{20925: 100, 20927: 150, 20928: 195, 1239570: 240}
	for _, rank := range HolyShieldValues {
		if rank.manaCost != want[rank.spellID] {
			t.Errorf("Holy Shield %d costs %v mana, want %v", rank.spellID, rank.manaCost, want[rank.spellID])
		}
	}
}
