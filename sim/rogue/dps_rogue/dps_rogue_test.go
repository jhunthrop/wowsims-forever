package dpsrogue

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

func init() {
	RegisterDpsRogue()
}

func TestCombatSinisterStrike(t *testing.T) {
	core.SkipAwaitingForeverTalentRewrite(t, "sim/rogue/dps_rogue")
	core.RunTestSuite(t, t.Name(), core.FullCharacterTestSuiteGenerator([]core.CharacterSuiteConfig{
		{
			Class:      proto.Class_ClassRogue,
			Race:       proto.Race_RaceHuman,
			OtherRaces: []proto.Race{proto.Race_RaceOrc},

			Talents:     CombatSwordsTalents,
			GearSet:     core.GetGearSet("../../../ui/rogue/gear_sets", "combat_sinister_strike_prebis"),
			Rotation:    core.GetAplRotation("../../../ui/rogue/apls", "combat_sinister_strike"),
			Buffs:       core.FullBuffs,
			Consumes:    Phase1Consumes,
			Phase:       5,
			SpecOptions: core.SpecOptionsCombo{Label: "No Poisons", SpecOptions: DefaultRogue},

			ItemFilter:      ItemFilters,
			EPReferenceStat: proto.Stat_StatAttackPower,
			StatsToWeigh:    Stats,
		},
	}))
}

func TestCombatDaggers(t *testing.T) {
	core.SkipAwaitingForeverTalentRewrite(t, "sim/rogue/dps_rogue")
	core.RunTestSuite(t, t.Name(), core.FullCharacterTestSuiteGenerator([]core.CharacterSuiteConfig{
		{
			Class:      proto.Class_ClassRogue,
			Race:       proto.Race_RaceHuman,
			OtherRaces: []proto.Race{proto.Race_RaceOrc},

			Talents:     CombatDaggersTalents,
			GearSet:     core.GetGearSet("../../../ui/rogue/gear_sets", "combat_backstab_prebis"),
			Rotation:    core.GetAplRotation("../../../ui/rogue/apls", "combat_backstab"),
			Buffs:       core.FullBuffs,
			Consumes:    Phase1Consumes,
			Phase:       5,
			SpecOptions: core.SpecOptionsCombo{Label: "No Poisons", SpecOptions: DefaultRogue},

			ItemFilter:      ItemFilters,
			EPReferenceStat: proto.Stat_StatAttackPower,
			StatsToWeigh:    Stats,
		},
	}))
}

// The Combat builds in the 70755 trees (talents_auto_gen.go's field order):
// Malice 5, Ruthlessness 3, Improved Slice and Dice 3, Lethality 2,
// Relentless Strikes 5; Coup de Grace 3, Efficient Strikes 2, Precision 3,
// Flawless Execution, Weapon Expertise 2, Aggression 3, Dual Wield
// Specialization 2, Blade Flurry, Hack and Slash 5, Flash of Steel 5,
// Adrenaline Rush, with Lightning Reflexes 5 (swords) or Puncturing Wounds
// 3 and Lightning Reflexes 2 (daggers).
var CombatSwordsTalents = "0053030020005-3250030000012321551"
var CombatDaggersTalents = "0053030020005-3223030000012321551"

var DefaultRogue = &proto.Player_Rogue{
	Rogue: &proto.Rogue{
		Options: &proto.RogueOptions{},
	},
}

var ItemFilters = core.ItemFilter{
	ArmorType: proto.ArmorType_ArmorTypeLeather,
	WeaponTypes: []proto.WeaponType{
		proto.WeaponType_WeaponTypeDagger,
		proto.WeaponType_WeaponTypeFist,
		proto.WeaponType_WeaponTypeSword,
		proto.WeaponType_WeaponTypeMace,
	},
	RangedWeaponTypes: []proto.RangedWeaponType{
		proto.RangedWeaponType_RangedWeaponTypeBow,
		proto.RangedWeaponType_RangedWeaponTypeCrossbow,
		proto.RangedWeaponType_RangedWeaponTypeGun,
	},
}

var Stats = []proto.Stat{
	proto.Stat_StatAttackPower,
	proto.Stat_StatAgility,
	proto.Stat_StatStrength,
	proto.Stat_StatHit,
	proto.Stat_StatCrit,
}

var Phase1Consumes = core.ConsumesCombo{
	Label: "P1-Consumes",
	Consumes: &proto.Consumes{
		AgilityElixir:   proto.AgilityElixir_ElixirOfTheMongoose,
		MainHandImbue:   proto.WeaponImbue_Windfury,
		OffHandImbue:    proto.WeaponImbue_InstantPoison,
		StrengthBuff:    proto.StrengthBuff_JujuPower,
		AttackPowerBuff: proto.AttackPowerBuff_JujuMight,
	},
}
