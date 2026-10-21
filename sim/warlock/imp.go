package warlock

import (
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

func (warlock *Warlock) makeImp() *WarlockPet {
	cfg := PetConfig{
		Name:          "Imp",
		PowerModifier: 0.33,
	}

	switch warlock.Level {
	case 25:
		cfg.Stats = stats.Stats{
			stats.Strength:  47,
			stats.Agility:   25,
			stats.Stamina:   49,
			stats.Intellect: 94,
			stats.Spirit:    95,
			stats.Mana:      149,
			stats.MP5:       0,
			// Forever: merged from MeleeCrit 3.454 + SpellCrit 0.9075. unconfirmed
			stats.Crit: 3.454 * core.CritRatingPerCritChance,
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
			// Forever: merged from MeleeCrit 3.454 + SpellCrit 0.9075. unconfirmed
			stats.Crit: 3.454 * core.CritRatingPerCritChance,
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
			// Forever: merged from MeleeCrit 3.454 + SpellCrit 0.9075. unconfirmed
			stats.Crit: 3.454 * core.CritRatingPerCritChance,
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
			// Forever: merged from MeleeCrit 3.454 + SpellCrit 0.9075. unconfirmed
			stats.Crit: 3.454 * core.CritRatingPerCritChance,
		}
	}

	return warlock.makePet(cfg, warlock.Options.Summon == proto.WarlockOptions_Imp)
}

func (wp *WarlockPet) registerImpFireboltSpell() {
	warlockLevel := wp.owner.Level
	// assuming max rank available
	rank := map[int32]int{25: 3, 40: 5, 50: 6, 60: 7}[warlockLevel]

	if rank == 0 {
		rank = 1
	}

	if wp.owner.Options.MaxFireboltRank != proto.WarlockOptions_NoMaximum {
		rank = min(rank, int(wp.owner.Options.MaxFireboltRank))
	}

	spellCoeff := [8]float64{0, .164, .314, .529, .571, .571, .571, .571}[rank]
	// baseDamage was the classic tooltip roll (rank 7 {85, 96}) until
	// the rotation-accuracy audit compared it against spellconst/
	// warlock.json's own per-rank flat "amount" (rank 7, 11763, amount
	// 44, sp_coefficient 0.571 unchanged) - the same halving
	// shadowbolt.go's comment documents across the warlock's own kit;
	// the imp is not a player spell in spellconst, but shares the
	// pattern exactly (rank 1: client 4 vs the old roll's {7, 10}).
	baseDamage := [8]float64{0, 4, 7, 12, 17, 25, 35, 44}[rank]
	spellId := [8]int32{0, 3110, 7799, 7800, 7801, 7802, 11762, 11763}[rank]
	manaCost := [8]float64{0, 10, 20, 35, 50, 70, 95, 115}[rank]
	level := [8]int{0, 1, 8, 18, 28, 38, 48, 58}[rank]

	improvedImp := []float64{1, 1.1, 1.2, 1.3}[wp.owner.Talents.ImprovedImp]
	baseDamage *= improvedImp

	wp.primaryAbility = wp.RegisterSpell(core.SpellConfig{
		ActionID:      core.ActionID{SpellID: spellId},
		SpellSchool:   core.SpellSchoolFire,
		DefenseType:   core.DefenseTypeMagic,
		ProcMask:      core.ProcMaskSpellDamage,
		Rank:          rank,
		RequiredLevel: level,

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
			result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)

			spell.DealDamage(sim, result)
		},
	})
}
