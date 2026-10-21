package paladin

import (
	"time"

	"github.com/wowsims/classic/sim/core/proto"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

// HolyWrathDamage is spellconst/paladin.json's own roll for ids 2812 and
// 10318 (rank 2 rolls 490-576 at level 60: a centre of 533 at its own
// level, growing 1.9 a level to level 64, 0.1614 wide).
var HolyWrathDamage = [HolyWrathRanks + 1]clientdamage.Effect{
	{},
	{Amount: 395, Variance: 0.167089, PerLevel: 1.6, SpellLevel: 50, MaxLevel: 54},
	{Amount: 533, Variance: 0.161351, PerLevel: 1.9, SpellLevel: 60, MaxLevel: 64},
}

func (paladin *Paladin) registerHolyWrath() {
	ranks := []struct {
		level    int32
		spellID  int32
		manaCost float64
	}{
		{level: 50, spellID: 2812, manaCost: 645},
		{level: 60, spellID: 10318, manaCost: 805},
	}

	var results []*core.SpellResult

	for i, rank := range ranks {
		rank := rank
		if paladin.Level < rank.level {
			break
		}

		damage := HolyWrathDamage[i+1]
		casterLevel := int(paladin.Level)

		holyWrathSpell := paladin.GetOrRegisterSpell(core.SpellConfig{
			SpellCode:      SpellCode_PaladinHolyWrath,
			ClassSpellMask: PaladinSpellMaskHolyWrath,
			ActionID:       core.ActionID{SpellID: rank.spellID},
			SpellSchool:    core.SpellSchoolHoly,
			DefenseType:    core.DefenseTypeMagic,
			ProcMask:       core.ProcMaskSpellDamage, // TODO to be tested
			Flags:          core.SpellFlagAPL,

			RequiredLevel: int(rank.level),
			Rank:          i + 1,

			ManaCost: core.ManaCostOptions{
				FlatCost: rank.manaCost,
			},
			Cast: core.CastConfig{
				DefaultCast: core.Cast{
					GCD:      core.GCDDefault,
					CastTime: time.Second * 2,
				},

				CD: core.Cooldown{
					Timer:    paladin.NewTimer(),
					Duration: time.Second * 60,
				},
			},

			DamageMultiplier: 1.0,
			ThreatMultiplier: 1,
			BonusCoefficient: 0.19,
			ClientBaseDamage: damage.Range(casterLevel),

			ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
				results = results[:0]
				for _, target := range paladin.Env.Encounter.TargetUnits {
					if target.MobType == proto.MobType_MobTypeDemon || target.MobType == proto.MobType_MobTypeUndead {
						result := spell.CalcDamage(sim, target, damage.Roll(sim, casterLevel), spell.OutcomeMagicHitAndCrit)
						results = append(results, result)
					}
				}

				for _, result := range results {
					spell.DealDamage(sim, result)
				}
			},
		})

		paladin.holyWrath = append(paladin.holyWrath, holyWrathSpell)
	}
}
