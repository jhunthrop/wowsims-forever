package warlock

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// boltSpeedYardsPerSecond is SpellMisc.Speed for every Shadow Bolt rank (686
// to 25307) and every Incinerate rank (412758, 1293812, 1293813) in client
// 1.60.1.70753: the bolt travels at 20 yards a second, so from 30 yards it
// lands 1.5 s after the cast completes (Soul Fire and Death Coil, 24, were
// already modelled).
const boltSpeedYardsPerSecond = 20

func newTravelTestWarlock(t *testing.T, distance float64) *Warlock {
	t.Helper()
	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassWarlock,
			Race:               proto.Race_RaceOrc,
			Level:              60,
			Equipment:          &proto.EquipmentSpec{},
			TalentsString:      IncinerateOnlyTalentsString,
			DistanceFromTarget: distance,
		},
		&proto.Player_Warlock{
			Warlock: &proto.Warlock{
				Options: &proto.WarlockOptions{
					Armor:       proto.WarlockOptions_NoArmor,
					Summon:      proto.WarlockOptions_NoSummon,
					WeaponImbue: proto.WarlockOptions_NoWeaponImbue,
				},
			},
		},
	)
	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid: raid,
		Encounter: &proto.Encounter{
			Duration: 60,
			Targets:  []*proto.Target{core.DefaultTargetProtoLvl60},
		},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	agent, ok := sim.Raid.Parties[0].Players[0].(WarlockAgent)
	if !ok {
		t.Fatal("the raid's first player is not a warlock agent")
	}
	return agent.GetWarlock()
}

func TestShadowBoltAndIncinerateTravelAtTheClientSpeed(t *testing.T) {
	const distance = 30.0
	warlock := newTravelTestWarlock(t, distance)
	want := time.Duration(float64(time.Second) * distance / boltSpeedYardsPerSecond)

	spells := map[string][]*core.Spell{"Shadow Bolt": warlock.ShadowBolt, "Incinerate": warlock.Incinerate}
	for name, ranks := range spells {
		if len(ranks) == 0 {
			t.Fatalf("no %s rank registered", name)
		}
		for _, spell := range ranks {
			if got := spell.TravelTime(); got != want {
				t.Errorf("%s %d lands %v after its cast from %v yd, want %v (client speed %d)",
					name, spell.ActionID.SpellID, got, distance, want, boltSpeedYardsPerSecond)
			}
		}
	}
}
