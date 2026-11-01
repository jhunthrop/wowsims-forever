package retribution

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// sealOfRighteousnessRank8ProcID is the swing proc of Seal of Righteousness
// rank 8 (20293), the rank a level-60 paladin casts.
const sealOfRighteousnessRank8ProcID = 25713

// sorTestWeaponID is Dal'Rend's Sacred Charge, a 2.8 s one-handed sword
// with no proc of its own.
const sorTestWeaponID = 12940

// sorProcAverage is the average landed Seal of Righteousness swing proc of
// an untalented level-60 paladin with a one-hander with the given spell damage.
func sorProcAverage(t *testing.T, spellDamage float64) float64 {
	t.Helper()
	bonus := stats.Stats{}
	bonus[stats.SpellDamage] = spellDamage
	bonusStats := &proto.UnitStats{Stats: bonus[:]}
	player := core.WithSpec(&proto.Player{
		Class:              proto.Class_ClassPaladin,
		Race:               proto.Race_RaceHuman,
		Level:              60,
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
	metrics := findActionMetrics(result, sealOfRighteousnessRank8ProcID)
	if metrics == nil {
		t.Fatal("the Seal of Righteousness proc never landed")
	}
	var damage float64
	var landed int32
	for _, target := range metrics.Targets {
		damage += target.Damage
		landed += target.Hits + target.Crits
	}
	return damage / float64(landed)
}

// Blizzard's beta development notes (16 October 2026): "Fixed an issue
// causing Seal of Righteousness to gain almost twice as much benefit from
// Spell Damage than intended." The Classic port added the 0.1 coefficient
// to the proc's base damage and let the spell add it again (0.2 in all),
// the double benefit the fix removed. With a one-hander rank 8's base is
// 0.85 x 0.01 x (1786 + 47 x 2) per second of swing (2.8 s here), and spell damage adds 0.1 of itself once, so the multipliers and
// the crit rate cancel in the ratio of the two averages.
func TestSealOfRighteousnessTakesSpellDamageOnce(t *testing.T) {
	const spellDamage = 1000.0
	const coefficient = 0.1
	base := 0.85 * 0.01 * (1786 + 47*2) * 2.8
	without := sorProcAverage(t, 0)
	with := sorProcAverage(t, spellDamage)
	got := (with/without - 1) * base / spellDamage
	if math.Abs(got-coefficient) > 0.005 {
		t.Errorf("Seal of Righteousness gains %.3f of spell damage a proc, want %.3f once", got, coefficient)
	}
}
