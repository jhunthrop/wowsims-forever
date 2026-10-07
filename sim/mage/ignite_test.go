package mage

import (
	"testing"

	_ "github.com/wowsims/classic/sim/common" // imported to get item effects included.
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// TestIgniteDoesNotCountDamageBonusesTwice pins Forever's 24 September
// 2026 beta note ("Ignite: no longer counts damage bonuses twice"): the
// burn is a share of the crit's finished damage, which already carries
// every attacker multiplier, so the dot must snapshot a multiplier of 1
// rather than apply the mage's damage multipliers a second time on each
// tick, as the vanilla engine did.
func TestIgniteDoesNotCountDamageBonusesTwice(t *testing.T) {
	talentsStr := talentStringWithRank(t, ForeverFrostTalents, "ignite", 5)
	talentsStr = talentStringWithRank(t, talentsStr, "fire_power", 5)

	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassMage,
			Race:               proto.Race_RaceTroll,
			Level:              60,
			Equipment:          &proto.EquipmentSpec{},
			Buffs:              core.FullBuffs.Player,
			TalentsString:      talentsStr,
			DistanceFromTarget: 5,
		},
		PlayerOptions,
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

	agent, ok := sim.Raid.Parties[0].Players[0].(MageAgent)
	if !ok {
		t.Fatal("the raid's first player is not a mage agent")
	}
	built := agent.GetMage()
	if built.Talents.Ignite != 5 || built.igniteTick == nil {
		t.Fatalf("Ignite 5/5 expected with its dot registered; got rank %d", built.Talents.Ignite)
	}

	target := sim.Encounter.TargetUnits[0]
	fireball := built.Fireball[FireballRanks]
	if fireball == nil {
		t.Fatal("level-60 mage has no top-rank Fireball")
	}
	// Force the crit Ignite rolls off, instead of relying on the seed.
	fireball.BonusCritRating += 100 * core.CritRatingPerCritChance
	fireball.ApplyEffects(sim, target, fireball)
	waitForOutcomes(sim)

	dot := built.igniteTick.Dot(target)
	if dot == nil || !dot.IsActive() {
		t.Fatal("a Fireball crit did not apply Ignite")
	}
	if got := dot.SnapshotAttackerMultiplier; got != 1 {
		t.Errorf("Ignite SnapshotAttackerMultiplier = %v, want 1: the crit's damage already carries Fire Power and every other attacker multiplier", got)
	}
	if dot.SnapshotBaseDamage <= 0 {
		t.Errorf("Ignite SnapshotBaseDamage = %v, want > 0", dot.SnapshotBaseDamage)
	}
}
