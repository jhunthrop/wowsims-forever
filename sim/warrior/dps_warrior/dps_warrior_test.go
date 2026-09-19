package dpswarrior

import (
	"testing"

	_ "github.com/wowsims/classic/sim/common" // imported to get item effects included.
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/warrior"
)

func init() {
	RegisterDpsWarrior()
}

func TestP1DPSWarrior(t *testing.T) {
	core.RunTestSuite(t, t.Name(), core.FullCharacterTestSuiteGenerator([]core.CharacterSuiteConfig{
		{
			Class:      proto.Class_ClassWarrior,
			Phase:      1,
			Race:       proto.Race_RaceOrc,
			OtherRaces: []proto.Race{proto.Race_RaceHuman},

			Talents:     P1Talents,
			GearSet:     core.GetGearSet("../../../ui/warrior/gear_sets", "phase_1"),
			Rotation:    core.GetAplRotation("../../../ui/warrior/apls", "forever_fury"),
			Buffs:       core.FullBuffs,
			Consumes:    P1Consumes,
			SpecOptions: core.SpecOptionsCombo{Label: "DPS", SpecOptions: PlayerOptionsFury},

			ItemFilter:      ItemFilters,
			EPReferenceStat: proto.Stat_StatAttackPower,
			StatsToWeigh:    Stats,
		},
	}))
}

// P1Talents is warrior.ForeverFuryTalents, not a string of its own: the
// reference build belongs beside the tree it is written against, and two
// copies of a talent string drift the first time a tier moves.
//
// The old vanilla-shaped string that stood here ("30305001302-...")
// could not be parsed against the client's 17/18/18 trees at all - its
// first segment was 11 characters - so every talent after the first
// eleven was read from the wrong position.
var P1Talents = warrior.ForeverFuryTalents

var P1Consumes = core.ConsumesCombo{
	Label: "P1-Consumes",
	Consumes: &proto.Consumes{
		AgilityElixir:     proto.AgilityElixir_ElixirOfTheMongoose,
		AttackPowerBuff:   proto.AttackPowerBuff_JujuMight,
		DefaultPotion:     proto.Potions_MightyRagePotion,
		DragonBreathChili: true,
		Food:              proto.Food_FoodSmokedDesertDumpling,
		MainHandImbue:     proto.WeaponImbue_Windfury,
		OffHandImbue:      proto.WeaponImbue_ElementalSharpeningStone,
		StrengthBuff:      proto.StrengthBuff_JujuPower,
	},
}

var PlayerOptionsArms = &proto.Player_Warrior{
	Warrior: &proto.Warrior{
		Options: warriorOptions,
	},
}

var PlayerOptionsFury = &proto.Player_Warrior{
	Warrior: &proto.Warrior{
		Options: warriorOptions,
	},
}

var warriorOptions = &proto.Warrior_Options{
	StartingRage: 50,
	Shout:        proto.WarriorShout_WarriorShoutBattle,
}

var ItemFilters = core.ItemFilter{
	ArmorType: proto.ArmorType_ArmorTypePlate,

	WeaponTypes: []proto.WeaponType{
		proto.WeaponType_WeaponTypeAxe,
		proto.WeaponType_WeaponTypeSword,
		proto.WeaponType_WeaponTypeMace,
		proto.WeaponType_WeaponTypeDagger,
		proto.WeaponType_WeaponTypeFist,
	},
}

var Stats = []proto.Stat{
	proto.Stat_StatStrength,
	proto.Stat_StatAgility,
	proto.Stat_StatAttackPower,
	proto.Stat_StatCrit,
	proto.Stat_StatHit,
}
