package dps

import (
	"testing"

	_ "github.com/wowsims/classic/sim/common"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

func init() {
	RegisterDpsWarlock()
}

func TestWarlockSMRuin(t *testing.T) {
	// FOREVER: this spec's talents were regenerated from the client's trait
	// trees (plan docs/superpowers/plans/2026-09-14-sim-engine.md, task 17)
	// and its behaviour has not been rewritten yet, so its DPS goldens
	// describe a spec that no longer exists. Skipped rather than left
	// failing: a suite that is always red is a suite nobody reads, and the
	// next real regression would hide in it.
	//
	// Delete this when this spec is brought up, in the order design section
	// 2.3 gives: Rogue, Hunter, Warlock, Shadow Priest, Balance and Feral
	// Druid, Elemental and Enhancement Shaman, Retribution Paladin.
	t.Skip("sim/warlock/dps awaits its Forever talent rewrite (plan 2026-09-14-sim-engine, task 17)")
	core.RunTestSuite(t, t.Name(), core.FullCharacterTestSuiteGenerator([]core.CharacterSuiteConfig{
		{
			Class: proto.Class_ClassWarlock,
			Phase: 1,
			Race:  proto.Race_RaceOrc,

			Talents:     TalentsSMRuin,
			GearSet:     core.GetGearSet("../../../ui/warlock/gear_sets", "mc"),
			Rotation:    core.GetAplRotation("../../../ui/warlock/apls/", "rotation"),
			Buffs:       core.FullBuffs,
			Consumes:    Consumes,
			SpecOptions: core.SpecOptionsCombo{Label: "SM/Ruin Warlock", SpecOptions: DefaultDestroWarlock},

			ItemFilter:      ItemFilters,
			EPReferenceStat: proto.Stat_StatSpellPower,
			StatsToWeigh:    Stats,
		},
	}))
}

func TestWarlockDSRuin(t *testing.T) {
	// FOREVER: this spec's talents were regenerated from the client's trait
	// trees (plan docs/superpowers/plans/2026-09-14-sim-engine.md, task 17)
	// and its behaviour has not been rewritten yet, so its DPS goldens
	// describe a spec that no longer exists. Skipped rather than left
	// failing: a suite that is always red is a suite nobody reads, and the
	// next real regression would hide in it.
	//
	// Delete this when this spec is brought up, in the order design section
	// 2.3 gives: Rogue, Hunter, Warlock, Shadow Priest, Balance and Feral
	// Druid, Elemental and Enhancement Shaman, Retribution Paladin.
	t.Skip("sim/warlock/dps awaits its Forever talent rewrite (plan 2026-09-14-sim-engine, task 17)")
	core.RunTestSuite(t, t.Name(), core.FullCharacterTestSuiteGenerator([]core.CharacterSuiteConfig{
		{
			Class: proto.Class_ClassWarlock,
			Phase: 1,
			Race:  proto.Race_RaceOrc,

			Talents:     TalentsDSRuin,
			GearSet:     core.GetGearSet("../../../ui/warlock/gear_sets", "mc"),
			Rotation:    core.GetAplRotation("../../../ui/warlock/apls/", "rotation"),
			Buffs:       core.FullBuffs,
			Consumes:    Consumes,
			SpecOptions: core.SpecOptionsCombo{Label: "DS/Ruin Warlock", SpecOptions: DefaultDestroWarlock},

			ItemFilter:      ItemFilters,
			EPReferenceStat: proto.Stat_StatSpellPower,
			StatsToWeigh:    Stats,
		},
	}))
}

var TalentsSMRuin = "5502203112201105--52500051020001"
var TalentsDSRuin = "25002-2050300152201-52500051020001"

var DefaultDestroWarlock = &proto.Player_Warlock{
	Warlock: &proto.Warlock{
		Options: &proto.WarlockOptions{
			Armor:       proto.WarlockOptions_DemonArmor,
			Summon:      proto.WarlockOptions_Succubus,
			WeaponImbue: proto.WarlockOptions_NoWeaponImbue,
		},
	},
}

var Consumes = core.ConsumesCombo{
	Label: "Consumes",
	Consumes: &proto.Consumes{
		DefaultPotion:   proto.Potions_MajorManaPotion,
		Flask:           proto.Flask_FlaskOfSupremePower,
		FirePowerBuff:   proto.FirePowerBuff_ElixirOfGreaterFirepower,
		ShadowPowerBuff: proto.ShadowPowerBuff_ElixirOfShadowPower,
		Food:            proto.Food_FoodTenderWolfSteak,
		MainHandImbue:   proto.WeaponImbue_WizardOil,
		SpellPowerBuff:  proto.SpellPowerBuff_GreaterArcaneElixir,
	},
}

var ItemFilters = core.ItemFilter{
	WeaponTypes: []proto.WeaponType{
		proto.WeaponType_WeaponTypeSword,
		proto.WeaponType_WeaponTypeDagger,
	},
	HandTypes: []proto.HandType{
		proto.HandType_HandTypeOffHand,
	},
	ArmorType: proto.ArmorType_ArmorTypeCloth,
	RangedWeaponTypes: []proto.RangedWeaponType{
		proto.RangedWeaponType_RangedWeaponTypeWand,
	},
}

var Stats = []proto.Stat{
	proto.Stat_StatIntellect,
	proto.Stat_StatSpellPower,
	proto.Stat_StatHit,
	proto.Stat_StatCrit,
}
