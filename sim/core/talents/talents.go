// Package talents reads the client's mined talent trees.
//
// The input is the site pipeline's per-class output,
// data/builds/<build>/talents/<class-slug>.json, generated from the
// 1.60 client's trait tables. It is the single source of every talent's
// name, grid position, rank count, prerequisite and spell ids, and
// nothing in this repository may type any of those by hand: the proto
// message's field order, TalentTreeSizes, and the order the planner
// writes a talent string in must agree exactly, and they only can if one
// file produces all three.
package talents

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode"
)

// ForeverMilestones are the point totals at which a tree opens its
// one-point talents: vanilla's 11, 21 and 31 plus Forever's new 16.
// Declared once, here, because a tree's shape is the game's and not a
// class's, and two classes disagreeing about it would be a bug nobody
// would notice.
var ForeverMilestones = [4]int{11, 16, 21, 31}

// Talent is one node of one tree.
type Talent struct {
	NodeID  int32
	Name    string
	Icon    string
	MaxRank int
	Tier    int
	Column  int
	// SpellID is the talent's own spell, which the client gives once per
	// talent rather than once per rank.
	SpellID int32
	// RankSpellIDs is one id per rank, in rank order. len == MaxRank.
	RankSpellIDs []int32
	// PrereqNodeID is 0 when the talent has no prerequisite.
	PrereqNodeID int32
	PrereqRank   int
}

// Tree is one of a class's three tabs.
type Tree struct {
	ID       int32
	Name     string
	Position int
	// Talents are ordered tier then column, which is the order the
	// talent string is written in.
	Talents []Talent
}

// Class is one class's three trees.
type Class struct {
	Slug    string
	Build   string
	ClassID int32
	Trees   [3]Tree
}

// the on-disk shape, kept private so the public types can differ from it.
type fileClass struct {
	Build     string     `json:"build"`
	ClassID   int32      `json:"class_id"`
	ClassSlug string     `json:"class_slug"`
	Trees     []fileTree `json:"trees"`
}

type fileTree struct {
	ID       int32        `json:"id"`
	Name     string       `json:"name"`
	Position int          `json:"position"`
	Talents  []fileTalent `json:"talents"`
}

type fileTalent struct {
	ID             int32      `json:"id"`
	Name           string     `json:"name"`
	Icon           string     `json:"icon"`
	MaxRank        int        `json:"max_rank"`
	Tier           int        `json:"tier"`
	Column         int        `json:"column"`
	PrereqTalentID *int32     `json:"prereq_talent_id"`
	PrereqRank     *int       `json:"prereq_rank"`
	SpellID        int32      `json:"spell_id"`
	Ranks          []fileRank `json:"ranks"`
}

type fileRank struct {
	SpellID     int32  `json:"spell_id"`
	Description string `json:"description"`
}

// Load reads one class's trees.
func Load(path string) (Class, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Class{}, fmt.Errorf("talents: %w", err)
	}
	var fc fileClass
	if err := json.Unmarshal(b, &fc); err != nil {
		return Class{}, fmt.Errorf("talents: %s: %w", path, err)
	}
	if len(fc.Trees) != 3 {
		return Class{}, fmt.Errorf("talents: %s has %d trees, want 3", path, len(fc.Trees))
	}
	out := Class{Slug: fc.ClassSlug, Build: fc.Build, ClassID: fc.ClassID}
	trees := append([]fileTree(nil), fc.Trees...)
	sort.SliceStable(trees, func(i, j int) bool { return trees[i].Position < trees[j].Position })
	for i, ft := range trees {
		if ft.Position != i {
			return Class{}, fmt.Errorf("talents: %s tree %q has position %d at index %d", path, ft.Name, ft.Position, i)
		}
		t := Tree{ID: ft.ID, Name: ft.Name, Position: ft.Position}
		tal := append([]fileTalent(nil), ft.Talents...)
		sort.SliceStable(tal, func(a, b int) bool {
			if tal[a].Tier != tal[b].Tier {
				return tal[a].Tier < tal[b].Tier
			}
			return tal[a].Column < tal[b].Column
		})
		for _, ta := range tal {
			ids := make([]int32, 0, len(ta.Ranks))
			for _, r := range ta.Ranks {
				ids = append(ids, r.SpellID)
			}
			n := Talent{
				NodeID: ta.ID, Name: ta.Name, Icon: ta.Icon,
				MaxRank: ta.MaxRank, Tier: ta.Tier, Column: ta.Column,
				SpellID: ta.SpellID, RankSpellIDs: ids,
			}
			if ta.PrereqTalentID != nil {
				n.PrereqNodeID = *ta.PrereqTalentID
			}
			if ta.PrereqRank != nil {
				n.PrereqRank = *ta.PrereqRank
			}
			t.Talents = append(t.Talents, n)
		}
		out.Trees[i] = t
	}
	return out, nil
}

// TreeSizes is what core.FillTalentsProto slices a talent string with.
func (c Class) TreeSizes() [3]int {
	return [3]int{len(c.Trees[0].Talents), len(c.Trees[1].Talents), len(c.Trees[2].Talents)}
}

// FieldName is the talent's proto field name: the talent name in lower
// snake case, matching the convention the nine existing messages use.
// "Improved Heroic Strike" becomes improved_heroic_strike, "Eureka!"
// becomes eureka, "Hack and Slash" becomes hack_and_slash.
//
// An apostrophe is dropped rather than treated as a separator, which is
// the convention the hand-written messages already follow: "Nature's
// Grace" is natures_grace and "Winter's Chill" is winters_chill, not
// nature_s_grace and winter_s_chill. Every other punctuation mark is a
// separator and a run of them collapses, so "Improved Power Word:
// Shield" is improved_power_word_shield and "One-Handed Weapon
// Specialization" is one_handed_weapon_specialization.
func (c Class) FieldName(t Talent) string {
	var b strings.Builder
	prevUnderscore := true // suppress a leading underscore
	for _, r := range t.Name {
		switch {
		case r == '\'' || r == '\u2019':
			// dropped, and it does not break the surrounding word.
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(unicode.ToLower(r))
			prevUnderscore = false
		default:
			if !prevUnderscore {
				b.WriteByte('_')
				prevUnderscore = true
			}
		}
	}
	s := strings.TrimSuffix(b.String(), "_")
	// A proto field name may not start with a digit, and no Forever
	// talent does today; guard anyway so a future one fails loudly here
	// rather than in protoc.
	if s != "" && s[0] >= '0' && s[0] <= '9' {
		s = "t_" + s
	}
	return s
}

// ByNodeID finds a talent by its client node id.
func (c Class) ByNodeID(id int32) (Talent, bool) {
	for _, tr := range c.Trees {
		for _, t := range tr.Talents {
			if t.NodeID == id {
				return t, true
			}
		}
	}
	return Talent{}, false
}

// BySpellID finds a talent by its own spell id or by any rank's.
func (c Class) BySpellID(id int32) (Talent, bool) {
	for _, tr := range c.Trees {
		for _, t := range tr.Talents {
			if t.SpellID == id {
				return t, true
			}
			for _, r := range t.RankSpellIDs {
				if r == id {
					return t, true
				}
			}
		}
	}
	return Talent{}, false
}
