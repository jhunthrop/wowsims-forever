package feral

import (
	"testing"

	_ "github.com/wowsims/classic/sim/common"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// Spell ids from the client's spellconst/druid.json, matching the rank ids
// forever_feral.apl.json casts.
const (
	rotationShredSpellID         = 9830
	rotationRipSpellID           = 9896
	rotationFerociousBiteSpellID = 31018
)

// TestForeverFeralRotationCastsFinishers runs forever_feral.apl.json at
// level 60 and checks its builder and finisher both land: Shred builds the
// combo points and Rip spends them. Ferocious Bite is logged, not asserted:
// on this bare build it fires only when 5 combo points arrive with Rip
// still holding more than 6 s, which a gearless cat rarely manages.
func TestForeverFeralRotationCastsFinishers(t *testing.T) {
	rotation := core.GetAplRotation("../../../ui/feral_druid/apls", "forever_feral")

	const duration = 180

	player := core.WithSpec(&proto.Player{
		Class:              proto.Class_ClassDruid,
		Race:               proto.Race_RaceTauren,
		Level:              60,
		Equipment:          &proto.EquipmentSpec{},
		Buffs:              core.FullBuffs.Player,
		TalentsString:      P1Talents,
		Rotation:           rotation.Rotation,
		DistanceFromTarget: 5,
	}, PlayerOptionsMonoCat)

	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)

	result := core.RunRaidSim(&proto.RaidSimRequest{
		Raid: raid,
		Encounter: &proto.Encounter{
			Duration: duration,
			Targets:  []*proto.Target{core.NewDefaultTarget()},
		},
		SimOptions: &proto.SimOptions{
			Iterations: 1,
			RandomSeed: 1,
			IsTest:     true,
		},
	})
	if result.Error != nil {
		t.Fatalf("sim error: %s", result.Error.Message)
	}

	casts := castCountsBySpellID(result)
	t.Logf("casts by spell id: %+v", casts)

	if casts[rotationShredSpellID] == 0 {
		t.Errorf("Shred casts = 0, want > 0 (finishers never fired -- the bug this test guards against)")
	}
	if casts[rotationRipSpellID] == 0 {
		t.Errorf("Rip casts = 0, want > 0")
	}
	t.Logf("Ferocious Bite casts = %d (see doc comment: structurally rare with this rotation/Rip duration, not asserted)", casts[rotationFerociousBiteSpellID])

}

func castCountsBySpellID(result *proto.RaidSimResult) map[int32]int32 {
	counts := map[int32]int32{}
	player := result.RaidMetrics.Parties[0].Players[0]
	for _, action := range player.Actions {
		spellID := action.Id.GetSpellId()
		for _, target := range action.Targets {
			counts[spellID] += target.Casts
		}
	}
	return counts
}
