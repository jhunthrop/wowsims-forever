package healing

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/healsim"
)

const rotationsDir = "../../../ui/healing_priest/apls"

const (
	rotationIterations = 40
	rotationSeconds    = 180
	// maxOverheal is the most a sensible rotation overheals on the test
	// profile.
	maxOverheal = 0.8
)

// rotationCase is one spec's written rotation and what it must cast.
type rotationCase struct {
	name     string
	file     string
	talents  string
	spellIDs []int32
}

var rotationCases = []rotationCase{
	{"holy", "forever_holy", HolyTalents, []int32{1240827, 25315, 6064}},
	{"discipline", "forever_discipline", DiscTalents, []int32{1316995, 10901, 6064}},
}

func loadRotation(t *testing.T, file string) *proto.APLRotation {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(rotationsDir, file+".apl.json"))
	if err != nil {
		t.Fatalf("reading the written rotation: %v", err)
	}
	return core.APLRotationFromJsonString(string(data))
}

func runRotation(t *testing.T, c rotationCase) *proto.RaidSimResult {
	t.Helper()
	player := healer(60, c.talents, &proto.HealingPriest_Options{UseInnerFire: true}, loadRotation(t, c.file))
	result := core.RunRaidSim(healsim.Request(player, healsim.TestProfile(), rotationSeconds, rotationIterations))
	if result.Error != nil {
		t.Fatalf("%s: %s", c.name, result.Error.Message)
	}
	return result
}

// TestWrittenRotationsHealTheFakeRaid runs each spec's rotation against
// the shared test profile and checks it heals without wasting the heals,
// casts the spells it is written around and spends the mana those casts
// cost.
func TestWrittenRotationsHealTheFakeRaid(t *testing.T) {
	client := loadClient(t)
	for _, c := range rotationCases {
		t.Run(c.name, func(t *testing.T) {
			result := runRotation(t, c)
			player := result.RaidMetrics.Parties[0].Players[healsim.HealerIndex]
			if player.EffectiveHps.Avg <= 0 {
				t.Fatalf("effective HPS %v, want > 0", player.EffectiveHps.Avg)
			}
			overheal := 1 - player.EffectiveHps.Avg/player.Hps.Avg
			if overheal < 0 || overheal >= maxOverheal {
				t.Errorf("overheal share %.3f, want 0 to %.1f", overheal, maxOverheal)
			}
			casts := map[int32]int32{}
			for _, action := range player.Actions {
				for _, target := range action.Targets {
					casts[action.Id.GetSpellId()] += target.Casts
				}
			}
			spent := map[int32]float64{}
			for _, resource := range player.Resources {
				if resource.Type == proto.ResourceType_ResourceTypeMana && resource.Gain < 0 {
					spent[resource.Id.GetSpellId()] -= resource.Gain
				}
			}
			for _, id := range c.spellIDs {
				if casts[id] == 0 {
					t.Errorf("the rotation never cast spell %d (casts %v)", id, casts)
					continue
				}
				row, _ := client.ByID(id)
				ratio := spent[id] / (float64(casts[id]) * row.Cost)
				if math.IsNaN(ratio) || ratio < 0.6 || ratio > 1.0001 {
					t.Errorf("spell %d spent %.0f mana over %d casts of %.0f: %.2f of the list price", id, spent[id], casts[id], row.Cost, ratio)
				}
			}
			t.Logf("%s: effective HPS %.0f, overheal %.3f, time to OOM %.0f s, casts %v", c.name, player.EffectiveHps.Avg, overheal, player.Tto.Avg, casts)
		})
	}
}
