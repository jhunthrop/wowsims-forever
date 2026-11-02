package warlock

import (
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

func (warlock *Warlock) makeImp() *WarlockPet {
	cfg := PetConfig{
		Name:          "Imp",
		PowerModifier: 0.33,
	}

	cfg.Stats, cfg.AutoAttacks = petSheetAtLevel(warlock.Level, impSheet)

	return warlock.makePet(cfg, warlock.Options.Summon == proto.WarlockOptions_Imp)
}

// impSheet is the demon's base sheet at the levels the engine carries one
// for (petSheetLevels); petSheetAtLevel interpolates between them.
func impSheet(level int32) (cfg PetConfig) {
	switch level {
	case 25:
		cfg.Stats = stats.Stats{
			stats.Strength:  47,
			stats.Agility:   25,
			stats.Stamina:   49,
			stats.Intellect: 94,
			stats.Spirit:    95,
			stats.Mana:      149,
			stats.MP5:       0,
		}
	case 40:
		cfg.Stats = stats.Stats{
			stats.Strength:  70,
			stats.Agility:   29,
			stats.Stamina:   67,
			stats.Intellect: 163,
			stats.Spirit:    163,
			stats.Mana:      318,
			stats.MP5:       0,
		}
	case 50:
		cfg.Stats = stats.Stats{
			stats.Strength:  101,
			stats.Agility:   32,
			stats.Stamina:   71,
			stats.Intellect: 212,
			stats.Spirit:    211,
			stats.Mana:      476,
			stats.MP5:       0,
		}
	case 60:
		cfg.Stats = stats.Stats{
			stats.Strength:  122,
			stats.Agility:   35,
			stats.Stamina:   86,
			stats.Intellect: 264,
			stats.Spirit:    260,
			stats.Mana:      576,
			stats.MP5:       0,
		}
	}
	return cfg
}

// FireboltDamage is the client's roll for Firebolt 3110 through 11763
// (spellconst/warlock.json, 1.60.1.70707): rank 7 is a centre of 44 at its
// own level 58, 0.1136 wide, growing 0.6 a level to 63, so 42.63-47.77 at 60.
// The engine dealt a flat 44 before.
var FireboltDamage = [FireboltRanks + 1]clientdamage.Effect{
	{},
	{Amount: 4, Variance: 0.285714, PerLevel: 0.1, SpellLevel: 1, MaxLevel: 5},
	{Amount: 7, Variance: 0.153846, PerLevel: 0.2, SpellLevel: 8, MaxLevel: 13},
	{Amount: 12, Variance: 0.166667, PerLevel: 0.3, SpellLevel: 18, MaxLevel: 23},
	{Amount: 17, Variance: 0.114286, PerLevel: 0.3, SpellLevel: 28, MaxLevel: 33},
	{Amount: 25, Variance: 0.117647, PerLevel: 0.4, SpellLevel: 38, MaxLevel: 43},
	{Amount: 35, Variance: 0.112676, PerLevel: 0.5, SpellLevel: 48, MaxLevel: 53},
	{Amount: 44, Variance: 0.113636, PerLevel: 0.6, SpellLevel: 58, MaxLevel: 63},
}

func (wp *WarlockPet) registerImpFireboltSpell() {
	// The highest rank the demon's level has learned.
	rank := topLearnedRank(FireboltLevel[1:], int(wp.owner.Level))

	if wp.owner.Options.MaxFireboltRank != proto.WarlockOptions_NoMaximum {
		rank = min(rank, int(wp.owner.Options.MaxFireboltRank))
	}

	spellCoeff := [8]float64{0, .164, .314, .529, .571, .571, .571, .571}[rank]
	// The classic tooltip roll (rank 7 {85, 96}) became the client's own
	// amount in the rotation-accuracy audit (the same halving shadowbolt.go
	// documents across the warlock's kit); FireboltDamage now carries the
	// client's variance and per-level growth as well.
	damage := FireboltDamage[rank]
	petLevel := int(wp.Level)
	spellId := [8]int32{0, 3110, 7799, 7800, 7801, 7802, 11762, 11763}[rank]
	manaCost := [8]float64{0, 10, 20, 35, 50, 70, 95, 115}[rank]
	level := [8]int{0, 1, 8, 18, 28, 38, 48, 58}[rank]

	improvedImp := []float64{1, 1.1, 1.2, 1.3}[wp.owner.Talents.ImprovedImp]

	wp.primaryAbility = wp.RegisterSpell(core.SpellConfig{
		ActionID:         core.ActionID{SpellID: spellId},
		SpellSchool:      core.SpellSchoolFire,
		DefenseType:      core.DefenseTypeMagic,
		ProcMask:         core.ProcMaskSpellDamage,
		Rank:             rank,
		RequiredLevel:    level,
		ClientBaseDamage: damage.Range(petLevel),

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: time.Millisecond * 1000,
				// FOREVER: Improved Firebolt is not in the client's trees.
				// CastTime: time.Millisecond * (2000 - time.Duration(500*wp.owner.Talents.ImprovedFirebolt)),
				CastTime: time.Millisecond * 2000,
			},
			// Adding an artificial CD to account for real delay in imp casts in-game
			CD: core.Cooldown{
				Timer:    wp.NewTimer(),
				Duration: time.Millisecond * 200,
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: spellCoeff,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			wp.applyOwnerSpellHit(spell)
			result := spell.CalcDamage(sim, target, damage.Roll(sim, petLevel)*improvedImp, spell.OutcomeMagicHitAndCrit)

			spell.DealDamage(sim, result)
		},
	})
}
