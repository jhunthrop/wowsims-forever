package protection

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/paladin"
)

func init() {
	RegisterProtectionPaladin()
}

// TestProtection is the regression suite: the Forever reference build in
// the reference gear tanking the default tank boss, so a change in the
// golden is attributable to the engine. The tank's rotation is the curated
// one (data/curated/apl/paladin-protection.json, synced into
// ui/protection_paladin/apls by `make apl-sync`).
func TestProtection(t *testing.T) {
	core.RunTestSuite(t, t.Name(), core.FullCharacterTestSuiteGenerator([]core.CharacterSuiteConfig{
		{
			Class:      proto.Class_ClassPaladin,
			Phase:      1,
			Race:       proto.Race_RaceHuman,
			OtherRaces: []proto.Race{proto.Race_RaceDwarf},

			Talents:     paladin.ForeverProtectionTalents,
			GearSet:     core.GetGearSet("../../../ui/protection_paladin/gear_sets", harnessGearSet),
			Rotation:    core.GetAplRotation("../../../ui/protection_paladin/apls", harnessRotation),
			Buffs:       core.FullBuffs,
			Consumes:    ForeverConsumes,
			SpecOptions: core.SpecOptionsCombo{Label: "Righteous Fury", SpecOptions: PlayerOptionsRighteousFury},

			IsTank:          true,
			InFrontOfTarget: true,

			ItemFilter:      ItemFilters,
			EPReferenceStat: proto.Stat_StatStamina,
			StatsToWeigh:    Stats,
		},
	}))
}

var ForeverConsumes = core.ConsumesCombo{
	Label: "Forever-Consumes",
	Consumes: &proto.Consumes{
		DefaultPotion: proto.Potions_MajorManaPotion,
		Flask:         proto.Flask_FlaskOfTheTitans,
		ArmorElixir:   proto.ArmorElixir_ElixirOfSuperiorDefense,
		Food:          proto.Food_FoodSmokedDesertDumpling,
	},
}

// PlayerOptionsRighteousFury is what a tanking request sets: Righteous
// Fury on. The seal is the rotation's to cast (Seal of Fury is not a
// PaladinSeal option), so PrimarySeal is left at its default.
var PlayerOptionsRighteousFury = &proto.Player_ProtectionPaladin{
	ProtectionPaladin: &proto.ProtectionPaladin{
		Options: &proto.PaladinOptions{RighteousFury: true},
	},
}

var ItemFilters = core.ItemFilter{
	WeaponTypes: []proto.WeaponType{
		proto.WeaponType_WeaponTypeAxe,
		proto.WeaponType_WeaponTypeSword,
		proto.WeaponType_WeaponTypeMace,
		proto.WeaponType_WeaponTypePolearm,
		proto.WeaponType_WeaponTypeShield,
	},
	RangedWeaponTypes: []proto.RangedWeaponType{
		proto.RangedWeaponType_RangedWeaponTypeLibram,
	},
}

var Stats = []proto.Stat{
	proto.Stat_StatHealth,
	proto.Stat_StatMana,
	proto.Stat_StatStrength,
	proto.Stat_StatStamina,
	proto.Stat_StatAgility,
	proto.Stat_StatIntellect,
	proto.Stat_StatAttackPower,
	proto.Stat_StatHit,
	proto.Stat_StatCrit,
	proto.Stat_StatMeleeHaste,
	proto.Stat_StatSpellPower,
	proto.Stat_StatHolyPower,
	proto.Stat_StatHealingPower,
	proto.Stat_StatArmor,
	proto.Stat_StatBonusArmor,
	proto.Stat_StatDefense,
	proto.Stat_StatDodge,
	proto.Stat_StatParry,
	proto.Stat_StatBlock,
	proto.Stat_StatBlockValue,
	proto.Stat_StatFireResistance,
	proto.Stat_StatNatureResistance,
	proto.Stat_StatShadowResistance,
	proto.Stat_StatFrostResistance,
	proto.Stat_StatArcaneResistance,
}
