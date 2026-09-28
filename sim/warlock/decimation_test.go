package warlock

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// DecimationRank2TalentsString sets only Decimation rank 2 (field 12 of
// the 19-node Demonology tree), the max-rank numbers the talent text and
// wowhead both give: -90% Soul Fire cooldown passively, and a 10 s
// window (-40% cast time, -40% cost) after Shadow Bolt or Searing Pain
// lands.
const DecimationRank2TalentsString = "-000000000002"

func newDecimationTestWarlock(t *testing.T) (*core.Simulation, *Warlock, *core.Unit) {
	t.Helper()

	player := core.WithSpec(
		&proto.Player{
			Class:         proto.Class_ClassWarlock,
			Race:          proto.Race_RaceOrc,
			Level:         60,
			Equipment:     &proto.EquipmentSpec{},
			Buffs:         core.FullBuffs.Player,
			TalentsString: DecimationRank2TalentsString,
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
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)

	sim := core.NewSim(&proto.RaidSimRequest{
		Raid: raid,
		Encounter: &proto.Encounter{
			Duration: 60,
			Targets:  []*proto.Target{core.DefaultTargetProtoLvl60},
		},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()

	agent, ok := sim.Raid.Parties[0].Players[0].(WarlockAgent)
	if !ok {
		t.Fatal("the raid's first player is not a warlock agent")
	}
	built := agent.GetWarlock()
	target := sim.Encounter.TargetUnits[0]
	return sim, built, target
}

// TestDecimationPassiveReducesSoulFireCooldown checks the always-on half
// of the talent: rank 2 shortens Soul Fire's 1-minute cooldown by 90%
// regardless of any proc.
func TestDecimationPassiveReducesSoulFireCooldown(t *testing.T) {
	_, built, _ := newDecimationTestWarlock(t)

	if len(built.SoulFire) == 0 {
		t.Fatal("level-60 warlock has no Soul Fire registered")
	}
	const tolerance = time.Millisecond
	for _, spell := range built.SoulFire {
		want := time.Minute / 10 // 90% reduction of the base 1-minute CD.
		got := spell.CD.Duration
		diff := got - want
		if diff < 0 {
			diff = -diff
		}
		if diff > tolerance {
			t.Errorf("Soul Fire (rank %d) CD.Duration = %v, want %v (90%% reduction)", spell.Rank, got, want)
		}
	}
}

// TestDecimationProcReducesSoulFireCastTimeAndCost casts Shadow Bolt (the
// target has no health bar in this test encounter, so the "below 35%
// health" gate treats it as eligible, matching how the ladder's bare
// dummy targets behave) and checks the 10 s Decimation buff comes up and
// discounts Soul Fire's cast time and mana cost, then reverts once it
// expires.
func TestDecimationProcReducesSoulFireCastTimeAndCost(t *testing.T) {
	sim, built, target := newDecimationTestWarlock(t)

	if len(built.SoulFire) == 0 || len(built.ShadowBolt) == 0 {
		t.Fatal("level-60 warlock is missing Soul Fire or Shadow Bolt")
	}
	soulFire := built.SoulFire[len(built.SoulFire)-1]
	shadowBolt := built.ShadowBolt[len(built.ShadowBolt)-1]

	baseCastTime := soulFire.DefaultCast.CastTime
	baseCostMultiplier := soulFire.Cost.Multiplier

	shadowBolt.ApplyEffects(sim, target, shadowBolt)

	// Shadow Bolt's damage (and so its OnSpellHitDealt callback, which is
	// what activates the Decimation buff) is deferred behind the
	// missile's travel time (spell.WaitTravelTime), same as Aimed Shot;
	// step the sim until that pending action fires.
	aura := built.GetAura("Decimation")
	for i := 0; i < 200 && !aura.IsActive(); i++ {
		if done := sim.Step(); done {
			break
		}
	}
	if !aura.IsActive() {
		t.Fatal("Shadow Bolt landed on an eligible target but the Decimation buff did not activate")
	}

	wantCastTime := baseCastTime - time.Duration(float64(SoulFireCastTime)*0.40)
	if got := soulFire.DefaultCast.CastTime; got != wantCastTime {
		t.Errorf("Soul Fire cast time during Decimation = %v, want %v", got, wantCastTime)
	}
	wantCostMultiplier := baseCostMultiplier - 40
	if got := soulFire.Cost.Multiplier; got != wantCostMultiplier {
		t.Errorf("Soul Fire cost multiplier during Decimation = %d, want %d", got, wantCostMultiplier)
	}

	// Advance past the 10 s window and confirm both revert.
	for i := 0; i < 2000; i++ {
		if !aura.IsActive() {
			break
		}
		if done := sim.Step(); done {
			break
		}
	}
	if aura.IsActive() {
		t.Fatal("Decimation aura is still active after its 10 s duration should have elapsed")
	}
	if got := soulFire.DefaultCast.CastTime; got != baseCastTime {
		t.Errorf("Soul Fire cast time after Decimation expired = %v, want %v", got, baseCastTime)
	}
	if got := soulFire.Cost.Multiplier; got != baseCostMultiplier {
		t.Errorf("Soul Fire cost multiplier after Decimation expired = %d, want %d", got, baseCostMultiplier)
	}
}
