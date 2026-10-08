package tankwarrior

import (
	"strings"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/warrior"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Item ids of the unit tests' weapons, from the fork's item table
// (--tags=with_db): Spineshatter (a one-hand mace) and Aegis of the Blood
// God (a shield with block value 77).
const (
	testMainHandID = 19335
	testShieldID   = 19862
	testBlockValue = 77
)

// buildTank builds a tank warrior at level 60 with the unit tests' mace
// and, when shield is set, shield; the raid names it as the boss's tank so
// the health bar and the healing-free damage metrics exist. It returns the
// warrior and the reset simulation.
func buildTank(t *testing.T, talents string, shield bool) (*warrior.Warrior, *core.Simulation) {
	t.Helper()

	items := make([]*proto.ItemSpec, int(proto.ItemSlot_ItemSlotRanged)+1)
	for i := range items {
		items[i] = &proto.ItemSpec{}
	}
	items[proto.ItemSlot_ItemSlotMainHand] = &proto.ItemSpec{Id: testMainHandID}
	if shield {
		items[proto.ItemSlot_ItemSlotOffHand] = &proto.ItemSpec{Id: testShieldID}
	}

	sim := core.NewSim(&proto.RaidSimRequest{
		SimOptions: &proto.SimOptions{RandomSeed: 1},
		Raid: &proto.Raid{
			Parties: []*proto.Party{{
				Players: []*proto.Player{{
					Name:            "Tank",
					Class:           proto.Class_ClassWarrior,
					Race:            proto.Race_RaceOrc,
					Level:           60,
					TalentsString:   talents,
					Consumes:        &proto.Consumes{},
					Buffs:           &proto.IndividualBuffs{},
					Spec:            PlayerOptionsBasic,
					Equipment:       &proto.EquipmentSpec{Items: items},
					InFrontOfTarget: true,
					HealingModel:    &proto.HealingModel{Hps: 1, BurstWindow: 6},
				}},
				Buffs: &proto.PartyBuffs{},
			}},
			Tanks: []*proto.UnitReference{{Type: proto.UnitReference_Player, Index: 0}},
		},
		Encounter: &proto.Encounter{
			Targets:  []*proto.Target{{Name: "boss", Level: 63, TankIndex: 0, SwingSpeed: 2, MinBaseDamage: 2000, DamageSpread: 0.33}},
			Duration: 60,
		},
	}, simsignals.CreateSignals())
	sim.Reset()

	agent, ok := sim.Raid.Parties[0].Players[0].(warrior.WarriorAgent)
	if !ok {
		t.Fatal("the raid's first player is not a warrior agent")
	}
	return agent.GetWarrior(), sim
}

// emptyTalents is a talent string with no points spent.
const emptyTalents = "-"

// talentsWith returns talents with the named talent's rank set, where
// field is the talent's name in the generated proto (WarriorTalents) and
// the string is read positionally against warrior.TalentTreeSizes.
func talentsWith(t *testing.T, talents string, field string, rank int) string {
	t.Helper()

	fd := (&proto.WarriorTalents{}).ProtoReflect().Descriptor().Fields().ByName(protoreflect.Name(field))
	if fd == nil {
		t.Fatalf("WarriorTalents has no field named %q", field)
	}
	pos, tree := int(fd.Number())-1, 0
	for tree < len(warrior.TalentTreeSizes) && pos >= warrior.TalentTreeSizes[tree] {
		pos -= warrior.TalentTreeSizes[tree]
		tree++
	}

	parts := strings.Split(talents, "-")
	for len(parts) < len(warrior.TalentTreeSizes) {
		parts = append(parts, "")
	}
	chars := []rune(parts[tree] + strings.Repeat("0", pos+1))
	chars[pos] = rune('0' + rank)
	parts[tree] = string(chars[:max(len(parts[tree]), pos+1)])
	return strings.Join(parts, "-")
}
