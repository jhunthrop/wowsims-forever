package tankwarrior

import (
	"testing"

	_ "github.com/wowsims/classic/sim/common" // imported to get item effects included.
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/warrior"
)

func init() {
	RegisterTankWarrior()
}

func TestP1TankWarrior(t *testing.T) {
	core.RunTestSuite(t, t.Name(), core.FullCharacterTestSuiteGenerator([]core.CharacterSuiteConfig{
		{
			Class:      proto.Class_ClassWarrior,
			Phase:      1,
			Race:       proto.Race_RaceOrc,
			OtherRaces: []proto.Race{proto.Race_RaceHuman},

			Talents: P1Talents,
			GearSet: core.GetGearSet(gearSetsDir, harnessGearSet),
			// forever_protection is data/curated/apl/warrior-protection.json
			// in the site, synced by `make apl-sync`; dps_no_reck is the
			// suite's in-file control, a vanilla Fury rotation that tanks
			// as badly as it sounds and is not tuned against.
			Rotation: core.GetAplRotation("../../../ui/tank_warrior/apls", "forever_protection"),
			OtherRotations: []core.RotationCombo{
				core.GetAplRotation("../../../ui/warrior/apls", "dps_no_reck"),
			},
			Buffs:       core.FullBuffs,
			Consumes:    P1Consumes,
			SpecOptions: core.SpecOptionsCombo{Label: "Protection", SpecOptions: PlayerOptionsBasic},

			IsTank:          true,
			InFrontOfTarget: true,

			ItemFilter:      ItemFilters,
			EPReferenceStat: proto.Stat_StatStamina,
			StatsToWeigh:    Stats,
		},
	}))
}

// P1Talents is warrior.ForeverProtectionTalents. The string that stood
// here was vanilla-shaped - three segments of 11, 2 and 17 characters
// against the client's 17, 18 and 18 - so core.FillTalentsProto read
// every talent after the eleventh from the wrong position.
var P1Talents = warrior.ForeverProtectionTalents

var PlayerOptionsBasic = &proto.Player_TankWarrior{
	TankWarrior: &proto.TankWarrior{
		Options: warriorOptions,
	},
}

var warriorOptions = &proto.TankWarrior_Options{
	Shout:        proto.WarriorShout_WarriorShoutCommanding,
	StartingRage: 0,
}

var P1Consumes = core.ConsumesCombo{
	Label: "P1-Consumes",
	Consumes: &proto.Consumes{
		AgilityElixir:     proto.AgilityElixir_ElixirOfTheMongoose,
		AttackPowerBuff:   proto.AttackPowerBuff_JujuMight,
		DefaultPotion:     proto.Potions_MightyRagePotion,
		DragonBreathChili: true,
		Flask:             proto.Flask_FlaskOfTheTitans,
		Food:              proto.Food_FoodSmokedDesertDumpling,
		MainHandImbue:     proto.WeaponImbue_Windfury,
		StrengthBuff:      proto.StrengthBuff_JujuPower,
	},
}

var ItemFilters = core.ItemFilter{
	ArmorType: proto.ArmorType_ArmorTypePlate,

	WeaponTypes: []proto.WeaponType{
		proto.WeaponType_WeaponTypeAxe,
		proto.WeaponType_WeaponTypeSword,
		proto.WeaponType_WeaponTypeMace,
		proto.WeaponType_WeaponTypeDagger,
		proto.WeaponType_WeaponTypeFist,
		proto.WeaponType_WeaponTypeShield,
	},
}

// Stats is what the suite weighs: the survival stats the ranker weighs a
// Protection warrior on (data/curated/specs.json's weight_stats), by the
// engine's stat ids.
var Stats = []proto.Stat{
	proto.Stat_StatStamina,
	proto.Stat_StatArmor,
	proto.Stat_StatDefense,
	proto.Stat_StatDodge,
	proto.Stat_StatParry,
	proto.Stat_StatBlock,
	proto.Stat_StatBlockValue,
	proto.Stat_StatStrength,
	proto.Stat_StatAgility,
	proto.Stat_StatAttackPower,
	proto.Stat_StatHit,
	proto.Stat_StatCrit,
	proto.Stat_StatExpertise,
}
