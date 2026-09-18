package stats

import (
	"strings"
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
)

func TestStatsAdd(t *testing.T) {
	a := Stats{
		Intellect: 1,
	}
	b := Stats{
		Intellect: 1,
	}
	expectedResult := Stats{
		Intellect: 2,
	}

	result := a.Add(b)

	if !result.Equals(expectedResult) {
		t.Fatalf("Expected equal stats but were not equal: %s, %s", result, expectedResult)
	}
}

func TestStatsEquals_Success(t *testing.T) {
	a := Stats{
		Intellect: 1,
	}
	b := Stats{
		Intellect: 1,
	}

	if !a.Equals(b) {
		t.Fatalf("Expected equal stats but were not equal: %s, %s", a, b)
	}
}

func TestStatsEquals_Failure(t *testing.T) {
	a := Stats{
		Intellect: 1,
	}
	b := Stats{
		Intellect: 0,
	}

	if a.Equals(b) {
		t.Fatalf("Expected not equal stats but were equal: %s, %s", a, b)
	}
}

func TestStatsEqualsWithTolerance_Success(t *testing.T) {
	a := Stats{
		Intellect: 1,
	}
	b := Stats{
		Intellect: 0.5,
	}

	if !a.EqualsWithTolerance(b, 0.5) {
		t.Fatalf("Expected equal stats but were not equal: %s, %s", a, b)
	}
}

func TestStatsEqualsWithTolerance_Failure(t *testing.T) {
	a := Stats{
		Intellect: 1,
	}
	b := Stats{
		Intellect: 0.4,
	}

	if a.EqualsWithTolerance(b, 0.5) {
		t.Fatalf("Expected not equal stats but were equal: %s, %s", a, b)
	}
}

// The Go Stat enum and proto.Stat are index-synced: Stat(v) is how a
// proto value becomes a Go one (see ProtoArrayToStatsList), so a
// divergence silently reads the wrong stat. Forever merges MeleeHit and
// SpellHit into Hit and MeleeCrit and SpellCrit into Crit, which shifts
// every later index, so the sync is checked rather than commented.
func TestStatsProtoInSync(t *testing.T) {
	d := proto.Stat_StatStrength.Descriptor().Values()
	if d.Len() != int(Len) {
		t.Fatalf("Unequal number of stats defined in proto.Stats (%d) and Go (%d)", d.Len(), Len)
	}

	for i := 0; i < d.Len(); i++ {
		enum := d.Get(i)
		protoName := enum.Name()
		goName := Stat(enum.Number()).StatName()
		sanitizedGoName := strings.ReplaceAll(goName, " ", "")
		if string(protoName) != "Stat"+sanitizedGoName {
			t.Fatalf("Encountered stat enum %d in proto.Stats with name %s differs from Go enum name %s (ignoring Stat prefix)", enum.Number(), protoName, goName)
		}
	}
}

// Forever has one hit stat and one crit stat.
func TestForeverHasOneHitAndOneCritStat(t *testing.T) {
	if Hit >= Len || Crit >= Len {
		t.Fatal("Hit and Crit must be real stats")
	}
	if Hit.StatName() != "Hit" {
		t.Errorf("Hit.StatName() = %q, want %q", Hit.StatName(), "Hit")
	}
	if Crit.StatName() != "Crit" {
		t.Errorf("Crit.StatName() = %q, want %q", Crit.StatName(), "Crit")
	}
	for i := Stat(0); i < Len; i++ {
		switch i.StatName() {
		case "MeleeHit", "SpellHit", "MeleeCrit", "SpellCrit":
			t.Errorf("index %d still has the split stat %q", int(i), i.StatName())
		}
	}
}

// Forever has ten races. Skyborne is one neutral race carried as two
// rows because its second active differs by faction; the data lane's
// races.json has the same ten and the same two, and a mismatch would
// let the site offer a race the engine cannot build.
func TestForeverHasTenRaces(t *testing.T) {
	var n int
	for v := range proto.Race_name {
		if v != int32(proto.Race_RaceUnknown) {
			n++
		}
	}
	if n != 10 {
		t.Errorf("proto.Race has %d playable values, want 10", n)
	}
	for _, want := range []proto.Race{proto.Race_RaceHighOrderSkyborne, proto.Race_RaceWindshaperSkyborne} {
		if _, ok := proto.Race_name[int32(want)]; !ok {
			t.Errorf("proto.Race is missing %v", want)
		}
	}
}
