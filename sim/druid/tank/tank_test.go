package tank

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/druid"
)

func TestFeralTank(t *testing.T) {
	core.RunTestSuite(t, t.Name(), core.FullCharacterTestSuiteGenerator([]core.CharacterSuiteConfig{
		{
			Class: proto.Class_ClassDruid,
			Phase: 1,
			Race:  proto.Race_RaceTauren,

			Talents:     druid.ForeverBearTalents,
			GearSet:     core.GetGearSet("../../../ui/feral_tank_druid/gear_sets", "forever_l60"),
			Rotation:    core.GetAplRotation("../../../ui/feral_tank_druid/apls", "forever_feral_bear"),
			Buffs:       core.FullBuffs,
			Consumes:    tankConsumes,
			SpecOptions: core.SpecOptionsCombo{Label: "Default", SpecOptions: defaultBearOptions()},

			IsTank:          true,
			InFrontOfTarget: true,

			ItemFilter:      itemFilters,
			EPReferenceStat: proto.Stat_StatStamina,
			StatsToWeigh:    weighedStats,
		},
	}))
}

var tankConsumes = core.ConsumesCombo{
	Label: "Tank-Consumes",
	Consumes: &proto.Consumes{
		Flask:        proto.Flask_FlaskOfTheTitans,
		Food:         proto.Food_FoodSmokedDesertDumpling,
		MiscConsumes: &proto.MiscConsumes{},
	},
}

var itemFilters = core.ItemFilter{
	WeaponTypes: []proto.WeaponType{
		proto.WeaponType_WeaponTypeDagger,
		proto.WeaponType_WeaponTypeMace,
		proto.WeaponType_WeaponTypeOffHand,
		proto.WeaponType_WeaponTypeStaff,
		proto.WeaponType_WeaponTypePolearm,
	},
	ArmorType: proto.ArmorType_ArmorTypeLeather,
	RangedWeaponTypes: []proto.RangedWeaponType{
		proto.RangedWeaponType_RangedWeaponTypeIdol,
	},
}

// weighedStats are the stats a bear is weighed on: avoidance, mitigation,
// health, and what feeds threat.
var weighedStats = []proto.Stat{
	proto.Stat_StatStamina,
	proto.Stat_StatArmor,
	proto.Stat_StatDefense,
	proto.Stat_StatDodge,
	proto.Stat_StatStrength,
	proto.Stat_StatAgility,
	proto.Stat_StatAttackPower,
	proto.Stat_StatCrit,
	proto.Stat_StatHit,
	proto.Stat_StatExpertise,
}
