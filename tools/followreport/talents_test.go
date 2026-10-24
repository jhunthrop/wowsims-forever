package main

import (
	"os"
	"path/filepath"
	"testing"
)

func loadTalentFixture(t *testing.T, name string) map[string]TalentsFile {
	t.Helper()
	m, err := LoadTalents(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("LoadTalents(%s): %v", name, err)
	}
	return m
}

func TestDiffTalents(t *testing.T) {
	prev := loadTalentFixture(t, "talents-prev")
	d := DiffTalents(prev, loadTalentFixture(t, "talents-cur"))
	if len(d.Added) != 1 || d.Added[0].NodeID != 13 {
		t.Errorf("added = %+v, want node 13", d.Added)
	}
	if len(d.Removed) != 1 || d.Removed[0].NodeID != 12 {
		t.Errorf("removed = %+v, want node 12", d.Removed)
	}
	if len(d.Changed) != 2 {
		t.Fatalf("changed = %+v, want nodes 10 (description) and 11 (max_rank)", d.Changed)
	}
	if d.Changed[0].NodeID != 10 || d.Changed[1].NodeID != 11 {
		t.Errorf("changed order = %+v", d.Changed)
	}
}

func TestDiffTalentsSpellIDChange(t *testing.T) {
	d := DiffTalents(loadTalentFixture(t, "talents-prev"), loadTalentFixture(t, "talents-spellid"))
	if len(d.Changed) != 1 || d.Changed[0].NodeID != 11 || len(d.Added)+len(d.Removed) != 0 {
		t.Fatalf("delta = %+v", d)
	}
}

func TestDiffTalentsUnchanged(t *testing.T) {
	d := DiffTalents(loadTalentFixture(t, "talents-prev"), loadTalentFixture(t, "talents-same"))
	if !d.Empty() {
		t.Fatalf("delta = %+v, want empty", d)
	}
}

func TestDiffTalentsClassAppearsAndVanishes(t *testing.T) {
	cur := loadTalentFixture(t, "talents-cur")
	if d := DiffTalents(nil, cur); len(d.Added) != 3 {
		t.Errorf("new class: %+v", d)
	}
	if d := DiffTalents(cur, nil); len(d.Removed) != 3 {
		t.Errorf("gone class: %+v", d)
	}
}

func TestLoadTalentsErrors(t *testing.T) {
	if _, err := LoadTalents(filepath.Join("testdata", "no-such-dir")); err == nil {
		t.Error("missing dir: want error")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "mage.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadTalents(dir); err == nil {
		t.Error("bad json: want error")
	}
}

func TestDiffRacials(t *testing.T) {
	changes, err := DiffRacials(filepath.Join("testdata", "races", "prev.json"), filepath.Join("testdata", "races", "cur.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 1 {
		t.Fatalf("changes = %+v, want one", changes)
	}
	c := changes[0]
	if c.Race != "human" || len(c.Added) != 1 || c.Added[0] != "A2" || len(c.Removed) != 1 || c.Removed[0] != "A" {
		t.Fatalf("change = %+v", c)
	}
}

func TestDiffRacialsIdentical(t *testing.T) {
	p := filepath.Join("testdata", "races", "prev.json")
	changes, err := DiffRacials(p, filepath.Join("testdata", "races", "same.json"))
	if err != nil || len(changes) != 0 {
		t.Fatalf("changes = %+v, err = %v", changes, err)
	}
}

func TestDiffRacialsErrors(t *testing.T) {
	good := filepath.Join("testdata", "races", "prev.json")
	if _, err := DiffRacials(good, filepath.Join("testdata", "races", "nope.json")); err == nil {
		t.Error("missing file: want error")
	}
	bad := filepath.Join(t.TempDir(), "r.json")
	if err := os.WriteFile(bad, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := DiffRacials(bad, good); err == nil {
		t.Error("bad json: want error")
	}
}
