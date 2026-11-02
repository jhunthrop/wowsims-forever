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
// 5 min item cooldown. Spell 18385 sits in category 30 with a 60 s
// category recovery, the category of the mana gems, Demonic Rune and the
// healthstones, so using the robe holds the conjured items for a minute.
func TestRobeOfTheArchmageRestoresManaAndHoldsTheGems(t *testing.T) {
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
	if got := mage.GetConjuredCD().TimeToReady(sim); got < time.Minute {
		t.Errorf("the conjured items are ready in %v after the robe, want a 60 s category hold", got)
	}
}

// Round 11 (casters, from the items lane): the category is shared both
// ways. The mana gems' use spells (5405, 10052, 10057, 10058) and Demonic
// Rune's (16666) sit in category 30 with the robe's 18385, each with a
// 60 s CategoryRecoveryTime (SpellCategories, SpellCooldowns), so a gem
// holds the robe for a minute too.
func TestAManaGemHoldsRobeOfTheArchmageForAMinute(t *testing.T) {
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
	if robe.CanCast(sim, nil) {
		t.Error("Robe of the Archmage castable right after a mana gem; category 30 holds it for 60 s")
	}
	sim.CurrentTime += time.Minute
	if !robe.CanCast(sim, nil) {
		t.Error("Robe of the Archmage still held 60 s after the mana gem")
	}
}
