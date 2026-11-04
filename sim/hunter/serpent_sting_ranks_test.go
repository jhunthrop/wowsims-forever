package hunter

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// Round 19 (hunter lane): a hunter keeps every Serpent Sting rank in the
// spellbook, as it keeps every Arcane Shot and Aimed Shot rank, so a
// rotation (or a site rank-choice row) can cast a lower rank. The engine
// registered only the highest learned rank, so a downranked Serpent Sting
// line found no spell and dropped out of the rotation.
func TestSerpentStingRegistersEveryLearnedRank(t *testing.T) {
	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassHunter,
			Race:               proto.Race_RaceOrc,
			Level:              53,
			Equipment:          &proto.EquipmentSpec{},
			DistanceFromTarget: 25,
		},
		P1PlayerOptions,
	)
	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid: raid,
		Encounter: &proto.Encounter{
			Duration: 30,
			Targets:  []*proto.Target{core.DefaultTargetProtoLvl60},
		},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()

	built := sim.Raid.Parties[0].Players[0].(HunterAgent).GetHunter()

	// Ranks 1-7 are learned by 53 (rank 7, 13554, at 50; rank 8 at 58).
	for rank, id := range map[int]int32{1: 1978, 5: 13552, 6: 13553, 7: 13554} {
		if built.GetSpell(core.ActionID{SpellID: id}) == nil {
			t.Errorf("Serpent Sting rank %d (%d) is not registered at 53", rank, id)
		}
	}
	if built.GetSpell(core.ActionID{SpellID: 13555}) != nil {
		t.Errorf("Serpent Sting rank 8 (13555) is registered at 53; it is learned at 58")
	}
	if built.SerpentSting == nil || built.SerpentSting.ActionID.SpellID != 13554 {
		t.Errorf("hunter.SerpentSting should stay the highest learned rank, 13554")
	}
}
