package warlock

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// IncinerateOnlyTalentsString sets only the Destruction talent Incinerate
// (the last field in the Destruction tree, 16 nodes).
const IncinerateOnlyTalentsString = "--0000000000000001"

// TestIncinerateLevel60HasMaxRankAndDealsDamage casts Incinerate on a
// level-60 warlock with the talent: registerIncinerateSpell is gated on
// warlock.Talents.Incinerate and, per spellranks.json, a level-60
// warlock's castable rank is 1293813 (rank 3).
func TestIncinerateLevel60HasMaxRankAndDealsDamage(t *testing.T) {
	player := core.WithSpec(
		&proto.Player{
			Class:         proto.Class_ClassWarlock,
			Race:          proto.Race_RaceOrc,
			Level:         60,
			Equipment:     &proto.EquipmentSpec{},
			Buffs:         core.FullBuffs.Player,
			TalentsString: IncinerateOnlyTalentsString,
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

	if len(built.Incinerate) == 0 {
		t.Fatal("level-60 warlock with the Incinerate talent has no Incinerate spell registered")
	}
	maxRank := built.Incinerate[len(built.Incinerate)-1]
	if got, want := maxRank.ActionID.SpellID, int32(1293813); got != want {
		t.Errorf("Incinerate max rank spell ID = %d, want %d", got, want)
	}
	if got, want := maxRank.Rank, 3; got != want {
		t.Errorf("Incinerate max rank = %d, want %d", got, want)
	}

	target := sim.Encounter.TargetUnits[0]

	maxRank.ApplyEffects(sim, target, maxRank)

	metrics := maxRank.SpellMetrics[target.UnitIndex]
	if metrics.Hits+metrics.Crits == 0 {
		t.Fatalf("Incinerate outcome was neither a hit nor a crit (misses=%d)", metrics.Misses)
	}
	if metrics.TotalDamage <= 0 {
		t.Errorf("Incinerate dealt %v damage, want > 0", metrics.TotalDamage)
	}
}
