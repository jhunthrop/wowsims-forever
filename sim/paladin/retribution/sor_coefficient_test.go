package retribution

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// sorTestWeaponID is Dal'Rend's Sacred Charge, a 2.8 s one-handed sword
// with no proc of its own.
const sorTestWeaponID = 12940

// sorProcAverage is the average landed Seal of Righteousness swing proc
// (procID) of an untalented paladin of the given level, with a one-hander
// and the given spell damage. The rotation casts the level's top rank.
func sorProcAverage(t *testing.T, level int32, procID int32, spellDamage float64) float64 {
	t.Helper()
	bonus := stats.Stats{}
	bonus[stats.SpellDamage] = spellDamage
	bonusStats := &proto.UnitStats{Stats: bonus[:]}
	player := core.WithSpec(&proto.Player{
		Class:              proto.Class_ClassPaladin,
		Race:               proto.Race_RaceHuman,
		Level:              level,
		Equipment:          &proto.EquipmentSpec{Items: []*proto.ItemSpec{{}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {Id: sorTestWeaponID}}},
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

// The 70755 client puts Seal of Righteousness's spell damage coefficient on
// each rank's Seal aura (SpellEffect EffectBonusCoefficient, effect 0):
// 0.1 on ranks 1 to 7 (20154/21084, 20287 to 20292), 0 on rank 8 (20293).
// Every swing proc (25742, 25740 to 25735, 25713) carries 0 since 70425.
// Before 70425 the procs carried 0.1 as well and the server added both,
// which is the double benefit Blizzard's 16 October beta note fixed ("gain
// almost twice as much benefit from Spell Damage than intended") by
// zeroing the procs. That left rank 8, whose aura was 0 already (as in
// Era), with nothing. Era's 0.029/0.063/0.093 on ranks 1 to 3 are gone:
// Forever raised them to 0.1 from its first build (69893).
//
// With a one-hander a rank's base is 0.85 x 0.01 x (amount + per level x
// levels) per second of swing (2.8 s here), and spell damage adds the
// coefficient of itself once, so the multipliers and the crit rate cancel
// in the ratio of the two averages.
func TestSealOfRighteousnessCoefficientFollowsTheClientPerRank(t *testing.T) {
	const spellDamage = 1000.0
	cases := []struct {
		name   string
		level  int32
		procID int32
		base   float64 // the rank's amount at this level
		want   float64
	}{
		{"rank 3 at 25", 25, 25739, 352 + 23*6, 0.1},
		{"rank 7 at 57", 57, 25735, 1407 + 47*6, 0.1},
		{"rank 8 at 60", 60, 25713, 1786 + 47*2, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			base := 0.85 * 0.01 * c.base * 2.8
			without := sorProcAverage(t, c.level, c.procID, 0)
			with := sorProcAverage(t, c.level, c.procID, spellDamage)
			got := (with/without - 1) * base / spellDamage
			if math.Abs(got-c.want) > 0.005 {
				t.Errorf("Seal of Righteousness gains %.3f of spell damage a proc, want %.3f", got, c.want)
			}
		})
	}
}
