package rogue

import (
	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

// The client's damage for every rogue ability that has a flat, direct or
// periodic amount, one clientdamage.Effect per rank, indexed by the same
// rank label as the ability files. Where the generated <Spell>BaseDamage
// row is the effect the ability rolls for the ids it registers, the table
// is read from it; the rest are stated from the client file, per the
// comment above each. spellconst_damage_test.go checks every rank against
// the vendored client file. A weapon-percent spell's weapon share is
// modelled in its ability file; the table here is only the flat the
// client adds to it. The per-combo-point terms the client's rank text
// states (Eviscerate, Rupture) are not in the vendored file's columns and
// stay in their ability files.

var (
	SinisterStrikeDamage = generatedDamage(SinisterStrikeBaseDamage[:], SinisterStrikePointsPerLevel[:], SinisterStrikeLevel[:], SinisterStrikeMaxLevel[:])
	AmbushDamage         = generatedDamage(AmbushBaseDamage[:], AmbushPointsPerLevel[:], AmbushLevel[:], AmbushMaxLevel[:])
	GarroteTickDamage    = generatedDamage(GarroteBaseDamage[:], GarrotePointsPerLevel[:], GarroteLevel[:], GarroteMaxLevel[:])
	RuptureTickDamage    = generatedDamage(RuptureBaseDamage[:], RupturePointsPerLevel[:], RuptureLevel[:], RuptureMaxLevel[:])
	GougeDamage          = generatedDamage(GougeBaseDamage[:], GougePointsPerLevel[:], GougeLevel[:], GougeMaxLevel[:])
	KickDamage           = generatedDamage(KickBaseDamage[:], KickPointsPerLevel[:], KickLevel[:], KickMaxLevel[:])

	// BackstabDamage is indexed by the engine's rank: the generator
	// emits the client's nine, rank 9 being the AQ id (25300) the engine's
	// eighth slot casts with AQ content.
	BackstabDamage = engineRanks(
		generatedDamage(BackstabBaseDamage[:], BackstabPointsPerLevel[:], BackstabLevel[:], BackstabMaxLevel[:]),
		8, core.TernaryInt(core.IncludeAQ, BackstabRanks, BackstabRanks-1))

	// EviscerateDamage is indexed by the engine's rank; its ninth slot is
	// the AQ rank (31016) or, without AQ content, rank 8's terms.
	EviscerateDamage = engineRanks(
		generatedDamage(EviscerateBaseDamage[:], EvisceratePointsPerLevel[:], EviscerateLevel[:], EviscerateMaxLevel[:]),
		EviscerateRanks, core.TernaryInt(core.IncludeAQ, EviscerateRanks, EviscerateRanks-1))
)

func generatedDamage(rolls [][]float64, perLevel []float64, spellLevels, maxLevels []int) []clientdamage.Effect {
	return clientdamage.FromTable(rolls, perLevel, spellLevels, maxLevels)
}

// engineRanks keeps ranks 0 through topRank-1 of a generated table and
// puts the table's clientTopRank in slot topRank, the way the engine's
// top rank slot follows the AQ content flag.
func engineRanks(effects []clientdamage.Effect, topRank, clientTopRank int) []clientdamage.Effect {
	ranks := append([]clientdamage.Effect(nil), effects[:topRank]...)
	return append(ranks, effects[clientTopRank])
}

// MutilateDamage is effect 121's flat bonus of the main-hand strikes
// (1310705, 399960, 1241585, 1241586): the generated MutilateBaseDamage
// reads the talent spell's first effect, so rank 1 is a dummy.
var MutilateDamage = [mutilateRanks + 1]clientdamage.Effect{
	{},
	{Amount: 23, SpellLevel: 30},
	{Amount: 33, SpellLevel: 40},
	{Amount: 48, SpellLevel: 50},
	{Amount: 67, SpellLevel: 60},
}

// InstantPoisonDamage is the roll of the "Instant Poison" damage spells
// (8680, 8685, 8689, 11335, 11336, 11337), where the client keeps the
// proc's hit apart from the weapon-enchant spells the engine registers
// (8679 and the rest). Each is about a quarter of its centre wide.
var InstantPoisonDamage = [instantPoisonRanks + 1]clientdamage.Effect{
	{},
	{Amount: 15, Variance: 0.272727, SpellLevel: 20},
	{Amount: 23, Variance: 0.235294, SpellLevel: 28},
	{Amount: 33, Variance: 0.24, SpellLevel: 36},
	{Amount: 51, Variance: 0.236842, SpellLevel: 44},
	{Amount: 71, Variance: 0.247619, SpellLevel: 52},
	{Amount: 88, Variance: 0.276923, SpellLevel: 60},
}

// DeadlyPoisonTickDamage is the per-tick amount of the "Deadly Poison"
// damage spells 434312 through 434316 (the family the generated
// DeadlyPoisonBaseDamage names for rank 1), indexed by the engine's rank;
// its fifth slot is the AQ rank or, without AQ content, rank 4's amount.
var DeadlyPoisonTickDamage = [deadlyPoisonRanks + 1]clientdamage.Effect{
	{},
	{Amount: 9, SpellLevel: 30},
	{Amount: 13, SpellLevel: 38},
	{Amount: 20, SpellLevel: 46},
	{Amount: 27, SpellLevel: 54},
	core.Ternary(core.IncludeAQ, clientdamage.Effect{Amount: 34, SpellLevel: 60}, clientdamage.Effect{Amount: 27, SpellLevel: 54}),
}

const (
	instantPoisonRanks = 6
	deadlyPoisonRanks  = 5
)
