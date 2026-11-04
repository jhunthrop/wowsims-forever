package warlock

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// TestImmolateCastTimeTakesCastSpeed: Immolate's 2 s cast (11668, cast-time
// index 2000 ms) shortens with cast speed like every other cast-time spell
// (Berserking, Spellstone, haste gear). Its own CastTime hook used to return
// the unhasted DefaultCast.CastTime, so no haste ever reached it.
func TestImmolateCastTimeTakesCastSpeed(t *testing.T) {
	player := core.WithSpec(
		&proto.Player{
			Class:     proto.Class_ClassWarlock,
			Race:      proto.Race_RaceOrc,
			Level:     60,
			Equipment: &proto.EquipmentSpec{},
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

	warlock := sim.Raid.Parties[0].Players[0].(WarlockAgent).GetWarlock()
	immolate := warlock.Immolate[len(warlock.Immolate)-1]
	shadowBolt := warlock.ShadowBolt[len(warlock.ShadowBolt)-1]

	warlock.MultiplyCastSpeed(2)

	if got, want := shadowBolt.CastTime(), 1500*time.Millisecond; got != want {
		t.Fatalf("Shadow Bolt at double cast speed = %v, want %v (the control)", got, want)
	}
	if got, want := immolate.CastTime(), 1000*time.Millisecond; got != want {
		t.Errorf("Immolate at double cast speed = %v, want %v", got, want)
	}
}
