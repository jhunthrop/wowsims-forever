package retribution

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// In 1.60.1.70707 Judgement of Righteousness (20286 and its ranks) and
// Judgement of Command (20968) moved to the melee table: SpellCategories
// DefenseType 2 (Era 1.15 carries 1, magic), and SpellMisc Attributes_0
// gains 0x200000, SPELL_ATTR0_NO_ACTIVE_DEFENSE (no dodge, parry or block):
// 20286 0x50000 -> 0x250000, 20968 329728 -> 2426880.
func TestJudgementsRollTheMeleeTableWithoutActiveDefenses(t *testing.T) {
	raid := core.SinglePlayerRaidProto(utilitySealPaladin(60, `{"type": "TypeAPL"}`), core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 60, Targets: []*proto.Target{core.NewDefaultTarget()}},
		SimOptions: &proto.SimOptions{Iterations: 1, RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	unit := &sim.Raid.Parties[0].Players[0].GetCharacter().Unit
	judgement := unit.GetSpell(core.ActionID{SpellID: 20286})
	if judgement == nil {
		t.Fatal("Judgement of Righteousness 20286 is not registered at 60")
	}
	if judgement.DefenseType != core.DefenseTypeMelee {
		t.Errorf("Judgement of Righteousness defense type = %v, want melee", judgement.DefenseType)
	}

}
