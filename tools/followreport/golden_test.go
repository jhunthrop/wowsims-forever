package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseGoldenSections(t *testing.T) {
	g := mustParse(t, "mage-a", "mage")
	if g.Class != "Mage" {
		t.Fatalf("class = %q", g.Class)
	}
	fb := RowKey{Section: sectionSpells, Spec: "Mage", Level: 10, SpellID: 133, Rank: 1}
	if got := g.Rows[fb]; got != (Row{Verdict: "match", Damage: "declared, matches"}) {
		t.Fatalf("Fireball row = %+v", got)
	}
	tg := RowKey{Section: sectionTalentGated, Spec: "Mage (Arcane Blast talent)", Level: 20, SpellID: 400574, Rank: 1}
	if got := g.Rows[tg].Verdict; got != "client-scripted" {
		t.Fatalf("talent-gated verdict = %q", got)
	}
	sleep := g.Trainables["Sleep (700)"]
	if !sleep.HasDispositionColumn || sleep.Disposition != "utility: crowd control" {
		t.Fatalf("Sleep = %+v", sleep)
	}
}

func TestParseGoldenWithoutDispositionColumn(t *testing.T) {
	g := mustParse(t, "rogue-a", "rogue")
	pick := g.Trainables["Pick Lock (1804)"]
	if pick.HasDispositionColumn || pick.Disposition != "" {
		t.Fatalf("Pick Lock = %+v", pick)
	}
}

func TestParseGoldenErrors(t *testing.T) {
	if _, err := ParseGolden(filepath.Join("testdata", "missing.golden.md")); err == nil {
		t.Fatal("missing file: want error")
	}
	dir := t.TempDir()
	cases := map[string]string{
		"no-h1":      "## Spells\n",
		"bad-level":  "# X: y\n\n## Spells\n\n| Spec | Level | Spell | Rank | SpellID | Verdict | Damage |\n|---|---|---|---|---|---|---|\n| a | zz | b | 1 | 2 | match | n/a |\n",
		"no-column":  "# X: y\n\n## Spells\n\n| Spec | Level |\n|---|---|\n| a | 1 |\n",
		"short-row":  "# X: y\n\n## Spells\n\n| Spec | Level | Spell | Rank | SpellID | Verdict | Damage |\n|---|---|---|---|---|---|---|\n| a | 1 |\n",
		"tr-no-name": "# X: y\n\n## Trainable abilities the engine does not register\n\n| Cost |\n|---|\n| 1 |\n",
	}
	for name, body := range cases {
		path := filepath.Join(dir, name+".golden.md")
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := ParseGolden(path); err == nil {
			t.Errorf("%s: want error", name)
		} else if !strings.Contains(err.Error(), "followreport") {
			t.Errorf("%s: error lacks context: %v", name, err)
		}
	}
}

func TestParseGoldenSubTablesMerge(t *testing.T) {
	g := mustParse(t, "mage-subtable-classified", "mage")
	if got := g.Trainables["Mark of the Wild (1126)"].Disposition; got != "modeled as core buff" {
		t.Fatalf("sub-table disposition = %q", got)
	}
	if _, ok := g.Trainables["Sleep (700)"]; !ok {
		t.Fatal("first table row lost")
	}
}

func TestParseGoldenFailFast(t *testing.T) {
	good, err := os.ReadFile(filepath.Join("testdata", "mage-a", "mage.golden.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(good)
	fireball := "| Mage | 10 | Fireball | 1 | 133 |"
	line := text[strings.Index(text, fireball):]
	line = line[:strings.Index(line, "\n")+1]
	cases := map[string]struct{ body, want string }{
		"no spells":         {strings.Replace(text, headingSpells, "## Other", 1), headingSpells},
		"no talent-gated":   {strings.Replace(text, headingTalentGated, "## Other", 1), headingTalentGated},
		"duplicate row":     {strings.Replace(text, line, line+line, 1), "duplicate row"},
		"duplicate ability": {text + "\n### Again\n\n| Ability |\n|---|\n| Sleep (700) |\n", "duplicate ability"},
	}
	for name, tc := range cases {
		_, err := parseGoldenText(tc.body)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: err = %v, want mention of %q", name, err, tc.want)
		}
	}
}

func TestParseGoldenKeepsDiff(t *testing.T) {
	g := mustParse(t, "mage-one-mismatch", "mage")
	key := RowKey{Section: sectionSpells, Spec: "Mage", Level: 10, SpellID: 116, Rank: 1}
	if got := g.Rows[key].Diff; got != "cost: 25 vs 30" {
		t.Fatalf("diff = %q", got)
	}
}

func TestParseRealGoldens(t *testing.T) {
	paths, _ := filepath.Glob("../../sim/core/testdata/conformance/*.golden.md")
	if len(paths) == 0 {
		t.Skip("no real goldens")
	}
	for _, p := range paths {
		if strings.HasSuffix(p, setsGolden+goldenExt) {
			continue
		}
		g, err := ParseGolden(p)
		if err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		if len(g.Rows) == 0 {
			t.Errorf("%s: no rows parsed", p)
		}
	}
}
