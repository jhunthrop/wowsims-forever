package talents

import "testing"

func warrior(t *testing.T) Class {
	t.Helper()
	c, err := Load("../../../tools/talentgen/testdata/warrior.json")
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// The client's own tree order is the game's: Arms, Fury, Protection. The
// planner renders in this order and the talent string is written in it,
// so a reader that sorts alphabetically silently reads Fury's points as
// Arms's.
func TestTreesAreInTheClientsOrder(t *testing.T) {
	c := warrior(t)
	want := []string{"Arms", "Fury", "Protection"}
	for i, n := range want {
		if c.Trees[i].Name != n {
			t.Errorf("tree %d is %q, want %q", i, c.Trees[i].Name, n)
		}
		if c.Trees[i].Position != i {
			t.Errorf("tree %q has position %d, want %d", c.Trees[i].Name, c.Trees[i].Position, i)
		}
	}
	if c.Trees[0].ID != 161 || c.Trees[1].ID != 164 || c.Trees[2].ID != 163 {
		t.Errorf("tree ids = %d/%d/%d, want 161/164/163", c.Trees[0].ID, c.Trees[1].ID, c.Trees[2].ID)
	}
}

// TreeSizes is what core.FillTalentsProto slices the talent string with.
// A wrong size does not error, it reads the wrong talent.
func TestTreeSizesMatchTheClient(t *testing.T) {
	c := warrior(t)
	want := [3]int{17, 18, 18}
	if got := c.TreeSizes(); got != want {
		t.Errorf("TreeSizes() = %v, want %v", got, want)
	}
	if c.Build != "1.60.1.69893" {
		t.Errorf("Build = %q; the reader must carry the client build through", c.Build)
	}
}

// Seven rows and four columns, which is the grid the design describes and
// the planner draws.
func TestTheGridIsSevenByFour(t *testing.T) {
	c := warrior(t)
	for _, tr := range c.Trees {
		for _, ta := range tr.Talents {
			if ta.Tier < 0 || ta.Tier > 6 {
				t.Errorf("%s/%s is on tier %d, outside 0-6", tr.Name, ta.Name, ta.Tier)
			}
			if ta.Column < 0 || ta.Column > 3 {
				t.Errorf("%s/%s is in column %d, outside 0-3", tr.Name, ta.Name, ta.Column)
			}
		}
	}
}

// Within a tree the talents must be ordered tier-then-column, because
// that is the order the talent string is written in and the order the
// generated proto's fields take.
func TestTalentsAreOrderedTierThenColumn(t *testing.T) {
	c := warrior(t)
	for _, tr := range c.Trees {
		for i := 1; i < len(tr.Talents); i++ {
			a, b := tr.Talents[i-1], tr.Talents[i]
			if a.Tier > b.Tier || (a.Tier == b.Tier && a.Column >= b.Column) {
				t.Errorf("%s: %q (t%d c%d) is before %q (t%d c%d)", tr.Name, a.Name, a.Tier, a.Column, b.Name, b.Tier, b.Column)
			}
		}
	}
}

// Every talent carries its client node id, its spell id, and one spell id
// per rank. Tasks 11 and 12 attach spell mods by rank, and a missing rank
// id would send them back to typing numbers.
func TestEveryTalentCarriesItsIDs(t *testing.T) {
	c := warrior(t)
	for _, tr := range c.Trees {
		for _, ta := range tr.Talents {
			if ta.NodeID == 0 {
				t.Errorf("%s/%s has no node id", tr.Name, ta.Name)
			}
			if ta.SpellID == 0 {
				t.Errorf("%s/%s has no spell id", tr.Name, ta.Name)
			}
			if ta.MaxRank < 1 {
				t.Errorf("%s/%s has max rank %d", tr.Name, ta.Name, ta.MaxRank)
			}
			if len(ta.RankSpellIDs) != ta.MaxRank {
				t.Errorf("%s/%s has %d rank spell ids for %d ranks", tr.Name, ta.Name, len(ta.RankSpellIDs), ta.MaxRank)
			}
		}
	}
}

// A prerequisite must point at a talent in the same tree, or the planner
// and the engine disagree about what gates what.
func TestPrerequisitesPointIntoTheSameTree(t *testing.T) {
	c := warrior(t)
	for _, tr := range c.Trees {
		in := map[int32]bool{}
		for _, ta := range tr.Talents {
			in[ta.NodeID] = true
		}
		for _, ta := range tr.Talents {
			if ta.PrereqNodeID == 0 {
				continue
			}
			if !in[ta.PrereqNodeID] {
				t.Errorf("%s/%s requires node %d, which is not in this tree", tr.Name, ta.Name, ta.PrereqNodeID)
			}
			if ta.PrereqRank < 1 {
				t.Errorf("%s/%s has a prerequisite with rank %d", tr.Name, ta.Name, ta.PrereqRank)
			}
		}
	}
}

// The proto field name is derived, not typed, and it must be a legal
// proto identifier and unique across the whole class.
func TestFieldNamesAreLegalAndUnique(t *testing.T) {
	c := warrior(t)
	seen := map[string]string{}
	for _, tr := range c.Trees {
		for _, ta := range tr.Talents {
			n := c.FieldName(ta)
			if n == "" {
				t.Errorf("%s/%s produced an empty field name", tr.Name, ta.Name)
				continue
			}
			if n[0] < 'a' || n[0] > 'z' {
				t.Errorf("%q does not start with a lowercase letter", n)
			}
			for _, r := range n {
				if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_') {
					t.Errorf("%q contains %q, which is not legal in a proto field name", n, r)
					break
				}
			}
			if prev, dup := seen[n]; dup {
				t.Errorf("%q is produced by both %q and %q", n, prev, ta.Name)
			}
			seen[n] = ta.Name
		}
	}
}

