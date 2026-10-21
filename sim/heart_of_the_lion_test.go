package sim

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// Heart of the Lion (client spells 409580 and 409583): the area buff is +10%
// to every stat and 40 + 4 per level above 1 (cap 60) attack power and
// ranged attack power; the hunter's own cast adds a further +10% to the
// hunter. See design/reviews/2026-10-08-heart-of-the-lion.md on the site.

const (
	heartOfTheLionStatTolerance = 1e-9
	// ranged attack power per point of agility for a hunter.
	hunterRangedAttackPowerPerAgility = 2.0
)

var allBaseStats = []stats.Stat{stats.Stamina, stats.Agility, stats.Strength, stats.Intellect, stats.Spirit}

func priestPlayer(level int32) *proto.Player {
	return core.WithSpec(&proto.Player{
		Class: proto.Class_ClassPriest, Race: proto.Race_RaceUndead, Level: level,
		Equipment: &proto.EquipmentSpec{}, Rotation: &proto.APLRotation{},
	}, &proto.Player_ShadowPriest{ShadowPriest: &proto.ShadowPriest{Options: &proto.ShadowPriest_Options{}}})
}

func hunterPlayer(level int32) *proto.Player {
	return core.WithSpec(&proto.Player{
		Class: proto.Class_ClassHunter, Race: proto.Race_RaceOrc, Level: level,
		Equipment: &proto.EquipmentSpec{}, Rotation: &proto.APLRotation{},
	}, &proto.Player_Hunter{Hunter: &proto.Hunter{Options: &proto.Hunter_Options{}}})
}

// firstPlayerStats computes the first player's base and final stats for a
// raid of the given players under the given raid buffs.
func firstPlayerStats(t *testing.T, raidBuffs *proto.RaidBuffs, players ...*proto.Player) (base, final stats.Stats) {
	t.Helper()
	raid := core.SinglePlayerRaidProto(players[0], &proto.PartyBuffs{}, raidBuffs, &proto.Debuffs{})
	raid.Parties[0].Players = append(raid.Parties[0].Players, players[1:]...)
	result := core.ComputeStats(&proto.ComputeStatsRequest{Raid: raid})
	player := result.RaidStats.Parties[0].Players[0]
	return stats.FromFloatArray(player.BaseStats.Stats), stats.FromFloatArray(player.FinalStats.Stats)
}

func assertStatRatio(t *testing.T, label string, base, final stats.Stats, want float64) {
	t.Helper()
	for _, stat := range allBaseStats {
		if got := final[stat] / base[stat]; math.Abs(got-want) > heartOfTheLionStatTolerance {
			t.Errorf("%s: %s is %.6f of its base, want %.2f", label, stat.StatName(), got, want)
		}
	}
}

// The raid buff stands for another hunter's aura: +10% on a non-hunter and
// the client's level-scaled attack power on the ranged side (a priest has
// no dependency on it), 276 at 60 and 156 at 30.
func TestHeartOfTheLionRaidBuffReachesAnotherClass(t *testing.T) {
	cases := []struct {
		level           int32
		wantAttackPower float64
	}{{60, 276}, {30, 156}, {1, 40}}
	for _, c := range cases {
		_, bare := firstPlayerStats(t, &proto.RaidBuffs{}, priestPlayer(c.level))
		_, buffed := firstPlayerStats(t, &proto.RaidBuffs{HeartOfTheLion: true}, priestPlayer(c.level))
		assertStatRatio(t, "priest with the raid buff", bare, buffed, 1.10)
		if got := buffed[stats.RangedAttackPower] - bare[stats.RangedAttackPower]; got != c.wantAttackPower {
			t.Errorf("level %d: ranged attack power gain = %v, want %v", c.level, got, c.wantAttackPower)
		}
		// A priest turns strength into melee attack power one for one.
		meleeGain := buffed[stats.AttackPower] - bare[stats.AttackPower] - (buffed[stats.Strength] - bare[stats.Strength])
		if meleeGain != c.wantAttackPower {
			t.Errorf("level %d: melee attack power gain beyond strength = %v, want %v", c.level, meleeGain, c.wantAttackPower)
		}
	}
}

// Above the spell's level 60 the amount stops growing.
func TestHeartOfTheLionAttackPowerStopsAtSixty(t *testing.T) {
	if got := core.HeartOfTheLionRanks.At(61); got != 276 {
		t.Errorf("attack power at 61 = %v, want 276", got)
	}
}

// A hunter always casts it on himself: +10% from the spell and +10% from
// its own area buff, which multiply (1.21), whether or not the preset names
// the raid buff, and another hunter's copy adds nothing.
func TestHeartOfTheLionHunterReadsBothAndNeverDoubles(t *testing.T) {
	for _, level := range []int32{60, 30} {
		base, alone := firstPlayerStats(t, &proto.RaidBuffs{}, hunterPlayer(level))
		assertStatRatio(t, "hunter alone", base, alone, 1.21)

		_, withRaidBuff := firstPlayerStats(t, &proto.RaidBuffs{HeartOfTheLion: true}, hunterPlayer(level))
		_, withSecondHunter := firstPlayerStats(t, &proto.RaidBuffs{HeartOfTheLion: true}, hunterPlayer(level), hunterPlayer(level))
		if withRaidBuff != alone {
			t.Errorf("level %d: the raid buff on top of the hunter's own cast changed his stats:\n got %v\nwant %v", level, withRaidBuff, alone)
		}
		if withSecondHunter != alone {
			t.Errorf("level %d: a second hunter changed the first hunter's stats", level)
		}

		wantAttackPower := core.HeartOfTheLionRanks.At(int(level))
		agilityGain := alone[stats.Agility] - base[stats.Agility]
		gain := alone[stats.RangedAttackPower] - base[stats.RangedAttackPower] - hunterRangedAttackPowerPerAgility*agilityGain
		if math.Abs(gain-wantAttackPower) > heartOfTheLionStatTolerance {
			t.Errorf("level %d: ranged attack power gain beyond agility = %v, want %v", level, gain, wantAttackPower)
		}
	}
}
