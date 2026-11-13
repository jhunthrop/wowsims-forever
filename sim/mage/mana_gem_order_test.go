package mage

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// gemTestMage is a level-60 mage with every gem (Ruby, Citrine, Jade and
// Agate), its bar emptied, carrying conjured in the conjured slot.
func gemTestMage(t *testing.T, conjured proto.Conjured) (*core.Simulation, *Mage) {
	t.Helper()
	player := core.WithSpec(&proto.Player{
		Class: proto.Class_ClassMage, Race: proto.Race_RaceTroll, Level: 60,
		Equipment:          &proto.EquipmentSpec{},
		Buffs:              &proto.IndividualBuffs{},
		Consumes:           &proto.Consumes{DefaultConjured: conjured},
		TalentsString:      ForeverFrostTalents,
		DistanceFromTarget: 20,
		Rotation:           &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
	}, PlayerOptions)
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter:  &proto.Encounter{Duration: 300, Targets: []*proto.Target{core.DefaultTargetProtoLvl60}},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()
	mage := sim.Raid.Parties[0].Players[0].(MageAgent).GetMage()
	mage.SpendMana(sim, mage.CurrentMana(), mage.NewManaMetrics(core.ActionID{OtherID: proto.OtherAction_OtherActionNone}))
	return sim, mage
}

func gemWantsToPop(sim *core.Simulation, mage *Mage, rank int) bool {
	mcd := mage.GetMajorCooldown(core.ActionID{ItemID: ManaGemItemId[rank]})
	if mcd == nil {
		return false
	}
	return mcd.ShouldActivate(sim, &mage.Character)
}

// Gems and runes share one two-minute category (1153), so a 180 s raid fight
// has two uses of it. A small gem popped while a bigger one is in the bag
// wastes one: the published raid 60 Arcane mage used its Ruby and then its
// Mana Agate (375 to 425) in 95 fights of 100 and never the Demonic Rune
// its preset carries. The gems go largest first.
func TestManaGemsGoLargestFirst(t *testing.T) {
	sim, mage := gemTestMage(t, proto.Conjured_ConjuredUnknown)
	if !gemWantsToPop(sim, mage, 4) {
		t.Fatal("an empty mage does not want its Mana Ruby")
	}
	for rank := 1; rank <= 3; rank++ {
		if gemWantsToPop(sim, mage, rank) {
			t.Errorf("gem rank %d wants to pop while the Mana Ruby is unused", rank)
		}
	}
	if !mage.ManaGem[4].Cast(sim, nil) {
		t.Fatal("the Mana Ruby did not cast")
	}
	mage.SpendMana(sim, mage.CurrentMana(), mage.NewManaMetrics(core.ActionID{OtherID: proto.OtherAction_OtherActionNone}))
	if !gemWantsToPop(sim, mage, 3) {
		t.Error("the Mana Citrine does not want to pop once the Ruby is spent")
	}
	for rank := 1; rank <= 2; rank++ {
		if gemWantsToPop(sim, mage, rank) {
			t.Errorf("gem rank %d wants to pop while the Mana Citrine is unused", rank)
		}
	}
}

// A Demonic Rune (900 to 1500) restores more than every gem below the Ruby,
// and the mage has one for every use of the shared category, so with a rune
// in the conjured slot the mage spends its biggest gem and then runes.
func TestManaGemsBelowTheBiggestHoldForADemonicRune(t *testing.T) {
	sim, mage := gemTestMage(t, proto.Conjured_ConjuredDemonicRune)
	if !mage.ManaGem[4].Cast(sim, nil) {
		t.Fatal("the Mana Ruby did not cast")
	}
	mage.SpendMana(sim, mage.CurrentMana(), mage.NewManaMetrics(core.ActionID{OtherID: proto.OtherAction_OtherActionNone}))
	for rank := 1; rank <= 3; rank++ {
		if gemWantsToPop(sim, mage, rank) {
			t.Errorf("gem rank %d wants to pop with a Demonic Rune in the conjured slot", rank)
		}
	}
	rune := mage.GetMajorCooldown(core.ActionID{ItemID: 12662})
	if rune == nil || !rune.ShouldActivate(sim, &mage.Character) {
		t.Error("the Demonic Rune does not want to pop on an empty bar")
	}
}
