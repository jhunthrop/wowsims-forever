package warrior

import "github.com/wowsims/classic/sim/common/clientdamage"

// The client's damage for every warrior ability that has a flat or
// periodic amount, one clientdamage.Effect per rank, indexed by the same
// rank label as the generated arrays. Where the generated
// <Spell>BaseDamage row is the effect the ability rolls (the spell's
// school-damage effect, else its first), the table is read from it;
// spellconst_damage_test.go checks every rank against the vendored
// client file. A weapon-percent spell's weapon share is modelled in its
// ability file; the table here is only the flat the client adds to it.

var (
	HamstringDamage    = generatedDamage(HamstringBaseDamage[:], HamstringPointsPerLevel[:], HamstringLevel[:], HamstringMaxLevel[:])
	PummelDamage       = generatedDamage(PummelBaseDamage[:], PummelPointsPerLevel[:], PummelLevel[:], PummelMaxLevel[:])
	ThunderClapDamage  = generatedDamage(ThunderClapBaseDamage[:], ThunderClapPointsPerLevel[:], ThunderClapLevel[:], ThunderClapMaxLevel[:])
	RendDamage         = generatedDamage(RendBaseDamage[:], RendPointsPerLevel[:], RendLevel[:], RendMaxLevel[:])
	HeroicStrikeDamage = generatedDamage(HeroicStrikeBaseDamage[:], HeroicStrikePointsPerLevel[:], HeroicStrikeLevel[:], HeroicStrikeMaxLevel[:])
	CleaveDamage       = generatedDamage(CleaveBaseDamage[:], CleavePointsPerLevel[:], CleaveLevel[:], CleaveMaxLevel[:])
	ExecuteDamage      = generatedDamage(ExecuteBaseDamage[:], ExecutePointsPerLevel[:], ExecuteLevel[:], ExecuteMaxLevel[:])
	OverpowerDamage    = generatedDamage(OverpowerBaseDamage[:], OverpowerPointsPerLevel[:], OverpowerLevel[:], OverpowerMaxLevel[:])
	BloodthirstDamage  = generatedDamage(BloodthirstBaseDamage[:], BloodthirstPointsPerLevel[:], BloodthirstLevel[:], BloodthirstMaxLevel[:])
	ShieldSlamDamage   = generatedDamage(ShieldSlamBaseDamage[:], ShieldSlamPointsPerLevel[:], ShieldSlamLevel[:], ShieldSlamMaxLevel[:])
)

// MortalStrikeDamage is effect 121's flat bonus (ids 12294, 21551, 21552
// and 21553 / 27580): the generated MortalStrikeBaseDamage reads the
// spell's first effect, the -50% healing aura, so the client's own
// amounts are stated here. None of them has a width or a level growth.
var MortalStrikeDamage = [MortalStrikeRanks + 1]clientdamage.Effect{
	{},
	{Amount: 85, SpellLevel: 40},
	{Amount: 110, SpellLevel: 48},
	{Amount: 135, SpellLevel: 54},
	{Amount: 160, SpellLevel: 60},
}

// RevengeDamage is the school-damage roll of spells 6572 through 25288:
// the centre is the client's amount and the width its Variance, about a
// fifth of the centre at every rank. The vanilla table (12-14 through
// 81-99) it replaces was a different spell.
var RevengeDamage = [RevengeRanks + 1]clientdamage.Effect{
	{},
	{Amount: 22, Variance: 0.153846, SpellLevel: 14},
	{Amount: 34, Variance: 0.2, SpellLevel: 24},
	{Amount: 48, Variance: 0.214286, SpellLevel: 34},
	{Amount: 82, Variance: 0.208333, SpellLevel: 44},
	{Amount: 121, Variance: 0.197183, SpellLevel: 54},
	{Amount: 153, Variance: 0.2, SpellLevel: 60},
}

func generatedDamage(rolls [][]float64, perLevel []float64, spellLevels, maxLevels []int) []clientdamage.Effect {
	return clientdamage.FromTable(rolls, perLevel, spellLevels, maxLevels)
}

// SlamDamage is effect 17's flat bonus for the real castable Slam ids
// (slamRankSpellID): the generated SlamBaseDamage rows belong to the
// stub ids the dedup kept (see slam.go), one tier off, so the client's
// amounts for the real ranks are stated here.
var SlamDamage = [SlamRanks + 1]clientdamage.Effect{
	{},
	{Amount: 16, SpellLevel: 20},
	{Amount: 32, SpellLevel: 30},
	{Amount: 43, SpellLevel: 38},
	{Amount: 68, SpellLevel: 46},
	{Amount: 87, SpellLevel: 54},
}
