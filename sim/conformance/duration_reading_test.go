package conformance

import (
	"os"
	"strings"
	"testing"

	"github.com/wowsims/classic/sim/core/spellconst"
)

func TestOnlyUnmodeledControlEffects(t *testing.T) {
	slow := spellconst.Effect{Effect: effectApplyAura, Aura: auraDecreaseSpeed}
	damage := spellconst.Effect{Effect: effectSchoolDamage}
	periodic := spellconst.Effect{Effect: effectApplyAura, Aura: auraPeriodicDamage}
	interrupt := spellconst.Effect{Effect: effectInterruptCast}

	cases := []struct {
		name    string
		effects []spellconst.Effect
		want    bool
	}{
		{"damage plus slow (Frostbolt)", []spellconst.Effect{slow, damage}, true},
		{"interrupt lockout (Counterspell)", []spellconst.Effect{interrupt}, true},
		{"damage over time is a real duration", []spellconst.Effect{damage, periodic}, false},
		{"slow beside a damage over time", []spellconst.Effect{slow, periodic}, false},
		{"no control effect at all", []spellconst.Effect{damage}, false},
	}
	for _, c := range cases {
		if got := onlyUnmodeledControlEffects(spellconst.Spell{Effects: c.effects}); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestDurationReadingIgnoresClassesThatHaveNotAdopted(t *testing.T) {
	client := spellconst.Spell{DurationMS: 5000, Effects: []spellconst.Effect{{Effect: effectApplyAura, Aura: auraDecreaseSpeed}}}
	row := Row{ClassSlug: "hunter"}
	if got := durationReading(row, client, nil, nil); got != "" {
		t.Errorf("hunter reading = %q, want none", got)
	}
	row.ClassSlug = "mage"
	if got := durationReading(row, client, nil, nil); got != reasonUnmodeledEffect {
		t.Errorf("mage reading = %q, want the unmodeled-effect reason", got)
	}
}

func TestClassifiedTrainablesAreAllDispositioned(t *testing.T) {
	for slug := range trainableDispositions {
		golden, err := os.ReadFile(goldenPath(slug))
		if err != nil {
			t.Fatalf("%s: %v", slug, err)
		}
		if strings.Contains(string(golden), "| "+unclassifiedDisposition+" |") {
			t.Errorf("%s golden has an ability with no disposition in dispositions.go", slug)
		}
	}
}
