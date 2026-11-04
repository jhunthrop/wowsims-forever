package hunter

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// Round 19 (hunter lane): every learned Raptor Strike rank stays in the
// spellbook with its own on-next-swing queue, so a rotation can queue a
// lower rank (Survival at 46-47, rank 5, where the cheaper rank pays for
// more Immolation Traps). All ranks share one 6 second cooldown. The
// engine registered only the highest rank.
func TestRaptorStrikeRegistersEveryLearnedRankOnOneCooldown(t *testing.T) {
	player := core.WithSpec(
		&proto.Player{
			Class:     proto.Class_ClassHunter,
			Race:      proto.Race_RaceOrc,
			Level:     46,
			Equipment: &proto.EquipmentSpec{},
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

	top := built.GetSpell(core.ActionID{SpellID: RaptorStrikeSpellId[6]})
	if top == nil || built.RaptorStrike != top {
		t.Fatalf("hunter.RaptorStrike should be rank 6 (%d), the highest learned at 46", RaptorStrikeSpellId[6])
	}
	if built.GetSpell(core.ActionID{SpellID: RaptorStrikeSpellId[7]}) != nil {
		t.Errorf("rank 7 is registered at 46; it is learned at 48")
	}
	for rank := 1; rank <= 6; rank++ {
		id := RaptorStrikeSpellId[rank]
		spell := built.GetSpell(core.ActionID{SpellID: id})
		if spell == nil {
			t.Errorf("Raptor Strike rank %d (%d) is not registered at 46", rank, id)
			continue
		}
		if built.GetSpell(core.ActionID{SpellID: id}.WithTag(3)) == nil {
			t.Errorf("Raptor Strike rank %d has no queue spell (tag 3)", rank)
		}
		if built.GetSpell(core.ActionID{SpellID: id}.WithTag(1)) == nil {
			t.Errorf("Raptor Strike rank %d has no hit spell (tag 1)", rank)
		}
		if spell.CD.Timer != top.CD.Timer {
			t.Errorf("Raptor Strike rank %d does not share rank 6's cooldown", rank)
		}
	}
}
