// Package healsim is the shared harness the healing specs' tests run
// under: one healer, five fake raid members taking damage from a
// RaidDamageModel, and a summary of what the healer did about it.
//
// The model below is a TEST profile. It exists so every healer's tests
// face the same fight; the numbers the Forever Sixty site publishes come
// from its own curated profile (data/curated/heal-profile.json), which
// reaches the engine through the request layer.
package healsim

import (
	"fmt"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// The fake raid's layout. The healer is player 0 of party 0; the engine
// fills the healer's party with the first four fake members (players 1
// to 4) and the fifth, the tank, lands in party 1 (player 5).
const (
	HealerIndex = 0
	TankIndex   = 5
	// FakeMembers is how many fake raid members a request adds.
	FakeMembers = 5
)

// MemberIndices are the raid members pulses land on (the healer's party).
var MemberIndices = []int32{1, 2, 3, 4}

// TestProfile is the damage model the fork's healing tests use.
func TestProfile() *proto.RaidDamageModel {
	return &proto.RaidDamageModel{
		Profile:              "fork-test-profile",
		TankHealth:           9000,
		MemberHealth:         5000,
		TankHitDamage:        900,
		TankSwingSeconds:     2,
		DamageSpread:         0.25,
		PulseDamage:          400,
		PulseIntervalSeconds: 5,
		PulseMembers:         3,
	}
}

// Request builds a sim of healer against the model for duration seconds.
func Request(healer *proto.Player, model *proto.RaidDamageModel, duration float64, iterations int32) *proto.RaidSimRequest {
	raid := core.SinglePlayerRaidProto(healer, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})
	raid.Parties = append(raid.Parties, &proto.Party{})
	raid.TargetDummies = FakeMembers
	raid.RaidDamageModel = model
	return &proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: duration, Targets: []*proto.Target{core.NewDefaultTarget()}},
		SimOptions: &proto.SimOptions{Iterations: iterations, IsTest: true, RandomSeed: 1},
	}
}

// Summary is what a healing run reports for the healer.
type Summary struct {
	// EffectiveHPS is healing that landed per second; RawHPS includes overheal.
	EffectiveHPS, RawHPS float64
	// OverhealPct is the share of raw healing that overhealed, 0 to 1.
	OverhealPct float64
	// TimeToOOMSeconds is when the healer first ran out of mana (an hour
	// when it never did, the engine's own convention).
	TimeToOOMSeconds float64
	// Casts counts casts by spell id.
	Casts map[int32]int32
}

// Run runs the request and summarises player 0.
func Run(req *proto.RaidSimRequest) (Summary, error) {
	result := core.RunRaidSim(req)
	if result.Error != nil {
		return Summary{}, fmt.Errorf("healsim: %s", result.Error.Message)
	}
	player := result.RaidMetrics.Parties[0].Players[HealerIndex]
	summary := Summary{
		EffectiveHPS:     player.EffectiveHps.Avg,
		RawHPS:           player.Hps.Avg,
		TimeToOOMSeconds: player.Tto.Avg,
		Casts:            map[int32]int32{},
	}
	if summary.RawHPS > 0 {
		summary.OverhealPct = 1 - summary.EffectiveHPS/summary.RawHPS
	}
	for _, action := range player.Actions {
		if id := action.Id.GetSpellId(); id != 0 {
			for _, target := range action.Targets {
				summary.Casts[id] += target.Casts
			}
		}
	}
	return summary, nil
}
