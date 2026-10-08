package healing

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/healsim"
)

// Lightwell from the 1.60.1.70009 client (data/builds/1.60.1.70009/raw):
//
//	Spells 724 / 27870 / 27871 (ranks 1-3, spell levels 40 / 50 / 60):
//	  SpellPower mana 225 / 295 / 365, cast time 1500 ms, StartRecoveryTime
//	  1500, CategoryRecoveryTime 600000, duration index 26 (180000 ms);
//	  effect 50 (summon object) 181102 / 181105 / 181106. "Creates a holy
//	  Lightwell near the priest. Members of your raid or party can click
//	  the Lightwell to restore $7001o1 health over $7001d. Being attacked
//	  cancels the effect. Lightwell lasts for $d or 5 charges."
//	Renew spells 7001 / 27873 / 27874: effect 6 aura 8 (periodic heal),
//	  EffectBasePointsF 160 / 233 / 320 a tick, EffectAuraPeriod 2000,
//	  duration index 8 (10000 ms), no coefficient.

func lightwellProfile(pulseInterval float64) *proto.RaidDamageModel {
	return &proto.RaidDamageModel{
		Profile:              "lightwell-test",
		TankHealth:           9000,
		MemberHealth:         5000,
		DamageSpread:         0,
		PulseDamage:          600, // 12% of a member's health: below the 90% a member clicks at
		PulseIntervalSeconds: pulseInterval,
		PulseMembers:         4,
	}
}

func lightwellSim(t *testing.T, model *proto.RaidDamageModel) (*core.Simulation, *HealingPriest) {
	t.Helper()
	sim := core.NewSim(healsim.Request(healer(60, "", &proto.HealingPriest_Options{}, &proto.APLRotation{}), model, 300, 1), simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()
	agent, ok := sim.Raid.Parties[0].Players[0].(*HealingPriest)
	if !ok {
		t.Fatal("the raid's first player is not a healing priest")
	}
	return sim, agent
}

func TestLightwellIsRegisteredAtEveryLearnedRank(t *testing.T) {
	cases := []struct {
		level int32
		ids   []int32
	}{
		{60, []int32{724, 27870, 27871}},
		{50, []int32{724, 27870}},
		{40, []int32{724}},
		{39, nil},
	}
	costs := map[int32]float64{724: 225, 27870: 295, 27871: 365}
	for _, c := range cases {
		_, agent := agentSim(t, healer(c.level, "", &proto.HealingPriest_Options{}, &proto.APLRotation{}))
		ranks := registered(agent.Lightwell)
		if len(ranks) != len(c.ids) {
			t.Errorf("level %d: %d Lightwell ranks, want %d", c.level, len(ranks), len(c.ids))
		}
		for i, id := range c.ids {
			spell := ranks[i+1]
			if spell == nil || spell.SpellID != id {
				t.Errorf("level %d: rank %d = %v, want spell %d", c.level, i+1, spell, id)
				continue
			}
			if spell.DefaultCast.Cost != costs[id] {
				t.Errorf("Lightwell %d costs %v, want %v", id, spell.DefaultCast.Cost, costs[id])
			}
			if spell.DefaultCast.CastTime != 1500*time.Millisecond {
				t.Errorf("Lightwell %d casts in %v, want 1.5s", id, spell.DefaultCast.CastTime)
			}
			if spell.CD.Duration != 10*time.Minute {
				t.Errorf("Lightwell %d cooldown = %v, want 10m", id, spell.CD.Duration)
			}
		}
	}
}

// Casting it leaves a well of 5 charges standing for 3 minutes.
func TestLightwellStandsForThreeMinutesWithFiveCharges(t *testing.T) {
	sim, agent := lightwellSim(t, lightwellProfile(0))
	top := agent.Lightwell[3]
	top.ApplyEffects(sim, &agent.Unit, top)
	if got := agent.LightwellCharges(); got != 5 {
		t.Fatalf("a new Lightwell has %d charges, want 5", got)
	}
	if got := agent.LightwellAura.RemainingDuration(sim); got != 3*time.Minute {
		t.Errorf("the Lightwell stands for %v, want 3m", got)
	}
}

// Nobody is hurt, so nobody clicks: the well keeps its charges and heals
// nothing.
func TestAUnusedLightwellKeepsItsCharges(t *testing.T) {
	sim, agent := lightwellSim(t, lightwellProfile(0))
	top := agent.Lightwell[3]
	top.ApplyEffects(sim, &agent.Unit, top)
	for !sim.Step() && sim.CurrentTime < time.Minute {
	}
	if got := agent.LightwellCharges(); got != 5 {
		t.Errorf("an unused Lightwell has %d charges after a minute, want 5", got)
	}
}

// Pulses twelve seconds apart hurt all four members, who click; each click
// heals 5 ticks of 320 before the next pulse can cancel it, and the well is
// spent after five clicks: 25 ticks, 8000 health.
func TestEachLightwellChargeHealsFiveTicksOf320(t *testing.T) {
	sim, agent := lightwellSim(t, lightwellProfile(12))
	top := agent.Lightwell[3]
	top.ApplyEffects(sim, &agent.Unit, top)
	for !sim.Step() {
	}
	if got := agent.LightwellCharges(); got != 0 {
		t.Errorf("a used Lightwell has %d charges left, want 0", got)
	}
	if healing := lightwellHealing(t, agent); healing != 5*5*320 {
		t.Errorf("Lightwell renew healed %.0f, want 25 ticks of 320 = 8000", healing)
	}
}

// Being attacked cancels the renew: with a pulse every 3 seconds no click
// reaches its fifth tick.
func TestBeingAttackedCancelsALightwellRenew(t *testing.T) {
	sim, agent := lightwellSim(t, lightwellProfile(3))
	top := agent.Lightwell[3]
	top.ApplyEffects(sim, &agent.Unit, top)
	for !sim.Step() {
	}
	if healing := lightwellHealing(t, agent); healing >= 5*5*320 || healing == 0 {
		t.Errorf("Lightwell renew healed %.0f under constant attack, want some but fewer than the 8000 five uninterrupted clicks give", healing)
	}
}

// lightwellHealing is the raw health the top rank's renew has dealt.
func lightwellHealing(t *testing.T, agent *HealingPriest) float64 {
	t.Helper()
	renew := agent.GetSpell(core.ActionID{SpellID: 27874})
	if renew == nil {
		t.Fatal("the Lightwell renew spell 27874 is not registered")
	}
	var healing float64
	for _, metrics := range renew.SpellMetrics {
		healing += metrics.TotalHealing
	}
	return healing
}
