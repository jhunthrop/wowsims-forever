package spellconst

import (
	"math"
	"testing"
)

func TestLoadReadsAClassFile(t *testing.T) {
	c, err := Load("testdata/warrior.json")
	if err != nil {
		t.Fatal(err)
	}
	if c.Slug != "warrior" {
		t.Errorf("Slug = %q, want %q", c.Slug, "warrior")
	}
	if c.Build == "" {
		t.Error("Build is empty; a constants file must record which client it came from")
	}
	if len(c.Spells) == 0 {
		t.Fatal("no spells loaded")
	}
}

func TestByID(t *testing.T) {
	c, err := Load("testdata/warrior.json")
	if err != nil {
		t.Fatal(err)
	}
	got, ok := c.ByID(23894)
	if !ok {
		t.Fatal("spell 23894 (Bloodthirst) not found")
	}
	if got.Name != "Bloodthirst" {
		t.Errorf("name = %q, want Bloodthirst", got.Name)
	}
	if _, ok := c.ByID(1); ok {
		t.Error("spell 1 was found in a warrior file")
	}
}

func TestRanksAreOrdered(t *testing.T) {
	c, err := Load("testdata/warrior.json")
	if err != nil {
		t.Fatal(err)
	}
	ranks := c.Ranks("Bloodthirst")
	if len(ranks) < 2 {
		t.Fatalf("Bloodthirst has %d ranks in the fixture, want at least 2", len(ranks))
	}
	for i := 1; i < len(ranks); i++ {
		if ranks[i].Rank <= ranks[i-1].Rank {
			t.Errorf("ranks are not ascending: %d then %d", ranks[i-1].Rank, ranks[i].Rank)
		}
	}
	if c.Ranks("Nonexistent") != nil {
		t.Error("an unknown spell name returned ranks")
	}
}

// The data lane emits the DB2 coefficient columns verbatim, zeros
// included, because EffectBonusCoefficient is routinely 0 or wrong for
// Classic-lineage spells. A zero therefore means "absent", and the
// vanilla convention fills it in — never a literal zero coefficient,
// which would silently remove all spell-power scaling from a spell.
func TestZeroCoefficientFallsBackToTheConvention(t *testing.T) {
	c, err := Load("testdata/warrior.json")
	if err != nil {
		t.Fatal(err)
	}
	s, ok := c.ByID(11605) // Slam rank 4 in the fixture, coefficient 0
	if !ok {
		t.Fatal("spell 11605 not found")
	}
	if s.Coefficient == 0 {
		t.Error("a zero DB2 coefficient was kept as zero; it must fall back to the convention")
	}
	if s.CoefficientSource != "convention" {
		t.Errorf("CoefficientSource = %q, want %q", s.CoefficientSource, "convention")
	}
}

func TestNonZeroCoefficientIsKept(t *testing.T) {
	c, err := Load("testdata/warrior.json")
	if err != nil {
		t.Fatal(err)
	}
	s, ok := c.ByID(23881) // Bloodthirst rank 1 in the fixture, coefficient 0.15
	if !ok {
		t.Fatal("spell 23881 not found")
	}
	if math.Abs(s.Coefficient-0.15) > 1e-9 {
		t.Errorf("Coefficient = %v, want the table's 0.15", s.Coefficient)
	}
	if s.CoefficientSource != "table" {
		t.Errorf("CoefficientSource = %q, want %q", s.CoefficientSource, "table")
	}
}

// The vanilla conventions, in one function so no ability file re-derives
// them: a direct spell gets cast_time/3.5, a periodic one duration/15,
// and a hybrid class gets half.
func TestCoefficientConvention(t *testing.T) {
	cases := []struct {
		name       string
		castMS     int32
		durationMS int32
		hybrid     bool
		want       float64
		source     string
	}{
		{"three second cast", 3000, 0, false, 3.0 / 3.5, "convention"},
		{"instant direct", 0, 0, false, 1.5 / 3.5, "convention"},
		{"fifteen second dot", 0, 15000, false, 1.0, "convention"},
		{"hybrid three second cast", 3000, 0, true, 3.0 / 3.5 / 2, "convention"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, src := CoefficientFor(tc.castMS, tc.durationMS, tc.hybrid)
			if math.Abs(got-tc.want) > 1e-9 {
				t.Errorf("CoefficientFor(%d, %d, %v) = %v, want %v", tc.castMS, tc.durationMS, tc.hybrid, got, tc.want)
			}
			if src != tc.source {
				t.Errorf("source = %q, want %q", src, tc.source)
			}
		})
	}
}

// A cast time under the global cooldown is treated as a GCD cast, which
// is the vanilla rule and the reason an instant nuke is not coefficient
// zero.
func TestInstantCastUsesTheGlobalCooldown(t *testing.T) {
	got, _ := CoefficientFor(500, 0, false)
	want := 1.5 / 3.5
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("a 0.5s cast gave %v, want the GCD-floored %v", got, want)
	}
}

func TestLoadRejectsAMissingFile(t *testing.T) {
	if _, err := Load("testdata/nope.json"); err == nil {
		t.Fatal("loading a missing file returned no error")
	}
}
