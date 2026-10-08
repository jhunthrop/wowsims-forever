package conformance

import (
	"os"
	"strings"
	"testing"
)

// clientBuild is the build the committed client rows come from.
const clientBuild = "1.60.1.70291"

// TestSetBonusConformance holds every item set the engine registers to the
// client's ItemSetSpell rows: each client threshold must exist, each flat
// stat bonus must add exactly the client's amount, and the result is the
// committed sets.golden.md (FOREVER_UPDATE_GOLDEN=1 rewrites it).
func TestSetBonusConformance(t *testing.T) {
	report := BuildSetsReport(clientBuild)
	content := RenderSetsGolden(report)

	var failures []string
	for _, row := range report.Rows {
		if row.Verdict == VerdictMismatch || row.Verdict == VerdictMissingThreshold {
			failures = append(failures, row.SetName+" "+row.Verdict+": "+row.SpellName+" "+row.Note)
		}
	}
	if len(failures) > 0 {
		t.Errorf("set bonuses that do not match the client:\n%s", strings.Join(failures, "\n"))
	}

	if os.Getenv("FOREVER_UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(SetsGoldenPath, []byte(content), 0o644); err != nil {
			t.Fatalf("writing %s: %v", SetsGoldenPath, err)
		}
		return
	}
	committed, err := os.ReadFile(SetsGoldenPath)
	if err != nil {
		t.Fatalf("%s is missing; run FOREVER_UPDATE_GOLDEN=1 go test ./sim/conformance/... and commit it: %v", SetsGoldenPath, err)
	}
	if string(committed) != content {
		t.Errorf("%s is stale; run FOREVER_UPDATE_GOLDEN=1 go test ./sim/conformance/... and review the diff", SetsGoldenPath)
	}
}
