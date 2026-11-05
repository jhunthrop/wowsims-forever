package mage

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// Robe of the Archmage (14152), worn in every published mage set, has a
// use: ItemEffect 99905, spell 18385 "Restores $s1 mana", 500 with
// variance 0.5 (375 to 625, as Wowhead Forever's tooltip reads), on a
// 5 min item cooldown and SpellCategoryID 0. An item's own ItemEffect
// cooldown replaces its spell's SpellCooldowns and SpellCategories rows, so
// the robe starts no category: spell 18385's category 30 does not apply to
// it. Patch 1.4.0 says the same of the robe: "No longer triggers the
// cooldowns of a Mage's mana gems. Mana gems are now on their own unique
// cooldown category" (their ItemEffect category 1153, 2 min).
func TestRobeOfTheArchmageRestoresManaAndLeavesTheGems(t *testing.T) {
	player := core.WithSpec(&proto.Player{
		Class: proto.Class_ClassMage, Race: proto.Race_RaceTroll, Level: 60,
		Equipment: &proto.EquipmentSpec{Items: []*proto.ItemSpec{
			{}, {}, {}, {}, {Id: 14152},
		}},
		Buffs:              &proto.IndividualBuffs{},
		TalentsString:      ForeverFrostTalents,
		DistanceFromTarget: 20,
	}, PlayerOptions)
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter:  &proto.Encounter{Duration: 60, Targets: []*proto.Target{core.DefaultTargetProtoLvl60}},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()
	mage := sim.Raid.Parties[0].Players[0].(MageAgent).GetMage()

	mcd := mage.GetMajorCooldown(core.ActionID{ItemID: 14152})
	if mcd == nil {
		t.Fatal("Robe of the Archmage registers no use")
	}
	mage.SpendMana(sim, 2000, mage.NewManaMetrics(core.ActionID{OtherID: proto.OtherAction_OtherActionNone}))
	before := mage.CurrentMana()
	mcd.Spell.Cast(sim, nil)
	if got := mage.CurrentMana() - before; got < 375 || got > 625 {
		t.Errorf("the robe restored %.0f mana, want 375 to 625", got)
	}
	if got := mcd.Spell.CD.TimeToReady(sim); got != 5*time.Minute {
		t.Errorf("the robe's cooldown is %v, want 5m", got)
	}
	if got := mage.GetConjuredCD().TimeToReady(sim); got != 0 {
		t.Errorf("the conjured items are held %v after the robe, want no hold (ItemEffect category 0)", got)
	}
}

// Round 23 (items): the other way round. The mana gems' ItemEffect rows
// (5513, 5514, 8007, 8008) put them in category 1153 for 2 min with Demonic
// and Dark Rune and the healthstones; the robe's row is category 0. A gem
// does not hold the robe.
func TestAManaGemLeavesRobeOfTheArchmageReady(t *testing.T) {
	player := core.WithSpec(&proto.Player{
		Class: proto.Class_ClassMage, Race: proto.Race_RaceTroll, Level: 60,
		Equipment: &proto.EquipmentSpec{Items: []*proto.ItemSpec{
			{}, {}, {}, {}, {Id: 14152},
		}},
		Buffs:              &proto.IndividualBuffs{},
		TalentsString:      ForeverFrostTalents,
		DistanceFromTarget: 20,
	}, PlayerOptions)
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter:  &proto.Encounter{Duration: 180, Targets: []*proto.Target{core.DefaultTargetProtoLvl60}},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()
	mage := sim.Raid.Parties[0].Players[0].(MageAgent).GetMage()

	robe := mage.GetMajorCooldown(core.ActionID{ItemID: 14152}).Spell
	gem := mage.ManaGem[ManaGemRanks]
	if gem == nil {
		t.Fatal("level-60 mage has no Mana Ruby")
	}
	mage.SpendMana(sim, 3000, mage.NewManaMetrics(core.ActionID{OtherID: proto.OtherAction_OtherActionNone}))
	if !gem.Cast(sim, nil) {
		t.Fatal("the Mana Ruby did not cast")
	}
	if !robe.CanCast(sim, nil) {
		t.Error("Robe of the Archmage held right after a mana gem; the two share no category")
	}
}
