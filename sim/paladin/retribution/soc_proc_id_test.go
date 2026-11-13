package retribution

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/spellconst"
)

const clientPaladinSpellconst = "../../core/testdata/conformance/client/paladin.json"

// sealOfCommandRankIDs are the five Seal of Command ranks of the Forever client.
var sealOfCommandRankIDs = []int32{20375, 20915, 20918, 20919, 20920}

// sealOfCommandEraProcIDs are the per-rank procs of the Era client, which the
// Forever client does not carry.
var sealOfCommandEraProcIDs = []int32{20944, 20945, 20946, 20947}

// Every Seal of Command rank in the Forever client triggers one proc spell,
// 20424 (70% weapon damage, 0.29 of spell damage), and the launch logs name
// that id on every proc under ranks 1, 2 and 3 (458 procs in our reports).
// The engine named ranks 2 to 5's procs with the Era client's 20944 to 20947.
func TestSealOfCommandProcIsTheClientsOneSpellAtEveryRank(t *testing.T) {
	client, err := spellconst.Load(clientPaladinSpellconst)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range sealOfCommandRankIDs {
		seal, ok := client.ByID(id)
		if !ok {
			t.Fatalf("Seal of Command %d is not in the client", id)
		}
		if got := seal.Effects[0].TriggerSpell; got != sealOfCommandProcID {
			t.Errorf("Seal of Command %d triggers %d in the client, the engine casts %d", id, got, sealOfCommandProcID)
		}
	}

	player := core.WithSpec(
		&proto.Player{
			Class:         proto.Class_ClassPaladin,
			Race:          proto.Race_RaceHuman,
			Level:         60,
			Equipment:     &proto.EquipmentSpec{},
			TalentsString: retributionTalentString(t, "seal_of_command", 1),
		},
		&proto.Player_RetributionPaladin{
			RetributionPaladin: &proto.RetributionPaladin{Options: optionsSealOfCommand},
		},
	)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)
	encounter := &proto.Encounter{Duration: 10, Targets: []*proto.Target{core.NewDefaultTarget()}}
	env, _, _ := core.NewEnvironment(raid, encounter, true)
	ret := env.Raid.Parties[0].Players[0].(*RetributionPaladin)
	unit := &ret.Paladin.Character.Unit

	if unit.GetSpell(core.ActionID{SpellID: sealOfCommandProcID}) == nil {
		t.Errorf("no Seal of Command proc %d registered at level 60", sealOfCommandProcID)
	}
	for _, id := range sealOfCommandEraProcIDs {
		if unit.GetSpell(core.ActionID{SpellID: id}) != nil {
			t.Errorf("the Era proc id %d is registered; the Forever client has no such spell", id)
		}
	}
}

const sealOfCommandProcID = 20424
