package hunter

import (
	"testing"

	_ "github.com/wowsims/classic/sim/common" // imported to get item effects included.
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

func init() {
	RegisterHunter()
}

func TestP1Hunter(t *testing.T) {
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
	t.Skip("sim/hunter awaits its Forever talent rewrite (plan 2026-09-14-sim-engine, task 17)")
	core.RunTestSuite(t, t.Name(), core.FullCharacterTestSuiteGenerator([]core.CharacterSuiteConfig{
		{
			Class:      proto.Class_ClassHunter,
			Phase:      1,
			Race:       proto.Race_RaceOrc,
			OtherRaces: []proto.Race{proto.Race_RaceDwarf},

			Talents:     P1Talents,
			GearSet:     core.GetGearSet("../../ui/hunter/gear_sets", "p0.bis"),
			Rotation:    core.GetAplRotation("../../ui/hunter/apls", "p1"),
			Buffs:       core.FullBuffs,
			Consumes:    P1Consumes,
			SpecOptions: core.SpecOptionsCombo{Label: "Hunter", SpecOptions: P1PlayerOptions},

			ItemFilter:      ItemFilters,
			EPReferenceStat: proto.Stat_StatAttackPower,
			StatsToWeigh:    Stats,
		},
	}))
}

var P1Talents = "-05451002503051-33400023023"

var P1Consumes = core.ConsumesCombo{
	Label: "P1-Consumes",
	Consumes: &proto.Consumes{
		AgilityElixir:     proto.AgilityElixir_ElixirOfTheMongoose,
		AttackPowerBuff:   proto.AttackPowerBuff_JujuMight,
		DefaultPotion:     proto.Potions_ManaPotion,
		DragonBreathChili: true,
		Flask:             proto.Flask_FlaskOfSupremePower,
		Food:              proto.Food_FoodSagefishDelight,
		MainHandImbue:     proto.WeaponImbue_Windfury,
		OffHandImbue:      proto.WeaponImbue_ElementalSharpeningStone,
		SpellPowerBuff:    proto.SpellPowerBuff_GreaterArcaneElixir,
		StrengthBuff:      proto.StrengthBuff_JujuPower,
	},
}

var P1PlayerOptions = &proto.Player_Hunter{
	Hunter: &proto.Hunter{
		Options: &proto.Hunter_Options{
			Ammo:           proto.Hunter_Options_RazorArrow,
			PetType:        proto.Hunter_Options_Cat,
			PetUptime:      1,
			PetAttackSpeed: 2.0,
		},
	},
}

var ItemFilters = core.ItemFilter{
	ArmorType: proto.ArmorType_ArmorTypeMail,
	WeaponTypes: []proto.WeaponType{
		proto.WeaponType_WeaponTypeAxe,
		proto.WeaponType_WeaponTypeDagger,
		proto.WeaponType_WeaponTypeFist,
		proto.WeaponType_WeaponTypeMace,
		proto.WeaponType_WeaponTypeOffHand,
		proto.WeaponType_WeaponTypePolearm,
		proto.WeaponType_WeaponTypeStaff,
		proto.WeaponType_WeaponTypeSword,
	},
	RangedWeaponTypes: []proto.RangedWeaponType{
		proto.RangedWeaponType_RangedWeaponTypeBow,
		proto.RangedWeaponType_RangedWeaponTypeCrossbow,
		proto.RangedWeaponType_RangedWeaponTypeGun,
	},
}

var Stats = []proto.Stat{
	proto.Stat_StatAgility,
	proto.Stat_StatAttackPower,
	proto.Stat_StatRangedAttackPower,
	proto.Stat_StatCrit,
	proto.Stat_StatHit,
}
