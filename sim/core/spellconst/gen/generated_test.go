package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wowsims/classic/sim/core/spellconst"
)

// The real warrior.json lists Battle Shout rank 6 under two ids: 11551
// (a normal 100-mana cast, spell_level 52) and 27578 (a free-cast
// variant, also spell_level 52, cost 0). Both are in the checked-in
// fixture verbatim. resolveRankWinners must pick one deterministically
// — the higher spell_level, ties broken by the higher id — and report
// the loser rather than silently interleaving the two spells' data.
func TestResolveRankWinnersBattleShoutCollision(t *testing.T) {
	class, err := spellconst.Load(filepath.Join("..", "testdata", "warrior.json"))
	if err != nil {
		t.Fatal(err)
	}
	entries := class.Ranks("Battle Shout")
	if len(entries) != 8 {
		t.Fatalf("fixture has %d Battle Shout entries, want 8 (ranks 1-7, rank 6 twice)", len(entries))
	}

	winners, ranks, dropped := resolveRankWinners(entries)

	wantRanks := []int{1, 2, 3, 4, 5, 6, 7}
	if len(ranks) != len(wantRanks) {
		t.Fatalf("ranks = %v, want %v", ranks, wantRanks)
	}
	for i, r := range wantRanks {
		if ranks[i] != r {
			t.Fatalf("ranks = %v, want %v", ranks, wantRanks)
		}
	}

	winner, ok := winners[6]
	if !ok {
		t.Fatal("no winner resolved for rank 6")
	}
	if winner.ID != 27578 {
		t.Errorf("rank 6 winner ID = %d, want 27578 (spell_level ties at 52, so the higher id wins)", winner.ID)
	}

	losers := dropped[6]
	if len(losers) != 1 || losers[0].ID != 11551 {
		t.Errorf("rank 6 dropped = %v, want exactly [11551]", losers)
	}

	// Every other rank in the fixture has exactly one id, so nothing
	// should be dropped there.
	for _, r := range []int{1, 2, 3, 4, 5, 7} {
		if len(dropped[r]) != 0 {
			t.Errorf("rank %d unexpectedly dropped entries: %v", r, dropped[r])
		}
	}
}

// End to end: generate() must place the resolved winner at the array
// index matching its rank label (BattleShoutSpellId[6] == 27578, not
// BattleShoutSpellId[7] as the pre-fix generator produced), size the
// array to the true max rank (7, not 8), and name the dropped id in a
// comment. Run against a temp, empty output directory so Battle Shout
// is not skipped as "already hand-written" (it is, in sim/core, but that
// does not exist in this temp package).
func TestGenerateIndexesBattleShoutByRankLabel(t *testing.T) {
	class, err := spellconst.Load(filepath.Join("..", "testdata", "warrior.json"))
	if err != nil {
		t.Fatal(err)
	}
	outDir := t.TempDir()
	src, err := generate(class, outDir, "constants_auto_gen.go", "warrior", "testdata/warrior.json")
	if err != nil {
		t.Fatal(err)
	}
	text := string(src)

	if !strings.Contains(text, "const BattleShoutRanks = 7") {
		t.Error("want BattleShoutRanks = 7 (the true max rank), not 8 (the pre-fix count-based size)")
	}
	if !strings.Contains(text, "var BattleShoutSpellId = [BattleShoutRanks + 1]int32{0, 6673, 5242, 6192, 11549, 11550, 27578, 25289}") {
		t.Errorf("BattleShoutSpellId does not have 27578 (the rank-6 winner) at index 6 and 25289 at index 7:\n%s", extractLine(text, "BattleShoutSpellId"))
	}
	if !strings.Contains(text, "Battle Shout rank 6: kept id 27578 (spell_level 52); dropped 11551 (spell_level 52)") {
		t.Error("want a comment naming the dropped id 11551 beside Battle Shout's arrays")
	}
}

func extractLine(text, contains string) string {
	for _, line := range strings.Split(text, "\n") {
		if strings.Contains(line, contains) {
			return line
		}
	}
	return "(not found)"
}

// The distinct-name identifier-collision guard (two different spell
// names that collapse to the same Go identifier once punctuation is
// stripped) is not exercised by any real warrior/mage data in this
// branch's fixtures — the one real-world instance, mage's "Fireball" and
// "Fireball!", both separately hit the "already hand-written" guard
// first, since Fireball has its own ability file. This test forces the
// collision directly with two synthetic spell names neither of which
// exists in any package, to prove the "collides with" path itself fires
// and drops the second name rather than emitting a duplicate identifier.
func TestGenerateSkipsDistinctNamesCollidingOnIdentifier(t *testing.T) {
	class := spellconst.Class{
		Slug:  "warrior",
		Build: "test",
		Spells: []spellconst.Spell{
			{ID: 900001, Name: "Test Bolt", Rank: 1, SpellLevel: 10},
			{ID: 900002, Name: "Test Bolt!", Rank: 1, SpellLevel: 10},
		},
	}
	outDir := t.TempDir()
	src, err := generate(class, outDir, "constants_auto_gen.go", "warrior", "synthetic")
	if err != nil {
		t.Fatal(err)
	}
	text := string(src)

	if !strings.Contains(text, `skipped: "Test Bolt!" collides with "Test Bolt" as the Go identifier "TestBolt"`) {
		t.Errorf("want a collides-with comment for the second synthetic name; got:\n%s", text)
	}
	if strings.Count(text, "const TestBoltRanks") != 1 {
		t.Error("want exactly one TestBoltRanks declaration (the first name to claim the identifier), not zero or two")
	}
}

// generate must produce byte-identical output on two independent runs
// against the same real client file, and that output must match the
// committed constants_auto_gen.go — proving both that the pipeline's Go
// map iteration order (spellconst.Load reads a JSON object) cannot leak
// into the result, and that the committed file is not stale. Mirrors
// sim/core/proto/generated_test.go's pattern: skipped when the input
// this needs is not present, since the data lane's spellconst output is
// not yet on this repo's main.
func TestConstantsAutoGenRegeneratesByteIdenticallyAndDeterministically(t *testing.T) {
	dataDir := "/Users/jh/code/forever/.worktrees/sim-data/data/builds/1.60.1.69893/spellconst"

	cases := []struct {
		class  string
		outDir string
	}{
		{"warrior", filepath.Join("..", "..", "..", "warrior")},
		{"mage", filepath.Join("..", "..", "..", "mage")},
	}
	for _, tc := range cases {
		t.Run(tc.class, func(t *testing.T) {
			in := filepath.Join(dataDir, tc.class+".json")
			if _, err := os.Stat(in); err != nil {
				t.Skipf("real spellconst input not present at %s (the data lane's worktree is not checked out here)", in)
			}

			run := func() []byte {
				class, err := spellconst.Load(in)
				if err != nil {
					t.Fatal(err)
				}
				src, err := generate(class, tc.outDir, "constants_auto_gen.go", tc.class, in)
				if err != nil {
					t.Fatal(err)
				}
				return src
			}

			first := run()
			second := run()
			if string(first) != string(second) {
				t.Fatalf("%s: two runs against the identical input produced different output; regeneration is not deterministic", tc.class)
			}

			committed, err := os.ReadFile(filepath.Join(tc.outDir, "constants_auto_gen.go"))
			if err != nil {
				t.Fatal(err)
			}
			if string(first) != string(committed) {
				t.Errorf("sim/%s/constants_auto_gen.go is stale; regenerate with make spellconst and commit the result", tc.class)
			}
		})
	}
}
