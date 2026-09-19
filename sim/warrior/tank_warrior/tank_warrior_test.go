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
	// FOREVER: task 11 of plan 2026-09-14-sim-engine rewrote the Fury
	// spec's talent behaviour and un-skipped sim/warrior/dps_warrior. It
	// did NOT rewrite Protection's: Improved Revenge went from a stun
	// chance to +60% Revenge damage, Improved Bloodrage from flat rage
	// to +50% of Bloodrage's own, Improved Shield Wall from duration to
	// cooldown, Defiance from 3% threat a point to 5%, and Master of
	// Defense, Vanguard, Improved Shield Bash, Focused Rage and Bastion
	// are new. Un-skipping this suite now would bless vanilla's numbers
	// as Forever's, so it waits for the warrior-protection spec's own
	// task; the reference build below is already the client's shape, so
	// that task is a behaviour rewrite and a golden run, not a hunt for
	// why the talent string no longer parses.
	t.Skip("sim/warrior/tank_warrior awaits the warrior-protection spec's talent rewrite; task 11 covered Fury only")
	core.RunTestSuite(t, t.Name(), core.FullCharacterTestSuiteGenerator([]core.CharacterSuiteConfig{
		{
			Class:      proto.Class_ClassWarrior,
			Phase:      1,
			Race:       proto.Race_RaceOrc,
			OtherRaces: []proto.Race{proto.Race_RaceHuman},

			Talents:  P1Talents,
			GearSet:  core.GetGearSet("../../../ui/tank_warrior/gear_sets", "p0.bis"),
			Rotation: core.GetAplRotation("../../../ui/warrior/apls", "dps_reck"),
			OtherRotations: []core.RotationCombo{
				core.GetAplRotation("../../../ui/warrior/apls", "dps_no_reck"),
			},
			Buffs:       core.FullBuffs,
			Consumes:    P1Consumes,
			SpecOptions: core.SpecOptionsCombo{Label: "Protection", SpecOptions: PlayerOptionsBasic},

			ItemFilter:      ItemFilters,
			EPReferenceStat: proto.Stat_StatAttackPower,
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

var Stats = []proto.Stat{
	proto.Stat_StatStrength,
	proto.Stat_StatAttackPower,
	proto.Stat_StatArmor,
	proto.Stat_StatDodge,
	proto.Stat_StatParry,
	proto.Stat_StatBlockValue,
	proto.Stat_StatDefense,
}
