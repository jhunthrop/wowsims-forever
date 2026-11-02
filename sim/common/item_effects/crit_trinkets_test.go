package item_effects_test

import (
	"testing"
	"time"

	engine "github.com/wowsims/classic/sim"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
)

// newTrinketMage is a level-60 mage wearing one trinket, no talents or buffs.
func newTrinketMage(t *testing.T, trinket int32) (*core.Simulation, *core.Character) {
	t.Helper()
	engine.RegisterAll()
	items := make([]*proto.ItemSpec, proto.ItemSlot_ItemSlotRanged+1)
	for i := range items {
		items[i] = &proto.ItemSpec{}
	}
	items[proto.ItemSlot_ItemSlotTrinket1] = &proto.ItemSpec{Id: trinket}
	player := core.WithSpec(&proto.Player{
		Name:      "Trinket Test",
		Race:      proto.Race_RaceUndead,
		Class:     proto.Class_ClassMage,
		Level:     60,
		Equipment: &proto.EquipmentSpec{Items: items},
		Rotation:  &proto.APLRotation{},
	}, &proto.Player_Mage{Mage: &proto.Mage{Options: &proto.Mage_Options{}}})
	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 60, Targets: []*proto.Target{core.DefaultTargetProtoLvl60}},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()
	return sim, sim.Raid.Parties[0].Players[0].GetCharacter()
}

// The 1.60.1.70707 client links Blackhand's Breadth (13965) to a use
// effect, ItemEffect 231535 -> 1318944: aura 306, 5% crit chance against
// the marked target, 20 sec (SpellDuration 18), CoolDownMSec 300000 and the
// 20 sec trinket category; and Eye of the Beast (13968) to ItemEffect
// 231536 -> 1287840: aura 55, 7% spell hit for 20 sec, the same cooldowns.
func TestBlackrockCritTrinketsUseEffects(t *testing.T) {
	cases := []struct {
		item  int32
		check func(*core.Character, stats.Stats, stats.Stats) bool
		want  string
	}{
		{13965, func(c *core.Character, before, after stats.Stats) bool {
			return after[stats.Crit]-before[stats.Crit] == 5*core.CritRatingPerCritChance
		}, "+5% crit"},
		{13968, func(c *core.Character, before, after stats.Stats) bool {
			return c.PseudoStats.SchoolBonusHitChance[stats.SchoolIndexFire] == 7*core.HitRatingPerHitChance &&
				c.PseudoStats.SchoolBonusHitChance[stats.SchoolIndexPhysical] == 0
		}, "+7% spell hit on magic schools only"},
	}
	for _, tc := range cases {
		sim, character := newTrinketMage(t, tc.item)
		use := character.GetSpell(core.ActionID{ItemID: tc.item})
		if use == nil {
			t.Fatalf("item %d registers no use effect", tc.item)
		}
		if use.CD.Duration != 5*time.Minute {
			t.Errorf("item %d cooldown %v, want 5m", tc.item, use.CD.Duration)
		}
		before := character.GetStats()
		use.Cast(sim, character.CurrentTarget)
		if !tc.check(character, before, character.GetStats()) {
			t.Errorf("item %d use: want %s", tc.item, tc.want)
		}
	}
}
