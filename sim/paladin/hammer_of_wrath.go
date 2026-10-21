package paladin

import (
	"time"

	"github.com/wowsims/classic/sim/core/proto"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

// HammerOfWrathDamage is spellconst/paladin.json's own roll for ids 24275,
// 24274 and 24239 (rank 3 rolls 473.6-522.4 at level 60: a centre of 498
// at its own level, growing 3.1 a level to level 65, 0.0981 wide).
var HammerOfWrathDamage = [HammerOfWrathRanks + 1]clientdamage.Effect{
	{},
	{Amount: 288, Variance: 0.1, PerLevel: 2.4, SpellLevel: 44, MaxLevel: 49},
	{Amount: 388, Variance: 0.1, PerLevel: 2.7, SpellLevel: 52, MaxLevel: 57},
	{Amount: 498, Variance: 0.098113, PerLevel: 3.1, SpellLevel: 60, MaxLevel: 65},
}

func (paladin *Paladin) registerHammerOfWrath() {
	ranks := []struct {
		level    int32
		spellID  int32
		manaCost float64
	}{
		{level: 44, spellID: 24275, manaCost: 295},
		{level: 52, spellID: 24274, manaCost: 360},
		{level: 60, spellID: 24239, manaCost: 425},
	}

	cd := core.Cooldown{
		Timer:    paladin.NewTimer(),
		Duration: time.Second * 6,
	}

	for i, rank := range ranks {
		rank := rank
		if paladin.Level < rank.level {
			break
		}

		damage := HammerOfWrathDamage[i+1]
		casterLevel := int(paladin.Level)

		paladin.GetOrRegisterSpell(core.SpellConfig{
			ActionID:    core.ActionID{SpellID: rank.spellID},
			SpellSchool: core.SpellSchoolHoly,
			DefenseType: core.DefenseTypeRanged,
			ProcMask:    core.ProcMaskRangedSpecial, // TODO to be tested
			Flags:       core.SpellFlagMeleeMetrics | core.SpellFlagAPL,
			CastType:    proto.CastType_CastTypeRanged,

			Rank:           i + 1,
			RequiredLevel:  int(rank.level),
			SpellCode:      SpellCode_PaladinHammerOfWrath,
			ClassSpellMask: PaladinSpellMaskHammerOfWrath,

			ManaCost: core.ManaCostOptions{
				FlatCost: rank.manaCost,
			},
			Cast: core.CastConfig{
				DefaultCast: core.Cast{
					GCD:      time.Second,
					CastTime: time.Second,
				},
				IgnoreHaste: true,
				CD:          cd,
			},

			DamageMultiplier: 1,
			ThreatMultiplier: 1,
			BonusCoefficient: 0.429,
			ClientBaseDamage: damage.Range(casterLevel),
			BonusHitRating:   -float64(paladin.Talents.Precision) * core.HitRatingPerHitChance,

			ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
				return sim.IsExecutePhase20()
			},

			ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				spell.CalcAndDealDamage(sim, target, damage.Roll(sim, casterLevel), spell.OutcomeRangedHitAndCrit)
			},
		})
	}
}
