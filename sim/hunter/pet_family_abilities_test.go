package hunter

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage/clientdamagetest"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
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

// Dismember (Crocolisk), Pinch (Crab) and Dust Cloud (Tallstrider) are
// the other trainable Forever family abilities with a SkillLineAbility
// row. Sonic Blast (Bat) has none for any rank, so a hunter cannot teach
// it and no pet casts it.
func TestPetFamilyDirectAbilitiesMatchClient(t *testing.T) {
	cases := []struct {
		name     string
		petType  proto.Hunter_Options_PetType
		wantID   int32
		wantRoll [2]float64
		wantCost float64
		wantCD   time.Duration
	}{
		{"Dismember", proto.Hunter_Options_Crocolisk, 1264933, [2]float64{50.25, 57.75}, 35, 6 * time.Second},
		{"Pinch", proto.Hunter_Options_Crab, 1264742, [2]float64{88.4028, 101.5972}, 50, 30 * time.Second},
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
			for i := range c.wantRoll {
				if got := spell.ClientBaseDamage[i]; got < c.wantRoll[i]-0.01 || got > c.wantRoll[i]+0.01 {
					t.Errorf("roll = %v, want %v", spell.ClientBaseDamage, c.wantRoll)
					break
				}
			}
			if got := spell.Cost.BaseCost; got != c.wantCost {
				t.Errorf("focus cost = %v, want %v", got, c.wantCost)
			}
			if got := spell.CD.Duration; got != c.wantCD {
				t.Errorf("cooldown = %v, want %v", got, c.wantCD)
			}
		})
	}
}

func TestBatHasNoSonicBlast(t *testing.T) {
	_, built, _ := newFamilyPetHunter(t, proto.Hunter_Options_Bat)
	if built.pet.familyAbility != nil {
		t.Error("Sonic Blast has no SkillLineAbility row for any rank; the client does not teach it")
	}
}

func TestPetCastsDismemberOnCooldown(t *testing.T) {
	sim, built, target := newFamilyPetHunter(t, proto.Hunter_Options_Crocolisk)
	pet := built.pet
	pet.Enable(sim, pet)
	pet.CurrentTarget = target

	dismember := pet.familyAbility
	for i := 0; i < 20000 && sim.CurrentTime < 20*time.Second; i++ {
		if sim.Step() {
			break
		}
	}
	// One cast at the start and one every 6 seconds after it.
	if got := dismember.SpellMetrics[target.UnitIndex].Casts; got < 3 {
		t.Errorf("Dismember cast %d times in 20 seconds, want at least 3", got)
	}
}

// Dust Cloud's rows (1265899 .. 1265904): an armor reduction aura (effect
// 6, aura 22) of 65/175/285/395/505 for 30 seconds, 10 focus, no
// cooldown of its own.
func TestDustCloudArmorReductionsMatchTheClient(t *testing.T) {
	class := clientdamagetest.Load(t, "../core/testdata/conformance/client/hunter.json")
	for rank := 1; rank <= DustCloudRanks; rank++ {
		spell, ok := class.ByID(DustCloudSpellId[rank])
		if !ok {
			t.Fatalf("rank %d: spell %d is not in the client table", rank, DustCloudSpellId[rank])
		}
		if got, want := dustCloudArmorReduction[rank], -spell.Effects[0].Amount; got != want {
			t.Errorf("rank %d: armor reduction %v, client %v", rank, got, want)
		}
		if got, want := dustCloudDuration, time.Duration(spell.DurationMS)*time.Millisecond; got != want {
			t.Errorf("rank %d: duration %v, client %v", rank, got, want)
		}
	}
}

func TestDustCloudReducesArmorForItsDurationAndIsRecast(t *testing.T) {
	sim, built, target := newFamilyPetHunter(t, proto.Hunter_Options_Tallstrider)
	pet := built.pet
	cloud := pet.familyAbility
	if cloud == nil || cloud.ActionID.SpellID != 1265904 {
		t.Fatalf("Tallstrider family ability = %v, want Dust Cloud rank 5", cloud)
	}

	before := target.GetStat(stats.Armor)
	pet.Enable(sim, pet)
	pet.CurrentTarget = target
	for i := 0; i < 2000 && cloud.SpellMetrics[target.UnitIndex].Casts == 0; i++ {
		if sim.Step() {
			break
		}
	}
	if cloud.SpellMetrics[target.UnitIndex].Casts == 0 {
		t.Fatal("the Tallstrider never cast Dust Cloud")
	}
	if got := before - target.GetStat(stats.Armor); got != 505 {
		t.Errorf("Dust Cloud removed %v armor, want 505", got)
	}

	for i := 0; i < 40000 && sim.CurrentTime < 70*time.Second; i++ {
		if sim.Step() {
			break
		}
	}
	if got := cloud.SpellMetrics[target.UnitIndex].Casts; got < 2 {
		t.Errorf("Dust Cloud cast %d times in 70 seconds, want it renewed when the 30 s lapses", got)
	}
	if got := before - target.GetStat(stats.Armor); got != 505 && got != 0 {
		t.Errorf("armor reduction %v while renewing; it must not stack", got)
	}
}