// Forever's milestones are declared once here rather than per class, so
// two classes cannot disagree about the shape of a tree.
func TestForeverMilestones(t *testing.T) {
	if ForeverMilestones != [4]int{11, 16, 21, 31} {
		t.Errorf("ForeverMilestones = %v, want [11 16 21 31]", ForeverMilestones)
	}
}

// The field name follows the convention the nine hand-written messages
// already used, which the brief names as the rule: an apostrophe is
// dropped, every other punctuation mark separates, and a run of
// separators collapses to one underscore. Tasks 11 and 12 type these
// names in Go, so natures_grace rather than nature_s_grace is not a
// cosmetic difference: it is whether the fork's talent fields keep the
// names the rest of the tree already uses.
func TestFieldNameFollowsTheExistingConvention(t *testing.T) {
	c := warrior(t)
	for _, tc := range []struct{ in, want string }{
		{"Improved Heroic Strike", "improved_heroic_strike"},
		{"Nature's Grace", "natures_grace"},
		{"Winter's Chill", "winters_chill"},
		{"Improved Power Word: Shield", "improved_power_word_shield"},
		{"One-Handed Weapon Specialization", "one_handed_weapon_specialization"},
		{"Hack and Slash", "hack_and_slash"},
		{"Eureka!", "eureka"},
	} {
		if got := c.FieldName(Talent{Name: tc.in}); got != tc.want {
			t.Errorf("FieldName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// ByNodeID is how a planner build or a combat log - both of which speak
// the client's node ids - is matched to a talent. An id from another
// class's tree must not resolve.
func TestByNodeID(t *testing.T) {
	c := warrior(t)
	got, ok := c.ByNodeID(105958)
	if !ok {
		t.Fatal("node 105958 does not resolve; it is Arms tier 0 column 0")
	}
	if got.Name != "Improved Heroic Strike" {
		t.Errorf("node 105958 is %q, want %q", got.Name, "Improved Heroic Strike")
	}
	// Every talent of every tree resolves to itself, so no tree is
	// skipped by the walk.
	for _, tr := range c.Trees {
		for _, want := range tr.Talents {
			got, ok := c.ByNodeID(want.NodeID)
			if !ok || got.NodeID != want.NodeID {
				t.Errorf("%s/%s (node %d) does not resolve by its own id", tr.Name, want.Name, want.NodeID)
			}
		}
	}
	if _, ok := c.ByNodeID(1); ok {
		t.Error("node 1 resolves, and it is in no tree")
	}
	if _, ok := c.ByNodeID(0); ok {
		t.Error("node 0 resolves; 0 is the absent-prerequisite sentinel and must never match")
	}
}

// BySpellID matches on the talent's own spell id or on any rank's, so a
// log line naming a rank's spell finds the talent that granted it.
func TestBySpellID(t *testing.T) {
	c := warrior(t)
	got, ok := c.BySpellID(12282)
	if !ok {
		t.Fatal("spell 12282 does not resolve; it is Improved Heroic Strike")
	}
	if got.Name != "Improved Heroic Strike" {
		t.Errorf("spell 12282 is %q, want %q", got.Name, "Improved Heroic Strike")
	}
	// The rank branch: every rank id of every talent resolves.
	for _, tr := range c.Trees {
		for _, want := range tr.Talents {
			for i, id := range want.RankSpellIDs {
				got, ok := c.BySpellID(id)
				if !ok {
					t.Errorf("%s/%s rank %d spell %d does not resolve", tr.Name, want.Name, i+1, id)
					continue
				}
				if got.NodeID != want.NodeID {
					// Two talents sharing a rank spell id would be a data
					// problem worth seeing, not a reader bug.
					t.Errorf("%s/%s rank %d spell %d resolves to %q", tr.Name, want.Name, i+1, id, got.Name)
				}
			}
		}
	}
	if _, ok := c.BySpellID(1); ok {
		t.Error("spell 1 resolves, and no talent has it")
	}
	if _, ok := c.BySpellID(0); ok {
		t.Error("spell 0 resolves; no talent has spell 0")
	}
}
