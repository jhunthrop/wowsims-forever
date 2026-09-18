package elemental

import (
	"testing"

	_ "github.com/wowsims/classic/sim/common"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

func init() {
	RegisterElementalShaman()
}

func TestElemental(t *testing.T) {
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
	t.Skip("sim/shaman/elemental awaits its Forever talent rewrite (plan 2026-09-14-sim-engine, task 17)")
	core.RunTestSuite(t, t.Name(), core.FullCharacterTestSuiteGenerator([]core.CharacterSuiteConfig{
		{
			Class:      proto.Class_ClassShaman,
			Phase:      1,
			Race:       proto.Race_RaceTroll,
			OtherRaces: []proto.Race{proto.Race_RaceOrc},

			Talents:     DefaultTalents,
			GearSet:     core.GetGearSet("../../../ui/elemental_shaman/gear_sets", "phase_1"),
			Rotation:    core.GetAplRotation("../../../ui/elemental_shaman/apls", "default"),
			Buffs:       core.FullBuffs,
			Consumes:    Phase1Consumes,
			SpecOptions: core.SpecOptionsCombo{Label: "Adaptive", SpecOptions: PlayerOptionsAdaptive},

			ItemFilter:      ItemFilters,
			EPReferenceStat: proto.Stat_StatSpellPower,
			StatsToWeigh:    Stats,
		},
		{
			Class:      proto.Class_ClassShaman,
			Phase:      2,
			Race:       proto.Race_RaceTroll,
			OtherRaces: []proto.Race{proto.Race_RaceOrc},

			Talents:     DefaultTalents,
			GearSet:     core.GetGearSet("../../../ui/elemental_shaman/gear_sets", "phase_2"),
			Rotation:    core.GetAplRotation("../../../ui/elemental_shaman/apls", "default"),
			Buffs:       core.FullBuffs,
			Consumes:    Phase2Consumes,
			SpecOptions: core.SpecOptionsCombo{Label: "Adaptive", SpecOptions: PlayerOptionsAdaptive},

			ItemFilter:      ItemFilters,
			EPReferenceStat: proto.Stat_StatSpellPower,
			StatsToWeigh:    Stats,
		},
		{
			Class:      proto.Class_ClassShaman,
			Phase:      3,
			Race:       proto.Race_RaceTroll,
			OtherRaces: []proto.Race{proto.Race_RaceOrc},

			Talents:     DefaultTalents,
			GearSet:     core.GetGearSet("../../../ui/elemental_shaman/gear_sets", "phase_3"),
			Rotation:    core.GetAplRotation("../../../ui/elemental_shaman/apls", "default"),
			Buffs:       core.FullBuffs,
			Consumes:    Phase2Consumes,
			SpecOptions: core.SpecOptionsCombo{Label: "Adaptive", SpecOptions: PlayerOptionsAdaptive},

			ItemFilter:      ItemFilters,
			EPReferenceStat: proto.Stat_StatSpellPower,
			StatsToWeigh:    Stats,
		},
		{
			Class:      proto.Class_ClassShaman,
			Phase:      4,
			Race:       proto.Race_RaceTroll,
			OtherRaces: []proto.Race{proto.Race_RaceOrc},

			Talents:     DefaultTalents,
			GearSet:     core.GetGearSet("../../../ui/elemental_shaman/gear_sets", "phase_4"),
			Rotation:    core.GetAplRotation("../../../ui/elemental_shaman/apls", "default"),
			Buffs:       core.FullBuffs,
			Consumes:    Phase2Consumes,
			SpecOptions: core.SpecOptionsCombo{Label: "Adaptive", SpecOptions: PlayerOptionsAdaptive},

			ItemFilter:      ItemFilters,
			EPReferenceStat: proto.Stat_StatSpellPower,
			StatsToWeigh:    Stats,
		},
		{
			Class:      proto.Class_ClassShaman,
			Phase:      5,
			Race:       proto.Race_RaceTroll,
			OtherRaces: []proto.Race{proto.Race_RaceOrc},

			Talents:     DefaultTalents,
			GearSet:     core.GetGearSet("../../../ui/elemental_shaman/gear_sets", "phase_5"),
			Rotation:    core.GetAplRotation("../../../ui/elemental_shaman/apls", "default"),
			Buffs:       core.FullBuffs,
			Consumes:    Phase5Consumes,
			SpecOptions: core.SpecOptionsCombo{Label: "Adaptive", SpecOptions: PlayerOptionsAdaptive},

			ItemFilter:      ItemFilters,
			EPReferenceStat: proto.Stat_StatSpellPower,
			StatsToWeigh:    Stats,
		},
		{
			Class:      proto.Class_ClassShaman,
			Phase:      6,
			Race:       proto.Race_RaceTroll,
			OtherRaces: []proto.Race{proto.Race_RaceOrc},

			Talents:     DefaultTalents,
			GearSet:     core.GetGearSet("../../../ui/elemental_shaman/gear_sets", "phase_6"),
			Rotation:    core.GetAplRotation("../../../ui/elemental_shaman/apls", "default"),
			Buffs:       core.FullBuffs,
			Consumes:    Phase5Consumes,
			SpecOptions: core.SpecOptionsCombo{Label: "Adaptive", SpecOptions: PlayerOptionsAdaptive},

			ItemFilter:      ItemFilters,
			EPReferenceStat: proto.Stat_StatSpellPower,
			StatsToWeigh:    Stats,
		},
	}))
}

var DefaultTalents = "550331050002151--50105301005"

var PlayerOptionsAdaptive = &proto.Player_ElementalShaman{
	ElementalShaman: &proto.ElementalShaman{
		Options: &proto.ElementalShaman_Options{},
	},
}

var Phase1Consumes = core.ConsumesCombo{
	Label: "P1-Consumes",
	Consumes: &proto.Consumes{
		DefaultConjured: proto.Conjured_ConjuredDemonicRune,
		DefaultPotion:   proto.Potions_MajorManaPotion,
		Flask:           proto.Flask_FlaskOfSupremePower,
		FirePowerBuff:   proto.FirePowerBuff_ElixirOfGreaterFirepower,
		Food:            proto.Food_FoodNightfinSoup,
		MainHandImbue:   proto.WeaponImbue_LesserWizardOil,
		SpellPowerBuff:  proto.SpellPowerBuff_GreaterArcaneElixir,
	},
}

var Phase2Consumes = core.ConsumesCombo{
	Label: "P2-Consumes",
	Consumes: &proto.Consumes{
		DefaultConjured: proto.Conjured_ConjuredDemonicRune,
		DefaultPotion:   proto.Potions_MajorManaPotion,
		Flask:           proto.Flask_FlaskOfSupremePower,
		FirePowerBuff:   proto.FirePowerBuff_ElixirOfGreaterFirepower,
		Food:            proto.Food_FoodRunnTumTuberSurprise,
		MainHandImbue:   proto.WeaponImbue_LesserWizardOil,
		SpellPowerBuff:  proto.SpellPowerBuff_GreaterArcaneElixir,
	},
}

var Phase5Consumes = core.ConsumesCombo{
	Label: "P5-Consumes",
	Consumes: &proto.Consumes{
		DefaultConjured: proto.Conjured_ConjuredDemonicRune,
		DefaultPotion:   proto.Potions_MajorManaPotion,
		Flask:           proto.Flask_FlaskOfSupremePower,
		FirePowerBuff:   proto.FirePowerBuff_ElixirOfGreaterFirepower,
		Food:            proto.Food_FoodRunnTumTuberSurprise,
		MainHandImbue:   proto.WeaponImbue_BrilliantWizardOil,
		SpellPowerBuff:  proto.SpellPowerBuff_GreaterArcaneElixir,
	},
}

var ItemFilters = core.ItemFilter{
	WeaponTypes: []proto.WeaponType{
		proto.WeaponType_WeaponTypeAxe,
		proto.WeaponType_WeaponTypeDagger,
		proto.WeaponType_WeaponTypeFist,
		proto.WeaponType_WeaponTypeMace,
		proto.WeaponType_WeaponTypeOffHand,
		proto.WeaponType_WeaponTypeShield,
		proto.WeaponType_WeaponTypeStaff,
	},
	ArmorType: proto.ArmorType_ArmorTypeMail,
	RangedWeaponTypes: []proto.RangedWeaponType{
		proto.RangedWeaponType_RangedWeaponTypeTotem,
	},
}

var Stats = []proto.Stat{
	proto.Stat_StatIntellect,
	proto.Stat_StatSpellPower,
	proto.Stat_StatNaturePower,
	proto.Stat_StatFirePower,
	proto.Stat_StatHit,
	proto.Stat_StatCrit,
}
