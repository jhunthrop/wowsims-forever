package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const jsonExt = ".json"

// TalentRank is one rank of a talent node.
type TalentRank struct {
	SpellID     int    `json:"spell_id"`
	Description string `json:"description"`
}

// Talent is the part of a talent node a follow compares.
type Talent struct {
	ID      int          `json:"id"`
	Name    string       `json:"name"`
	MaxRank int          `json:"max_rank"`
	SpellID int          `json:"spell_id"`
	Ranks   []TalentRank `json:"ranks"`
}

// TalentTree is one tree of a class.
type TalentTree struct {
	Name    string   `json:"name"`
	Talents []Talent `json:"talents"`
}

// TalentsFile is the site's talents/<class>.json.
type TalentsFile struct {
	Build     string       `json:"build"`
	ClassSlug string       `json:"class_slug"`
	Trees     []TalentTree `json:"trees"`
}

// TalentChange is one added, removed or changed talent node.
type TalentChange struct {
	Class  string
	NodeID int
	Name   string
	Detail string
}

// TalentDelta lists every talent node that moved between two builds.
type TalentDelta struct{ Added, Removed, Changed []TalentChange }

// Empty reports whether no talent node moved.
func (d TalentDelta) Empty() bool {
	return len(d.Added)+len(d.Removed)+len(d.Changed) == 0
}

// RacialChange is the set difference of one race's Forever change texts.
type RacialChange struct {
	Race           string
	Added, Removed []string
}

// LoadTalents reads every <class>.json in dir, keyed by class slug (file name).
func LoadTalents(dir string) (map[string]TalentsFile, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*"+jsonExt))
	if err != nil {
		return nil, fmt.Errorf("followreport: %s: %w", dir, err)
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("followreport: %s: no talent tables found", dir)
	}
	out := map[string]TalentsFile{}
	for _, path := range paths {
		var f TalentsFile
		if err := readJSON(path, &f); err != nil {
			return nil, err
		}
		out[strings.TrimSuffix(filepath.Base(path), jsonExt)] = f
	}
	return out, nil
}

func readJSON(path string, into any) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("followreport: %s: %w", path, err)
	}
	if err := json.Unmarshal(body, into); err != nil {
		return fmt.Errorf("followreport: %s: %w", path, err)
	}
	return nil
}

func nodesByID(f TalentsFile) map[int]Talent {
	nodes := map[int]Talent{}
	for _, tree := range f.Trees {
		for _, t := range tree.Talents {
			nodes[t.ID] = t
		}
	}
	return nodes
}

// describeChange names what differs between two versions of a node, or "".
func describeChange(prev, cur Talent) string {
	var parts []string
	if prev.MaxRank != cur.MaxRank {
		parts = append(parts, fmt.Sprintf("max_rank %d -> %d", prev.MaxRank, cur.MaxRank))
	}
	if prev.SpellID != cur.SpellID {
		parts = append(parts, fmt.Sprintf("spell_id %d -> %d", prev.SpellID, cur.SpellID))
	}
	if !sameRanks(prev.Ranks, cur.Ranks) {
		parts = append(parts, "rank description changed")
	}
	return strings.Join(parts, "; ")
}

func sameRanks(a, b []TalentRank) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Description != b[i].Description {
			return false
		}
	}
	return true
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedIDs(m map[int]Talent) []int {
	ids := make([]int, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}

// DiffTalents compares two builds' talent tables, node by node id.
func DiffTalents(prev, cur map[string]TalentsFile) TalentDelta {
	var d TalentDelta
	classes := map[string]TalentsFile{}
	for _, m := range []map[string]TalentsFile{prev, cur} {
		for k, v := range m {
			classes[k] = v
		}
	}
	for _, class := range sortedKeys(classes) {
		p, c := nodesByID(prev[class]), nodesByID(cur[class])
		for _, id := range sortedIDs(c) {
			old, existed := p[id]
			switch detail := describeChange(old, c[id]); {
			case !existed:
				d.Added = append(d.Added, TalentChange{Class: class, NodeID: id, Name: c[id].Name})
			case detail != "":
				d.Changed = append(d.Changed, TalentChange{Class: class, NodeID: id, Name: c[id].Name, Detail: detail})
			}
		}
		for _, id := range sortedIDs(p) {
			if _, kept := c[id]; !kept {
				d.Removed = append(d.Removed, TalentChange{Class: class, NodeID: id, Name: p[id].Name})
			}
		}
	}
	return d
}

type race struct {
	Slug    string `json:"slug"`
	Changes []struct {
		Text string `json:"text"`
	} `json:"forever_changes"`
}

func loadRacialTexts(path string) (map[string]map[string]bool, error) {
	var races []race
	if err := readJSON(path, &races); err != nil {
		return nil, err
	}
	out := map[string]map[string]bool{}
	for _, r := range races {
		set := map[string]bool{}
		for _, c := range r.Changes {
			set[c.Text] = true
		}
		out[r.Slug] = set
	}
	return out, nil
}

func setDifference(a, b map[string]bool) []string {
	var out []string
	for text := range a {
		if !b[text] {
			out = append(out, text)
		}
	}
	sort.Strings(out)
	return out
}

// DiffRacials compares each race's forever_changes texts as a set.
func DiffRacials(prev, cur string) ([]RacialChange, error) {
	p, err := loadRacialTexts(prev)
	if err != nil {
		return nil, err
	}
	c, err := loadRacialTexts(cur)
	if err != nil {
		return nil, err
	}
	slugs := map[string]bool{}
	for s := range p {
		slugs[s] = true
	}
	for s := range c {
		slugs[s] = true
	}
	var out []RacialChange
	for _, slug := range sortedKeys(slugs) {
		added, removed := setDifference(c[slug], p[slug]), setDifference(p[slug], c[slug])
		if len(added)+len(removed) > 0 {
			out = append(out, RacialChange{Race: slug, Added: added, Removed: removed})
		}
	}
	return out, nil
}
