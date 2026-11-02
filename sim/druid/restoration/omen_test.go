package restoration

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/druid"
)

// Omen of Clarity's proc row (16864, SpellAuraOptions ProcTypeMask
// 0x14014) includes "deal helpful spell" (0x4000): casting a heal over
// time rolls it like any other heal. Rejuvenation, which has no direct
// heal, must be able to grant Clearcasting.
func TestRejuvenationCastsCanGrantClearcasting(t *testing.T) {
	rotation := core.APLRotationFromJsonString(`{"type": "TypeAPL", "priorityList": [
		{"action": {"castSpell": {"spellId": {"spellId": 9841}, "target": {"type": "Self"}}}}]}`)
	player := newPlayer(60, StandardTalents, healerBonusStats, rotation)
	result := core.RunRaidSim(&proto.RaidSimRequest{
		Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter:  &proto.Encounter{Duration: 120, Targets: []*proto.Target{core.NewDefaultTarget()}},
		SimOptions: &proto.SimOptions{Iterations: 50, RandomSeed: 1, IsTest: true},
	})
	if result.Error != nil {
		t.Fatal(result.Error.Message)
	}
	var procs float64
	for _, aura := range result.RaidMetrics.Parties[0].Players[0].Auras {
		if aura.Id.GetSpellId() == druid.ClearcastingSpellId[0] {
			procs += aura.ProcsAvg
		}
	}
	if procs <= 0 {
		t.Error("two minutes of Rejuvenation casts granted no Clearcasting")
	}
}
