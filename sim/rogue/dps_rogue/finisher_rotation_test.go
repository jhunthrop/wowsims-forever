package dpsrogue

import (
	"fmt"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// finisherRunResult is one RunRaidSim's rogue DPS and per-spell cast counts
// (average per iteration), keyed by spell id.
type finisherRunResult struct {
	dps   float64
	casts map[int32]float64
}

// runRotation runs the Combat preset gear and talents through RunRaidSim
// with the given APL JSON for the given fight length and iteration count.
func runRotation(t *testing.T, aplJSON string, durationSeconds float64, iterations int32) finisherRunResult {
	t.Helper()

	player := core.WithSpec(
		&proto.Player{
			Class:         proto.Class_ClassRogue,
			Race:          proto.Race_RaceHuman,
			Level:         60,
			Equipment:     core.GetGearSet("../../../ui/rogue/gear_sets", "combat_sinister_strike_prebis").GearSet,
			Buffs:         core.FullBuffs.Player,
			Consumes:      Phase1Consumes.Consumes,
			TalentsString: CombatSwordsTalents,
			Rotation:      core.APLRotationFromJsonString(aplJSON),
		},
		DefaultRogue,
	)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)
	result := core.RunRaidSim(&proto.RaidSimRequest{
		Raid: raid,
		Encounter: &proto.Encounter{
			Duration: durationSeconds,
			Targets:  []*proto.Target{core.DefaultTargetProtoLvl60},
		},
		SimOptions: &proto.SimOptions{RandomSeed: 7, Iterations: iterations},
	})
	if result.Error != nil {
		t.Fatalf("RunRaidSim: %s", result.Error.Message)
	}

	playerMetrics := result.RaidMetrics.Parties[0].Players[0]
	casts := map[int32]float64{}
	for _, action := range playerMetrics.Actions {
		for _, target := range action.Targets {
			casts[action.Id.GetSpellId()] += float64(target.Casts) / float64(iterations)
		}
	}
	return finisherRunResult{dps: playerMetrics.Dps.Avg, casts: casts}
}

const (
	ruptureSpellID    = 11275
	sinisterStrikeID  = 11294
	rotationBoilerFmt = `{"type":"TypeAPL","priorityList":[%s]}`
)

// TestRuptureDotIsActiveGatesRecasts casts Rupture only while its dot is not
// active and Sinister Strike otherwise. A Rupture that lasts 8-16 s must be
// recast at most once per 8 s over a 60 s fight; a dot the APL never sees as
// active would be recast at every combo-point opportunity (about every 5
// Sinister Strikes, i.e. 10+ times).
func TestRuptureDotIsActiveGatesRecasts(t *testing.T) {
	apl := fmt.Sprintf(rotationBoilerFmt, `
		{"action":{"condition":{"and":{"vals":[
			{"cmp":{"op":"OpGe","lhs":{"currentComboPoints":{}},"rhs":{"const":{"val":"4"}}}},
			{"not":{"val":{"dotIsActive":{"spellId":{"spellId":11275}}}}}]}},
		 "castSpell":{"spellId":{"spellId":11275}}}},
		{"action":{"castSpell":{"spellId":{"spellId":11294}}}}`)

	got := runRotation(t, apl, 60, 50)

	t.Logf("dps %.1f casts %v", got.dps, got.casts)
	if got.casts[ruptureSpellID] < 3 {
		t.Errorf("Rupture casts = %.2f, want at least 3 over 60s", got.casts[ruptureSpellID])
	}
	if got.casts[ruptureSpellID] > 8 {
		t.Errorf("Rupture casts = %.2f over 60s, want at most 8: dotIsActive is not seeing the bleed", got.casts[ruptureSpellID])
	}
}
