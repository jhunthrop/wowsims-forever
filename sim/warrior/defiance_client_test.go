package warrior

import (
	"encoding/json"
	"math"
	"os"
	"regexp"
	"strconv"
	"testing"
)

// Defiance's per-rank threat is the client's own: talent 12792's rank texts in
// the talent fixture, which the client renders from trait curve 109358 (the
// 1.60.1.70790 server push 113326 moved it from 5/10/15 to 7/13/20, as
// Blizzard's 3 November Forever notes say).
func TestDefianceThreatMatchesTheClientCurve(t *testing.T) {
	raw, err := os.ReadFile("../../tools/talentgen/testdata/warrior.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Trees []struct {
			Talents []struct {
				Name  string `json:"name"`
				Ranks []struct {
					Description string `json:"description"`
				} `json:"ranks"`
			} `json:"talents"`
		} `json:"trees"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	percent := regexp.MustCompile(`additional (\d+)%`)
	found := false
	for _, tree := range fixture.Trees {
		for _, talent := range tree.Talents {
			if talent.Name != "Defiance" {
				continue
			}
			found = true
			for i, rank := range talent.Ranks {
				m := percent.FindStringSubmatch(rank.Description)
				if m == nil {
					t.Fatalf("rank %d text %q states no percent", i+1, rank.Description)
				}
				want, _ := strconv.ParseFloat(m[1], 64)
				if got := 100 * defianceThreatByRank[i+1]; math.Abs(got-want) > 1e-9 {
					t.Errorf("Defiance rank %d: engine %v%%, client %v%%", i+1, got, want)
				}
			}
		}
	}
	if !found {
		t.Fatal("no Defiance in the talent fixture")
	}
}
