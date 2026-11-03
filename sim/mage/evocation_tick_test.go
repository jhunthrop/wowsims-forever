package mage

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// Evocation (12051) is an 8 s channel at +1500% spirit regen. Regen ticks
// every 2 s, and a pull's regen timer has no fixed phase against the
// mage's casts, so an 8 s channel always spans four ticks. The engine
// draws the regen grid's phase per iteration (core initManaTickAction), so
// a start exactly on a tick is now rare, but this test forces one. The tick at the
// start belongs to the time before the channel; the tick at the end must
// still see the channel's regen, or that Evocation gets three ticks.
func TestEvocationThatStartsOnARegenTickGetsFourTicks(t *testing.T) {
	player := core.WithSpec(&proto.Player{
		Class: proto.Class_ClassMage, Race: proto.Race_RaceTroll, Level: 60,
		Equipment:          &proto.EquipmentSpec{},
		Buffs:              &proto.IndividualBuffs{},
		TalentsString:      ForeverFrostTalents,
		DistanceFromTarget: 20,
		Rotation:           &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
	}, PlayerOptions)
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter:  &proto.Encounter{Duration: 60, Targets: []*proto.Target{core.DefaultTargetProtoLvl60}},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()
	mage := sim.Raid.Parties[0].Players[0].(MageAgent).GetMage()

	// Find the regen grid: step to the first tick after the pull and note
	// its time.
	mage.SpendMana(sim, mage.CurrentMana(), mage.NewManaMetrics(core.ActionID{OtherID: proto.OtherAction_OtherActionNone}))
	var tickAt time.Duration
	for last := mage.CurrentMana(); !sim.Step(); last = mage.CurrentMana() {
		if mage.CurrentMana() > last && sim.CurrentTime > 0 {
			tickAt = sim.CurrentTime
			break
		}
	}
	if tickAt == 0 {
		t.Fatal("no regen tick seen")
	}
	start := tickAt + 2*time.Second
	core.StartDelayedAction(sim, core.DelayedActionOptions{
		DoAt:     start,
		Priority: core.ActionPriorityGCD,
		OnAction: func(sim *core.Simulation) {
			mage.SpendMana(sim, mage.CurrentMana(), mage.NewManaMetrics(core.ActionID{OtherID: proto.OtherAction_OtherActionNone}))
			// The idle rotation's wait can hold the global past an
			// off-grid tick; the test owns this moment.
			mage.GCD.Reset()
			if !mage.Evocation.Cast(sim, nil) {
				t.Fatal("Evocation did not cast")
			}
		},
	})
	// A tick at Evocation's rate is sixteen times a plain one; count only
	// those.
	plainTick := mage.ManaRegenPerSecondWhileNotCasting() * 2
	ticks := 0
	var perTick float64
	for last := mage.CurrentMana(); sim.CurrentTime <= start+9*time.Second && !sim.Step(); last = mage.CurrentMana() {
		if gain := mage.CurrentMana() - last; gain > 4*plainTick {
			ticks++
			perTick = gain
		}
	}
	if mage.CurrentMana() >= mage.MaxMana() {
		t.Fatalf("mana capped at %.0f; the tick count is not readable", mage.MaxMana())
	}
	if ticks != 4 {
		t.Errorf("Evocation started on a regen tick gained %d boosted ticks (%.0f mana each), want 4", ticks, perTick)
	}
}
