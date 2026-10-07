package hunter

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// newFamilyPetHunter builds a bare level-60 hunter with the given pet
// family out, stepped to the point the pet exists in a running sim.
func newFamilyPetHunter(t *testing.T, petType proto.Hunter_Options_PetType) (*core.Simulation, *Hunter, *core.Unit) {
	t.Helper()

	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassHunter,
			Race:               proto.Race_RaceOrc,
			Level:              60,
			Equipment:          &proto.EquipmentSpec{},
			Buffs:              &proto.IndividualBuffs{},
			DistanceFromTarget: 25,
		},
		&proto.Player_Hunter{
			Hunter: &proto.Hunter{
				Options: &proto.Hunter_Options{
					Ammo:           proto.Hunter_Options_RazorArrow,
					PetType:        petType,
					PetUptime:      1,
					PetAttackSpeed: 2.0,
				},
			},
		},
	)
	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 60, Targets: []*proto.Target{core.DefaultTargetProtoLvl60}},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()

	agent, ok := sim.Raid.Parties[0].Players[0].(HunterAgent)
	if !ok {
		t.Fatal("the raid's first player is not a hunter agent")
	}
	built := agent.GetHunter()
	if built.pet == nil {
		t.Fatal("hunter has no pet")
	}
	return sim, built, sim.Encounter.TargetUnits[0]
}

func TestPetFamilyPeriodicAbilitiesMatchClient(t *testing.T) {
	cases := []struct {
		name     string
		petType  proto.Hunter_Options_PetType
		wantID   int32
		wantTick float64
		wantCost float64
		wantCD   time.Duration
		wantDot  int32
	}{
		{"Savage Rend", proto.Hunter_Options_Raptor, 1265069, 26, 50, 60 * time.Second, 6},
		{"Tendon Rip", proto.Hunter_Options_Hyena, 1265042, 20, 25, 30 * time.Second, 3},
		{"Web", proto.Hunter_Options_Spider, 1265883, 13, 20, 40 * time.Second, 4},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, built, _ := newFamilyPetHunter(t, c.petType)
			spell := built.pet.familyAbility
			if spell == nil {
				t.Fatalf("%s pet has no family ability", c.name)
			}
			if got := spell.ActionID.SpellID; got != c.wantID {
				t.Errorf("spell id = %d, want %d", got, c.wantID)
			}
			if got := spell.ClientBaseDamage; got != [2]float64{c.wantTick, c.wantTick} {
				t.Errorf("per-tick damage = %v, want %v", got, c.wantTick)
			}
			if got := spell.Cost.BaseCost; got != c.wantCost {
				t.Errorf("focus cost = %v, want %v", got, c.wantCost)
			}
			if got := spell.CD.Duration; got != c.wantCD {
				t.Errorf("cooldown = %v, want %v", got, c.wantCD)
			}
			if got := spell.Dot(built.pet.CurrentTarget).NumberOfTicks; got != c.wantDot {
				t.Errorf("ticks = %d, want %d", got, c.wantDot)
			}
		})
	}
}

func TestCatHasNoFamilyAbility(t *testing.T) {
	_, built, _ := newFamilyPetHunter(t, proto.Hunter_Options_Cat)
	if built.pet.familyAbility != nil {
		t.Error("the Cat family has no trainable periodic ability in the client")
	}
}

func TestPetCastsSavageRendOnCooldown(t *testing.T) {
	sim, built, target := newFamilyPetHunter(t, proto.Hunter_Options_Raptor)
	pet := built.pet
	pet.Enable(sim, pet)
	pet.CurrentTarget = target

	rend := pet.familyAbility
	for i := 0; i < 2000 && rend.SpellMetrics[target.UnitIndex].Casts == 0; i++ {
		if sim.Step() {
			break
		}
	}
	if rend.SpellMetrics[target.UnitIndex].Casts == 0 {
		t.Fatal("the Raptor never cast Savage Rend with full focus and the ability off cooldown")
	}
	if got := sim.CurrentTime; got > 5*time.Second {
		t.Errorf("Savage Rend was first cast at %v, want within the first seconds of the fight", got)
	}
}
