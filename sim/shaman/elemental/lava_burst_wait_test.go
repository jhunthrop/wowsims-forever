package elemental

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// lavaBurstCastsPerIteration runs the Forever Elemental rotation on an
// ungeared level-60 shaman with the given spell haste (percent) and
// returns Lava Burst rank 4's casts per iteration.
func lavaBurstCastsPerIteration(t *testing.T, hastePercent float64) float64 {
	t.Helper()
	const iterations = 300
	// A deep mana pool keeps the cycle running all fight: the question is timing, not mana.
	bonus := stats.Stats{stats.SpellHaste: hastePercent * core.HasteRatingPerHastePercent, stats.Mana: 100000}
	player := core.WithSpec(
		&proto.Player{
			Class:         proto.Class_ClassShaman,
			Race:          proto.Race_RaceOrc,
			Level:         60,
			Equipment:     &proto.EquipmentSpec{},
			TalentsString: "5510331300001350-030000000000000000-0533520000000000",
			Rotation:      core.GetAplRotation("../../../ui/elemental_shaman/apls", "forever_elemental").Rotation,
			BonusStats:    &proto.UnitStats{Stats: bonus.ToFloatArray()},
		},
		&proto.Player_ElementalShaman{
			ElementalShaman: &proto.ElementalShaman{
				Options: &proto.ElementalShaman_Options{},
			},
		},
	)
	result := core.RunRaidSim(&proto.RaidSimRequest{
		Raid: core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter: &proto.Encounter{
			Duration: 180,
			Targets:  []*proto.Target{core.DefaultTargetProtoLvl60},
		},
		SimOptions: &proto.SimOptions{Iterations: iterations, RandomSeed: 1, IsTest: true},
	})
	if result.Error != nil {
		t.Fatalf("sim error: %v", result.Error.Message)
	}
	for _, action := range result.RaidMetrics.Parties[0].Players[0].Actions {
		if action.Id.GetSpellId() != 1326047 {
			continue
		}
		casts := int32(0)
		for _, target := range action.Targets {
			casts += target.Casts
		}
		return float64(casts) / iterations
	}
	t.Fatal("Lava Burst rank 4 was never cast")
	return 0
}

// TestASliverOfHasteDoesNotCostLavaBursts: with Elemental Alacrity,
// Lightning Bolt and Lava Burst are 2.0 s casts, so five bolts end
// exactly as Lava Burst's 10 s cooldown comes back. Any haste at all
// ends the fifth bolt a hair early; a rotation with no wait then starts
// a sixth bolt and Lava Burst slips 2 s, every cycle. That made a
// +0.01% haste probe lose 0.35% of bare Elemental 60 (round 13 casters).
// A player waits the fraction of a second for Lava Burst, and so does
// the rotation.
func TestASliverOfHasteDoesNotCostLavaBursts(t *testing.T) {
	base := lavaBurstCastsPerIteration(t, 0)
	hasted := lavaBurstCastsPerIteration(t, 0.01)
	if hasted < 0.99*base {
		t.Errorf("Lava Burst casts per fight: %.2f at +0.01%% haste, %.2f at none; a sliver of haste must not delay it", hasted, base)
	}
}
