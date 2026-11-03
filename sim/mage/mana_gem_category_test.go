package mage

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// ItemEffect (1.60.1.70753): Mana Agate 5514, Mana Jade 5513, Mana Citrine
// 8007 and Mana Ruby 8008 are SpellCategory 1153 with a 120000 ms category
// cooldown, the same category as Demonic Rune 12662 and Dark Rune 20520.
// So a gem and a rune hold each other for two minutes, and each gem has its
// own 2 min cooldown through that category.
func TestManaGemAndDemonicRuneShareTheTwoMinuteCategory(t *testing.T) {
	player := core.WithSpec(&proto.Player{
		Class: proto.Class_ClassMage, Race: proto.Race_RaceTroll, Level: 60,
		Equipment:          &proto.EquipmentSpec{},
		Buffs:              &proto.IndividualBuffs{},
		Consumes:           &proto.Consumes{DefaultConjured: proto.Conjured_ConjuredDemonicRune},
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

	ruby := mage.ManaGem[4]
	if ruby == nil {
		t.Fatal("a level-60 mage has no Mana Ruby")
	}
	rune := mage.GetSpell(core.ActionID{ItemID: 12662})
	if rune == nil {
		t.Fatal("the Demonic Rune is not registered")
	}
	mage.SpendMana(sim, mage.CurrentMana(), mage.NewManaMetrics(core.ActionID{OtherID: proto.OtherAction_OtherActionNone}))
	if !ruby.Cast(sim, nil) {
		t.Fatal("the Mana Ruby did not cast")
	}
	if rune.CanCast(sim, nil) {
		t.Error("the Demonic Rune is usable right after a Mana Ruby; category 1153 holds it")
	}
	if got := rune.ReadyAt() - sim.CurrentTime; got != 2*time.Minute {
		t.Errorf("the Demonic Rune is ready in %v after a Mana Ruby, want 2m", got)
	}
	for rank := 1; rank <= ManaGemRanks; rank++ {
		if gem := mage.ManaGem[rank]; gem != nil && gem.CD.Duration != 2*time.Minute {
			t.Errorf("gem rank %d cooldown is %v, want 2m", rank, gem.CD.Duration)
		}
	}
}
