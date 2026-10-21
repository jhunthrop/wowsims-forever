package dpswarrior

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/warrior"
)

// buildWarriorWithWeaponsAndSim is buildWarriorWithWeapons
// (weapon_specs_test.go) with the *core.Simulation kept rather than
// discarded, for a test that needs to cast a spell against the built
// warrior's own target rather than only read a registered spell's
// static fields.
func buildWarriorWithWeaponsAndSim(t *testing.T, talents string, mainHand, offHand int32) (*warrior.Warrior, *core.Simulation) {
	t.Helper()

	items := []*proto.ItemSpec{}
	for i := 0; i < int(proto.ItemSlot_ItemSlotRanged)+1; i++ {
		items = append(items, &proto.ItemSpec{})
	}
	items[proto.ItemSlot_ItemSlotMainHand] = &proto.ItemSpec{Id: mainHand}
	if offHand != 0 {
		items[proto.ItemSlot_ItemSlotOffHand] = &proto.ItemSpec{Id: offHand}
	}

	sim := core.NewSim(&proto.RaidSimRequest{
		SimOptions: &proto.SimOptions{RandomSeed: 1},
		Raid: &proto.Raid{
			Parties: []*proto.Party{{
				Players: []*proto.Player{{
					Name:          "Warrior",
					Class:         proto.Class_ClassWarrior,
					Race:          proto.Race_RaceOrc,
					TalentsString: talents,
					Consumes:      &proto.Consumes{},
					Buffs:         &proto.IndividualBuffs{},
					Spec:          PlayerOptionsFury,
					Equipment:     &proto.EquipmentSpec{Items: items},
				}},
				Buffs: &proto.PartyBuffs{},
			}},
		},
		Encounter: &proto.Encounter{
			Targets:  []*proto.Target{{Name: "target", Level: 63, MobType: proto.MobType_MobTypeDemon}},
			Duration: 60,
		},
	}, simsignals.CreateSignals())
	sim.Reset()

	agent, ok := sim.Raid.Parties[0].Players[0].(warrior.WarriorAgent)
	if !ok {
		t.Fatalf("the raid's first player is not a warrior agent")
	}
	return agent.GetWarrior(), sim
}

// buildWarriorAndSimForCostTest is buildWarriorForCostTest
// (cost_discounts_test.go) with the *core.Simulation kept rather than
// discarded. A handful of the new talent tests need it: a rage grant
// deferred past Reset needs Step() to drain it, and a proc trigger
// needs ApplyEffects called against a real target with a real RNG
// behind sim.Proc.
func buildWarriorAndSimForCostTest(t *testing.T, talents string) (*warrior.Warrior, *core.Simulation) {
	t.Helper()

	sim := core.NewSim(&proto.RaidSimRequest{
		SimOptions: &proto.SimOptions{RandomSeed: 1},
		Raid: &proto.Raid{
			Parties: []*proto.Party{{
				Players: []*proto.Player{{
					Name:          "Warrior",
					Class:         proto.Class_ClassWarrior,
					Race:          proto.Race_RaceOrc,
					TalentsString: talents,
					Consumes:      &proto.Consumes{},
					Buffs:         &proto.IndividualBuffs{},
					Spec:          PlayerOptionsFury,
					Equipment:     &proto.EquipmentSpec{},
				}},
				Buffs: &proto.PartyBuffs{},
			}},
		},
		Encounter: &proto.Encounter{
			Targets:  []*proto.Target{{Name: "target", Level: 63, MobType: proto.MobType_MobTypeDemon}},
			Duration: 60,
		},
	}, simsignals.CreateSignals())
	sim.Reset()

	agent, ok := sim.Raid.Parties[0].Players[0].(warrior.WarriorAgent)
	if !ok {
		t.Fatalf("the raid's first player is not a warrior agent")
	}
	return agent.GetWarrior(), sim
}
