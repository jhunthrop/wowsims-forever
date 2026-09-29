// Package conformance walks every registered spec's spellbook at a
// ladder of levels and checks each spell's numbers against the Forever
// client's own spell constants (sim/core/spellconst), so a mismatch is
// a test failure instead of a guess.
//
// It cannot live in sim/core: core is imported by every class package,
// so core importing a class package back (to build a real character)
// would cycle. It cannot reuse the class packages' own
// level_smoke_test.go presets either — Talents and SpecOptions there
// are declared in _test.go files, which Go does not export outside
// that package's own test binary. This package therefore carries its
// own copies of the same Class/Race/Talents/SpecOptions each spec
// package's TestLevelSmoke already builds (sim/<class>/.../level_smoke_test.go),
// kept identical to the source so both build the same bare character.
package conformance

import (
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/mage"
	"github.com/wowsims/classic/sim/warrior"
)

// Preset is one class/spec combination this report measures.
type Preset struct {
	// Label names the spec in every golden row.
	Label string
	// ClientClassSlug is the key into
	// sim/core/testdata/conformance/client/<slug>.json — the client
	// class file a spec's spells are checked against. Several specs
	// share a slug (e.g. protection and retribution paladin both check
	// against paladin.json).
	ClientClassSlug string

	Class              proto.Class
	Race               proto.Race
	Talents            string
	SpecOptions        interface{}
	DistanceFromTarget float64
}

