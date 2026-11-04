package warlock

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
	goproto "google.golang.org/protobuf/proto"
)

// Forever demons scale from their summoner's stats (Blizzard's warlock deep
// dive, 2026-10-07) through Warlock Pet Scaling 416189, the same
// server-computed channels as Hunter Pet Scaling 415429, and the hunter deep
// dive says such pets "can no longer receive player buffs that increase their
// stats". The 70755 client does it by flagging every group buff "can only
// target players" (SpellMisc Attributes_3 0x100). Heart of the Lion's area
// buff (409583) carries no such flag and targets every nearby ally
// (ImplicitTarget 31), so a demon in a raid hunter's area takes it, as the
// hunter's own pet does (fork c6f2e362e): +10% to every stat and 40 + 4 a
// level attack power (280 at 60). A level-60 Succubus summoned under every
// raid buff has Strength 129 and Agility 85 of her own; with the buff that is
// 141.9 and 93.5, and 141.9 x 2 - 20 + 280 = 543.8 attack power before what
// she inherits from the warlock.
func TestDemonReceivesOnlyHeartOfTheLionFromPlayerBuffs(t *testing.T) {
	player := core.WithSpec(
		&proto.Player{
			Class:     proto.Class_ClassWarlock,
			Race:      proto.Race_RaceOrc,
			Level:     60,
			Equipment: &proto.EquipmentSpec{},
			Buffs:     core.FullBuffs.Player,
		},
		&proto.Player_Warlock{Warlock: &proto.Warlock{Options: &proto.WarlockOptions{
			Armor:  proto.WarlockOptions_NoArmor,
			Summon: proto.WarlockOptions_Succubus,
		}}},
	)
	raidBuffs := goproto.Clone(core.FullBuffs.Raid).(*proto.RaidBuffs)
	raidBuffs.HeartOfTheLion = true // a raid hunter's area buff
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, raidBuffs, core.FullBuffs.Debuffs)
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 60, Targets: []*proto.Target{core.NewDefaultTarget()}},
		SimOptions: &proto.SimOptions{Iterations: 1, RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()
	for i := 0; i < 10 && sim.CurrentTime <= 0; i++ {
		sim.Step()
	}
	agent, ok := sim.Raid.Parties[0].Players[0].(WarlockAgent)
	if !ok {
		t.Fatal("player 0 did not build as a warlock")
	}
	succubus := agent.GetWarlock().Succubus
	if !succubus.IsEnabled() {
		t.Fatal("the Succubus is not summoned")
	}
	if !hasActiveAura(succubus.GetAuras(), 409583) {
		t.Error("the Succubus does not carry Heart of the Lion's area buff (409583)")
	}
	got := succubus.GetStats()
	if math.Abs(got[stats.Strength]-141.9) > 1e-6 || math.Abs(got[stats.Agility]-93.5) > 1e-6 {
		t.Errorf("Succubus Strength/Agility = %.2f/%.2f, want 129/85 under Heart of the Lion's +10%% only (141.9/93.5)", got[stats.Strength], got[stats.Agility])
	}
	if got[stats.AttackPower] < 543.8-1e-6 {
		t.Errorf("Succubus attack power = %.1f, want at least 141.9 x 2 - 20 + Heart of the Lion's 280 = 543.8", got[stats.AttackPower])
	}
}

func hasActiveAura(auras []*core.Aura, spellID int32) bool {
	for _, aura := range auras {
		if aura.ActionID.SpellID == spellID && aura.IsActive() {
			return true
		}
	}
	return false
}
