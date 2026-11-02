package common

import (
	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/common/itemhelpers"
	"github.com/wowsims/classic/sim/core"
)

// clientWeaponProc is one chance-on-hit weapon whose proc deals a single
// damage roll: the proc spell the item names (ItemXItemEffect, trigger 2)
// and that spell's damage effect (SpellEffect Effect 2) as the client
// states it. The proc rate is the engine's own; the client states none.
type clientWeaponProc struct {
	name    string
	ppm     float64
	spellID int32
	school  core.SpellSchool
	roll    clientdamage.Effect
	coef    float64
	defense core.DefenseType
}

// Chance-on-hit weapons the engine had no effect for. Their proc rate is
// the convention the other weapons here use (1 a minute); the client
// states none.
const (
	ClawOfTheShadowmancer = 2912
	Thornspike            = 6681
)

// clientWeaponProcs is keyed by item id.
var clientWeaponProcs = map[int32]clientWeaponProc{
	ClawOfTheShadowmancer:    {"Claw of the Shadowmancer", 1.0, 16409, core.SpellSchoolShadow, clientdamage.Effect{Amount: 35}, 0, core.DefenseTypeMagic},
	Thornspike:               {"Thornspike", 1.0, 1293183, core.SpellSchoolPhysical, clientdamage.Effect{Amount: 45}, 0, core.DefenseTypeMelee},
	AlcorsSunrazor:           {"Alcor's Sunrazor", 1.0, 18833, core.SpellSchoolFire, clientdamage.Effect{Amount: 105, Variance: 0.2}, 0, core.DefenseTypeMagic},
	BaronCharrsSceptre:       {"Baron Charr's Sceptre", 1.0, 13442, core.SpellSchoolFire, clientdamage.Effect{Amount: 90, Variance: 0}, 0, core.DefenseTypeMagic},
	BlackhandDoomsaw:         {"Blackhand Doomsaw", 0.4, 16549, core.SpellSchoolPhysical, clientdamage.Effect{Amount: 448, Variance: 0.33333}, 0, core.DefenseTypeMelee},
	Bloodfist:                {"Bloodfist", 4, 16433, core.SpellSchoolPhysical, clientdamage.Effect{Amount: 35, Variance: 0}, 0, core.DefenseTypeMelee},
	BloodletterScalpel:       {"Bloodletter Scalpel", 1.0, 13486, core.SpellSchoolPhysical, clientdamage.Effect{Amount: 77, Variance: 0.15385}, 0, core.DefenseTypeMelee},
	BonechillHammer:          {"Bonechill Hammer", 1.0, 18276, core.SpellSchoolFrost, clientdamage.Effect{Amount: 105, Variance: 0}, 0, core.DefenseTypeMagic},
	ColdrageDagger:           {"Coldrage Dagger", 2.2, 1293790, core.SpellSchoolFrost, clientdamage.Effect{Amount: 42, Variance: 0.4}, 0, core.DefenseTypeMagic},
	Darrowspike:              {"Darrowspike", 1.0, 18276, core.SpellSchoolFrost, clientdamage.Effect{Amount: 105, Variance: 0}, 0, core.DefenseTypeMagic},
	Deathbringer:             {"Deathbringer", 1.0, 1305398, core.SpellSchoolShadow, clientdamage.Effect{Amount: 140, Variance: 0.5}, 0, core.DefenseTypeMagic},
	DrakeTalonCleaver:        {"Drake Talon Cleaver", 1.0, 21140, core.SpellSchoolPhysical, clientdamage.Effect{Amount: 240, Variance: 0}, 0.0, core.DefenseTypeMelee},
	EbonHand:                 {"Ebon Hand", 1.0, 18211, core.SpellSchoolShadow, clientdamage.Effect{Amount: 231, Variance: 0.6087}, 0, core.DefenseTypeMagic},
	GryphonRidersStormhammer: {"Gryphon Rider's Stormhammer", 1.0, 18081, core.SpellSchoolNature, clientdamage.Effect{Amount: 91, Variance: 0.31481}, 0, core.DefenseTypeMagic},
	GutRipper:                {"Gut Ripper", 1.0, 18107, core.SpellSchoolPhysical, clientdamage.Effect{Amount: 105, Variance: 0.24074}, 0, core.DefenseTypeMelee},
	HammerOfTheNorthernWind:  {"Hammer of the Northern Wind", 3.5, 13439, core.SpellSchoolFrost, clientdamage.Effect{Amount: 50, Variance: 0.4}, 0, core.DefenseTypeMagic},
	JekliksCrusher:           {"Jeklik's Crusher", 4.0, 24257, core.SpellSchoolPhysical, clientdamage.Effect{Amount: 210, Variance: 0.09524}, 0.0, core.DefenseTypeMelee},
	JoonhosMercy:             {"Joonho's Mercy", 1.0, 20883, core.SpellSchoolArcane, clientdamage.Effect{Amount: 84, Variance: 0}, 0, core.DefenseTypeMagic},
	LinkensSwordOfMastery:    {"Linken's Sword of Mastery", 1.0, 18089, core.SpellSchoolNature, clientdamage.Effect{Amount: 98, Variance: 0.5}, 0, core.DefenseTypeMagic},
	Nightblade:               {"Nightblade", 1.0, 18211, core.SpellSchoolShadow, clientdamage.Effect{Amount: 231, Variance: 0.6087}, 0, core.DefenseTypeMagic},
	PendulumOfDoom:           {"Pendulum of Doom", 0.5, 10373, core.SpellSchoolPhysical, clientdamage.Effect{Amount: 311, Variance: 0.33333}, 0, core.DefenseTypeMelee},
	PerditionsBlade:          {"Perdition's Blade", 2.8, 23267, core.SpellSchoolFire, clientdamage.Effect{Amount: 48, Variance: 0.33333}, 0, core.DefenseTypeMagic},
	SatyrsLash:               {"Satyr's Lash", 1.0, 1292155, core.SpellSchoolShadow, clientdamage.Effect{Amount: 70, Variance: 0.42857}, 0, core.DefenseTypeMagic},
	SearingNeedle:            {"Searing Needle", 1.0, 16454, core.SpellSchoolFire, clientdamage.Effect{Amount: 56, Variance: 0}, 0, core.DefenseTypeMagic},
	TheNeedler:               {"The Needler", 3.0, 16405, core.SpellSchoolPhysical, clientdamage.Effect{Amount: 75, Variance: 0}, 0, core.DefenseTypeMelee},
	VilerendSlicer:           {"Vilerend Slicer", 1.0, 1320808, core.SpellSchoolPhysical, clientdamage.Effect{Amount: 225, Variance: 0}, 0, core.DefenseTypeMelee},
	ViskagTheBloodletter:     {"Vis'kag the Bloodletter", 0.6, 1305394, core.SpellSchoolPhysical, clientdamage.Effect{Amount: 236, Variance: 0.33333}, 0, core.DefenseTypeMelee},
	ZulianSlicer:             {"Zulian Slicer", 1.2, 24251, core.SpellSchoolNature, clientdamage.Effect{Amount: 84, Variance: 0.28571}, 0.35, core.DefenseTypeMelee},
}

// registerClientWeaponProc registers item's proc from clientWeaponProcs.
func registerClientWeaponProc(item int32) {
	p, ok := clientWeaponProcs[item]
	if !ok {
		panic("no client weapon proc for an item")
	}
	// The fork's own --tags=with_db table omits some items Forever's simdb
	// carries (Thornspike); the engine panics registering an effect for an
	// item its table lacks, so those are left to the site's build.
	if _, inTable := core.ItemsByID[item]; core.WITH_DB && !inTable {
		return
	}
	bounds := p.roll.Range(0)
	itemhelpers.CreateWeaponCoHProcDamage(item, p.name, p.ppm, p.spellID, p.school, bounds[0], bounds[1]-bounds[0], p.coef, p.defense)
}
