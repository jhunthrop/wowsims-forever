package retribution

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// sorTestWeaponID is Dal'Rend's Sacred Charge, a 2.8 s one-handed sword
// with no proc of its own; sorTestTwoHanderID is the Arcanite Reaper, a
// 3.8 s two-handed axe with none either.
const (
	sorTestWeaponID    = 12940
	sorTestTwoHanderID = 12784
)

// sorProcAverage is the average landed Seal of Righteousness swing proc
// (procID) of an untalented paladin of the given level, with the given
// weapon and spell damage. The rotation casts the level's top rank.
func sorProcAverage(t *testing.T, level int32, procID int32, weaponID int32, spellDamage float64) float64 {
	t.Helper()
	bonus := stats.Stats{}
	bonus[stats.SpellDamage] = spellDamage
	bonusStats := &proto.UnitStats{Stats: bonus[:]}
	player := core.WithSpec(&proto.Player{
		Class:              proto.Class_ClassPaladin,
		Race:               proto.Race_RaceHuman,
		Level:              level,
		Equipment:          &proto.EquipmentSpec{Items: []*proto.ItemSpec{{}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {Id: weaponID}}},
		BonusStats:         bonusStats,
		Rotation:           core.APLRotationFromJsonString(`{"type": "TypeAPL", "priorityList": [{"action": {"castPaladinPrimarySeal": {}}}]}`),
		DistanceFromTarget: 5,
	}, PlayerOptionsSealofRighteousness)
	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})
	result := core.RunRaidSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 120, Targets: []*proto.Target{core.NewDefaultTarget()}},
		SimOptions: &proto.SimOptions{Iterations: 200, RandomSeed: 1, IsTest: true},
	})
	if result.Error != nil {
		t.Fatalf("sim error: %s", result.Error.Message)
	}
	metrics := findActionMetrics(result, procID)
	if metrics == nil {
		t.Fatalf("the Seal of Righteousness proc %d never landed", procID)
	}
	var damage float64
	var landed int32
	for _, target := range metrics.Targets {
		damage += target.Damage
		landed += target.Hits + target.Crits
	}
	return damage / float64(landed)
}

// Seal of Righteousness's spell damage term is the one the 70755 client's
// own tooltip states, the same for every rank: 0.0765 of spell damage with
// a one-hander and 0.12 with a two-hander (SpellDescriptionVariables 1023,
// `$1h=0.0765*$SPH` and `$2h=0.12*$SPH`, mapped to all nine Seal ids
// 20154 to 20293 by SpellXDescriptionVariables). Blizzard wrote that
// formula in build 70425, the build whose 16 October beta note fixed the
// Seal "gaining almost twice as much benefit from Spell Damage than
// intended" by zeroing the swing procs' own coefficients (25742 to 25713,
// 0.1 before): the script's term plus the proc's generic 0.1 was the
// double (0.12 + 0.1 on a two-hander). The term is the Judgement spell's
// dummy effect coefficient halved and scaled by the hand (0.2 x 1.2 / 2 =
// 0.12 on rank 8's 20286), the effect whose amount the tooltip reads
// ($20286s3), so rank 8 scales like every rank: the 0 on rank 8's aura
// effect (20293) is Era's and never carried the Seal's scaling (Era's
// rank 8 scaled with spell damage, and that column was 0 there too).
//
// A rank's base is 0.85 (one-hander) or 1.2 (two-hander) x 0.01 x (amount
// + per level x levels) per second of swing, and spell damage adds the
// coefficient of itself once, so the multipliers and the crit rate cancel
// in the ratio of the two averages.
func TestSealOfRighteousnessCoefficientFollowsTheClientPerRank(t *testing.T) {
	const spellDamage = 1000.0
	cases := []struct {
		name   string
		level  int32
		procID int32
		base   float64 // the rank's amount at this level
		weapon int32
		speed  float64
		hand   float64
		want   float64
	}{
		{"rank 3 at 25, one-hander", 25, 25739, 352 + 23*6, sorTestWeaponID, 2.8, 0.85, 0.0765},
		{"rank 7 at 57, one-hander", 57, 25735, 1407 + 47*6, sorTestWeaponID, 2.8, 0.85, 0.0765},
		{"rank 8 at 60, one-hander", 60, 25713, 1786 + 47*2, sorTestWeaponID, 2.8, 0.85, 0.0765},
		{"rank 7 at 57, two-hander", 57, 25735, 1407 + 47*6, sorTestTwoHanderID, 3.8, 1.2, 0.12},
		{"rank 8 at 60, two-hander", 60, 25713, 1786 + 47*2, sorTestTwoHanderID, 3.8, 1.2, 0.12},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			base := c.hand * 0.01 * c.base * c.speed
			without := sorProcAverage(t, c.level, c.procID, c.weapon, 0)
			with := sorProcAverage(t, c.level, c.procID, c.weapon, spellDamage)
			got := (with/without - 1) * base / spellDamage
			if math.Abs(got-c.want) > 0.005 {
				t.Errorf("Seal of Righteousness gains %.3f of spell damage a proc, want %.3f", got, c.want)
			}
		})
	}
}
