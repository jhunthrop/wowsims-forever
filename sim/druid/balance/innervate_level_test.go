package balance

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/druid"
)

func newBalanceDruidAtLevel(t *testing.T, level int32) *BalanceDruid {
	t.Helper()
	player := core.WithSpec(&proto.Player{
		Class:              proto.Class_ClassDruid,
		Race:               proto.Race_RaceTauren,
		Level:              level,
		Equipment:          &proto.EquipmentSpec{},
		DistanceFromTarget: 30,
		Rotation:           &proto.APLRotation{},
	}, PlayerOptionsAdaptive)
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter:  &proto.Encounter{Duration: 60, Targets: []*proto.Target{core.NewDefaultTarget()}},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	built, ok := sim.Raid.Parties[0].Players[0].(*BalanceDruid)
	if !ok {
		t.Fatal("the raid's first player is not a *BalanceDruid")
	}
	return built
}

// TestInnervateIsLearnedAtForty: SkillLineAbility teaches Innervate (29166)
// at SpellLevels 40 (druid.InnervateLevel). It was registered, and autocast
// as a mana cooldown, at every level, so the bare 20 and 30 Balance and
// Restoration druids each got a free Innervate a fight.
func TestInnervateIsLearnedAtForty(t *testing.T) {
	if druid.InnervateLevel[0] != 40 {
		t.Fatalf("Innervate's client level = %d, want 40", druid.InnervateLevel[0])
	}
	for _, c := range []struct {
		level int32
		want  bool
	}{{30, false}, {39, false}, {40, true}, {60, true}} {
		d := newBalanceDruidAtLevel(t, c.level)
		if got := d.Innervate != nil; got != c.want {
			t.Errorf("level %d: Innervate registered = %v, want %v", c.level, got, c.want)
		}
		if got := d.GetMajorCooldown(core.ActionID{SpellID: 29166, Tag: d.Index}) != nil; got != c.want {
			t.Errorf("level %d: Innervate is a major cooldown = %v, want %v", c.level, got, c.want)
		}
	}
}