// Presets mirrors, one entry per registered spec (sim/register_all.go),
// the Class/Race/Talents/SpecOptions its own level_smoke_test.go
// already builds.
var Presets = []Preset{
	{
		Label:           "Hunter",
		ClientClassSlug: "hunter",
		Class:           proto.Class_ClassHunter,
		Race:            proto.Race_RaceOrc,
		Talents:         "-05451002503051-33400023023",
		SpecOptions: &proto.Player_Hunter{
			Hunter: &proto.Hunter{
				Options: &proto.Hunter_Options{
					Ammo:           proto.Hunter_Options_RazorArrow,
					PetType:        proto.Hunter_Options_Cat,
					PetUptime:      1,
					PetAttackSpeed: 2.0,
				},
			},
		},
		DistanceFromTarget: 25,
	},
	{
		Label:           "Mage",
		ClientClassSlug: "mage",
		Class:           proto.Class_ClassMage,
		Race:            proto.Race_RaceTroll,
		// mage.ForeverFrostTalents (sim/mage/talents.go) is exported
		// non-test, unlike every other spec's preset talents, so this
		// one alone is a reference rather than a copy.
		Talents: mage.ForeverFrostTalents,
		SpecOptions: &proto.Player_Mage{
			Mage: &proto.Mage{
				Options: &proto.Mage_Options{
					Armor: proto.Mage_Options_MoltenArmor,
				},
			},
		},
	},
	{
		Label:           "SMRuinWarlock",
		ClientClassSlug: "warlock",
		Class:           proto.Class_ClassWarlock,
		Race:            proto.Race_RaceOrc,
		Talents:         "5502203112201105--52500051020001",
		SpecOptions:     defaultDestroWarlockOptions,
	},
	{
		Label:           "DSRuinWarlock",
		ClientClassSlug: "warlock",
		Class:           proto.Class_ClassWarlock,
		Race:            proto.Race_RaceOrc,
		Talents:         "25002-2050300152201-52500051020001",
		SpecOptions:     defaultDestroWarlockOptions,
	},
	{
		Label:           "ProtectionPaladin",
		ClientClassSlug: "paladin",
		Class:           proto.Class_ClassPaladin,
		Race:            proto.Race_RaceHuman,
		Talents:         "-053020335001551-0500535",
		SpecOptions: &proto.Player_ProtectionPaladin{
			ProtectionPaladin: &proto.ProtectionPaladin{
				Options: &proto.PaladinOptions{
					PrimarySeal:   proto.PaladinSeal_Righteousness,
					RighteousFury: true,
				},
			},
		},
	},
	{
		Label:           "RetributionPaladin",
		ClientClassSlug: "paladin",
		Class:           proto.Class_ClassPaladin,
		Race:            proto.Race_RaceHuman,
		Talents:         "500501-503-52230351200315",
		SpecOptions: &proto.Player_RetributionPaladin{
			RetributionPaladin: &proto.RetributionPaladin{
				Options: &proto.PaladinOptions{
					PrimarySeal: proto.PaladinSeal_Righteousness,
				},
			},
		},
	},
	{
		Label:           "FuryWarrior",
		ClientClassSlug: "warrior",
		Class:           proto.Class_ClassWarrior,
		Race:            proto.Race_RaceOrc,
		Talents:         warrior.ForeverFuryTalents,
		SpecOptions: &proto.Player_Warrior{
			Warrior: &proto.Warrior{
				Options: &proto.Warrior_Options{
					StartingRage: 50,
					Shout:        proto.WarriorShout_WarriorShoutBattle,
				},
			},
		},
	},
	{
		Label:           "ProtectionWarrior",
		ClientClassSlug: "warrior",
		Class:           proto.Class_ClassWarrior,
		Race:            proto.Race_RaceOrc,
		Talents:         warrior.ForeverProtectionTalents,
		SpecOptions: &proto.Player_TankWarrior{
			TankWarrior: &proto.TankWarrior{
				Options: &proto.TankWarrior_Options{
					Shout:        proto.WarriorShout_WarriorShoutCommanding,
					StartingRage: 0,
				},
			},
		},
	},
	{
		Label:           "BalanceDruid",
		ClientClassSlug: "druid",
		Class:           proto.Class_ClassDruid,
		Race:            proto.Race_RaceTauren,
		Talents:         "5000550012551251--5005031",
		SpecOptions: &proto.Player_BalanceDruid{
			BalanceDruid: &proto.BalanceDruid{
				Options: &proto.BalanceDruid_Options{
					OkfUptime: 0.2,
				},
			},
		},
	},
	{
		Label:           "FeralDruid",
		ClientClassSlug: "druid",
		Class:           proto.Class_ClassDruid,
		Race:            proto.Race_RaceTauren,
		Talents:         "500005301-5500020323202151-15",
		SpecOptions: &proto.Player_FeralDruid{
			FeralDruid: &proto.FeralDruid{
				Options: &proto.FeralDruid_Options{
					InnervateTarget:   &proto.UnitReference{},
					LatencyMs:         100,
					AssumeBleedActive: true,
				},
			},
		},
	},
	{
		Label:           "ShadowPriest",
		ClientClassSlug: "priest",
		Class:           proto.Class_ClassPriest,
		Race:            proto.Race_RaceUndead,
		// Shadow tree digit 5 (Improved Shadow Word: Pain, position 5)
		// was "5", exceeding that talent's real max_rank of 2
		// (data/builds/1.60.1.70009/talents/priest.json) - it doesn't
		// come from Shadowform's own row shifting (this string still
		// aligns talent-by-talent through Mind Flay/Improved Mind
		// Blast, confirmed against the parsed proto), just a stale
		// digit. An out-of-range rank made Shadow Word: Pain tick 11
		// times (33s) in the conformance report instead of the client's
		// real ranks-of-2 ceiling of 8 ticks (24s); capped to 2, the
		// max a character can actually have.
		//
		// Discipline tree digit 2 (Wand Specialization, position 2,
		// max_rank 2 in the same talents.json) was also "5" - the same
		// stale-digit defect, just in the other tree. Nothing panicked
		// on it before tonight because no engine code read
		// Talents.WandSpecialization until sim/priest/shoot.go started
		// wand-weaving casters; an out-of-range 5 indexes past the
		// engine's [3]float64 rank table and panics every build,
		// skipping every level for this preset. Capped to 2, the same
		// fix as the Shadow tree digit above.
		Talents: "0212301302--5002204103501251",
		SpecOptions: &proto.Player_ShadowPriest{
			ShadowPriest: &proto.ShadowPriest{
				Options: &proto.ShadowPriest_Options{
					Armor: proto.ShadowPriest_Options_InnerFire,
				},
			},
		},
	},
	{
		Label:           "ElementalShaman",
		ClientClassSlug: "shaman",
		Class:           proto.Class_ClassShaman,
		Race:            proto.Race_RaceTroll,
		Talents:         "550331050002151--50105301005",
		SpecOptions: &proto.Player_ElementalShaman{
			ElementalShaman: &proto.ElementalShaman{
				Options: &proto.ElementalShaman_Options{},
			},
		},
	},
	{
		Label:           "EnhancementShaman",
		ClientClassSlug: "shaman",
		Class:           proto.Class_ClassShaman,
		Race:            proto.Race_RaceTroll,
		Talents:         "05-5025002105023051-05105301",
		SpecOptions: &proto.Player_EnhancementShaman{
			EnhancementShaman: &proto.EnhancementShaman{
				Options: &proto.EnhancementShaman_Options{
					SyncType: proto.ShamanSyncType_Auto,
				},
			},
		},
	},
	{
		Label:           "WardenShaman",
		ClientClassSlug: "shaman",
		Class:           proto.Class_ClassShaman,
		Race:            proto.Race_RaceTroll,
		Talents:         "5203015-0505000145503151",
		SpecOptions: &proto.Player_WardenShaman{
			WardenShaman: &proto.WardenShaman{
				Options: &proto.WardenShaman_Options{},
			},
		},
	},
	{
		Label:           "CombatSwordsRogue",
		ClientClassSlug: "rogue",
		Class:           proto.Class_ClassRogue,
		Race:            proto.Race_RaceHuman,
		Talents:         "005323105-0240052020050150231",
		SpecOptions:     defaultRogueOptions,
	},
	{
		Label:           "CombatDaggersRogue",
		ClientClassSlug: "rogue",
		Class:           proto.Class_ClassRogue,
		Race:            proto.Race_RaceHuman,
		Talents:         "005023104-0233050020550100221-05",
		SpecOptions:     defaultRogueOptions,
	},
}

var defaultDestroWarlockOptions = &proto.Player_Warlock{
	Warlock: &proto.Warlock{
		Options: &proto.WarlockOptions{
			Armor:       proto.WarlockOptions_DemonArmor,
			Summon:      proto.WarlockOptions_Succubus,
			WeaponImbue: proto.WarlockOptions_NoWeaponImbue,
		},
	},
}

var defaultRogueOptions = &proto.Player_Rogue{
	Rogue: &proto.Rogue{
		Options: &proto.RogueOptions{},
	},
}
