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
	rotationTigersFurySpellID    = 5217
	rotationShredSpellID         = 9830
	rotationRipSpellID           = 9896
	rotationFerociousBiteSpellID = 31018
)

// TestForeverFeralRotationCastsFinishers regression-tests the site's audit
// referenced in engine-fixes brief 1: before Tiger's Fury was made free
// (cost 0, 30 s cooldown, matching the client's spell 5217) it still charged
// 30 energy on a 1 s cooldown, so the forever_feral.apl.json rotation -
// which casts Tiger's Fury unconditionally at the top of the priority list
// - spent nearly every GCD's energy on it. The audit saw ~62 Tiger's Fury
// casts and zero Shred/Rip/Ferocious Bite casts over 180 s. This builds the
// same rotation at level 60 and asserts Shred and Rip land, and that Tiger's
// Fury's cast count now tracks its real 30 s cooldown instead of energy
// availability.
//
// Ferocious Bite is logged but not asserted: this rotation only casts it at
// 5 combo points with Rip already ticking, and this engine's Rip is a fixed
// 6 ticks * 2 s = 12 s regardless of combo points spent (sim/druid/rip.go,
// RipTicks), while building 5 combo points from 0 via Shred alone (60
// energy/cast, ~10.1 energy/s regen) takes ~30 s - longer than Rip's own
// duration - so with this specific rotation and this engine's numbers the
// Bite branch is structurally unreachable even over very long sims (checked
// out to 3600 s locally). That is a pre-existing Rip-duration/energy-budget
// question, not something Tiger's Fury's cost/cooldown controls, so it is
// out of this fix's scope; noting it here rather than asserting on it.
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

	// 180s / 30s cooldown = 6 possible casts, plus the opener before the
	// first cooldown has ticked down once = at most 7. The old code let
	// Tiger's Fury fire on its old 1 s cooldown any time 30 energy was
	// available, which is what produced the audit's ~62.
	if got := casts[rotationTigersFurySpellID]; got == 0 || got > 7 {
		t.Errorf("Tiger's Fury casts = %d, want 1-7 (its real 30 s cooldown over %d s)", got, duration)
	}
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
