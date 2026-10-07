package hunter

import "github.com/wowsims/classic/sim/common/clientdamage"

// The client's damage for every hunter ability that has a flat, direct or
// periodic amount, one clientdamage.Effect per rank, indexed by the same
// rank label as the ability files. Where the generated <Spell>BaseDamage
// row is the effect the ability rolls for the ids it registers, the table
// is read from it; the rest are stated from the client file, per the
// comment above each. spellconst_damage_test.go checks every rank against
// the vendored client file. A weapon-percent spell's weapon share is
// modelled in its ability file; the table here is only the flat the
// client adds to it.

var (
	ArcaneShotDamage    = generatedDamage(ArcaneShotBaseDamage[:], ArcaneShotPointsPerLevel[:], ArcaneShotLevel[:], ArcaneShotMaxLevel[:])
	MongooseBiteDamage  = generatedDamage(MongooseBiteBaseDamage[:], MongooseBitePointsPerLevel[:], MongooseBiteLevel[:], MongooseBiteMaxLevel[:])
	CounterattackDamage = generatedDamage(CounterattackBaseDamage[:], CounterattackPointsPerLevel[:], CounterattackLevel[:], CounterattackMaxLevel[:])
	WingClipDamage      = generatedDamage(WingClipBaseDamage[:], WingClipPointsPerLevel[:], WingClipLevel[:], WingClipMaxLevel[:])
	SniperShotDamage    = generatedDamage(SniperShotBaseDamage[:], SniperShotPointsPerLevel[:], SniperShotLevel[:], SniperShotMaxLevel[:])
	SummonHawkDamage    = generatedDamage(SummonHawkBaseDamage[:], SummonHawkPointsPerLevel[:], SummonHawkLevel[:], SummonHawkMaxLevel[:])
	VolleyDamage        = generatedDamage(VolleyBaseDamage[:], VolleyPointsPerLevel[:], VolleyLevel[:], VolleyMaxLevel[:])
	HydraShotDamage     = generatedDamage(HydraShotBaseDamage[:], HydraShotPointsPerLevel[:], HydraShotLevel[:], HydraShotMaxLevel[:])
	LacerateDamage      = generatedDamage(LacerateBaseDamage[:], LaceratePointsPerLevel[:], LacerateLevel[:], LacerateMaxLevel[:])
)

func generatedDamage(rolls [][]float64, perLevel []float64, spellLevels, maxLevels []int) []clientdamage.Effect {
	return clientdamage.FromTable(rolls, perLevel, spellLevels, maxLevels)
}

// SerpentStingDamage is the per-tick amount of the spellbook ids 1978,
// 13549-13555 and 25295. The generated SerpentStingBaseDamage reads the
// reissued family (425728-425737), whose ranks 1-8 carry a different
// amount; the spellbook (spellranks.json) lists the legacy ids, which
// the engine registers, so their own amounts are stated here.
var SerpentStingDamage = [SerpentStingRanks + 1]clientdamage.Effect{
	{},
	{Amount: 2, SpellLevel: 4},
	{Amount: 6, SpellLevel: 10},
	{Amount: 12, SpellLevel: 18},
	{Amount: 22, SpellLevel: 26},
	{Amount: 34, SpellLevel: 34},
	{Amount: 48, SpellLevel: 42},
	{Amount: 64, SpellLevel: 50},
	{Amount: 83, SpellLevel: 58},
	{Amount: 111, SpellLevel: 60},
}

// AimedShotDamage is effect 121's flat bonus of ids 19434 and 20900
// through 20904. The generated AimedShotBaseDamage[6] reads 27632, the
// level-60 duplicate that states 600, which no rank of the spellbook
// uses.
var AimedShotDamage = [AimedShotRanks + 1]clientdamage.Effect{
	{},
	{Amount: 20, SpellLevel: 20},
	{Amount: 34, SpellLevel: 28},
	{Amount: 55, SpellLevel: 36},
	{Amount: 89, SpellLevel: 44},
	{Amount: 125, SpellLevel: 52},
	{Amount: 166, SpellLevel: 60},
}

// RaptorStrikeDamage is effect 58's flat bonus of the spellbook ids 2973
// and 14260 through 14266 (the generator does not emit Raptor Strike:
// the hand RaptorStrikeRanks collides with it).
var RaptorStrikeDamage = [RaptorStrikeRanks + 1]clientdamage.Effect{
	{},
	{Amount: 5, SpellLevel: 1},
	{Amount: 11, SpellLevel: 8},
	{Amount: 21, SpellLevel: 16},
	{Amount: 30, SpellLevel: 24},
	{Amount: 35, SpellLevel: 32},
	{Amount: 40, SpellLevel: 40},
	{Amount: 55, SpellLevel: 48},
	{Amount: 70, SpellLevel: 56},
}

// ImmolationTrapTickDamage is the per-tick amount of the "Immolation
// Trap Effect" spells (13797, 14298-14301), where the client keeps the
// burn the trap spells themselves (13795, 14302-14305) only trigger. The
// growth cap stays in MaxLevel; none of the ranks grows per level.
var ImmolationTrapTickDamage = [ImmolationTrapRanks + 1]clientdamage.Effect{
	{},
	{Amount: 21, SpellLevel: 16, MaxLevel: 22},
	{Amount: 43, SpellLevel: 26, MaxLevel: 32},
	{Amount: 68, SpellLevel: 36, MaxLevel: 42},
	{Amount: 102, SpellLevel: 46, MaxLevel: 52},
	{Amount: 138, SpellLevel: 56, MaxLevel: 62},
}

// ExplosiveTrapDamage is the instant hit of the "Explosive Trap Effect"
// spells (13812, 14314, 14315): a roll about a quarter of the centre
// wide, growing per caster level up to the rank's MaxLevel (the engine
// used a single flat number per rank).
var ExplosiveTrapDamage = [ExplosiveTrapRanks + 1]clientdamage.Effect{
	{},
	{Amount: 115, Variance: 0.260870, PerLevel: 0.8, SpellLevel: 34, MaxLevel: 40},
	{Amount: 163, Variance: 0.294479, PerLevel: 1.0, SpellLevel: 44, MaxLevel: 50},
	{Amount: 229, Variance: 0.244541, PerLevel: 1.2, SpellLevel: 54, MaxLevel: 60},
}

// ExplosiveTrapTickDamage is the per-tick amount of the same spells'
// two-second area burn (effect index 1).
var ExplosiveTrapTickDamage = [ExplosiveTrapRanks + 1]clientdamage.Effect{
	{},
	{Amount: 15, SpellLevel: 34, MaxLevel: 40},
	{Amount: 24, SpellLevel: 44, MaxLevel: 50},
	{Amount: 33, SpellLevel: 54, MaxLevel: 60},
}
