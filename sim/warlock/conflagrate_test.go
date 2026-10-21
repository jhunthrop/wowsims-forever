package warlock

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// ConflagrateOnlyTalentsString sets only the Destruction talent
// Conflagrate (position 11 of the Destruction tree's 16 tracked
// nodes).
const ConflagrateOnlyTalentsString = "--0000000000100000"

// TestConflagrateRegistersForeverLowRanks guards against
// spellranks.json's "Conflagrate" chain (build 1.60.1.70009) losing
// its two Forever-only low ranks again: 1293817 (rank 1, level 25) and
// 1293818 (rank 2, level 32), learned well before the classic rank 3
// (17962, level 40) this package used to start at. A level-30 warlock
// with the talent must have rank 1 (1293817) registered and nothing
// higher; a level-60 warlock must still top out at the classic rank 6
// (18932).
func TestConflagrateRegistersForeverLowRanks(t *testing.T) {
	newWarlock := func(level int32) *Warlock {
		player := core.WithSpec(
			&proto.Player{
				Class:         proto.Class_ClassWarlock,
				Race:          proto.Race_RaceOrc,
				Level:         level,
				Equipment:     &proto.EquipmentSpec{},
				Buffs:         core.FullBuffs.Player,
				TalentsString: ConflagrateOnlyTalentsString,
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
		return agent.GetWarlock()
	}

	t.Run("level 30 tops out at the Forever rank 1", func(t *testing.T) {
		warlock := newWarlock(30)
		if len(warlock.Conflagrate) == 0 {
			t.Fatal("level-30 warlock with the Conflagrate talent has no Conflagrate spell registered")
		}
		maxRank := warlock.Conflagrate[len(warlock.Conflagrate)-1]
		if got, want := maxRank.ActionID.SpellID, int32(1293817); got != want {
			t.Errorf("Conflagrate max rank spell ID at level 30 = %d, want %d", got, want)
		}
	})

	t.Run("level 60 tops out at the classic rank 6", func(t *testing.T) {
		warlock := newWarlock(60)
		if len(warlock.Conflagrate) == 0 {
			t.Fatal("level-60 warlock with the Conflagrate talent has no Conflagrate spell registered")
		}
		maxRank := warlock.Conflagrate[len(warlock.Conflagrate)-1]
		if got, want := maxRank.ActionID.SpellID, int32(18932); got != want {
			t.Errorf("Conflagrate max rank spell ID at level 60 = %d, want %d", got, want)
		}
	})
}
