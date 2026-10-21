package paladin

import (
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// ExorcismDamage is spellconst/paladin.json's own roll for ids 879
// through 10314: the centre at the spell's level, the per-level growth to
// five levels above it, and the width of the roll (rank 6 rolls
// 474.7-529.3 at level 60: the centre of 502 plus 3.2 a level for the five
// levels above its own, 0.1086 wide).
var ExorcismDamage = [ExorcismRanks + 1]clientdamage.Effect{
	{},
	{Amount: 79, Variance: 0.133333, PerLevel: 1.2, SpellLevel: 20, MaxLevel: 25},
	{Amount: 141, Variance: 0.123457, PerLevel: 1.6, SpellLevel: 28, MaxLevel: 33},
	{Amount: 202, Variance: 0.121212, PerLevel: 2, SpellLevel: 36, MaxLevel: 41},
	{Amount: 291, Variance: 0.117647, PerLevel: 2.4, SpellLevel: 44, MaxLevel: 49},
	{Amount: 384, Variance: 0.110577, PerLevel: 2.8, SpellLevel: 52, MaxLevel: 57},
	{Amount: 502, Variance: 0.108614, PerLevel: 3.2, SpellLevel: 60, MaxLevel: 65},
}

func (paladin *Paladin) registerExorcism() {
	ranks := []struct {
		level    int32
		spellID  int32
		manaCost float64
	}{
		{level: 20, spellID: 879, manaCost: 85},
		{level: 28, spellID: 5614, manaCost: 135},
		{level: 36, spellID: 5615, manaCost: 180},
		{level: 44, spellID: 10312, manaCost: 235},
		{level: 52, spellID: 10313, manaCost: 285},
		{level: 60, spellID: 10314, manaCost: 345},
	}

	for i, rank := range ranks {
		rank := rank
		if paladin.Level < rank.level {
			break
		}

		damage := ExorcismDamage[i+1]
		casterLevel := int(paladin.Level)

		spell := paladin.RegisterSpell(core.SpellConfig{
			ActionID:    core.ActionID{SpellID: rank.spellID},
			SpellSchool: core.SpellSchoolHoly,
			DefenseType: core.DefenseTypeMagic,
			ProcMask:    core.ProcMaskSpellDamage,
			Flags:       core.SpellFlagMeleeMetrics | core.SpellFlagAPL | core.SpellFlagBinary, //Logs show it never has partial resists, No clue why, still misses

			RequiredLevel: int(rank.level),
			Rank:          i + 1,

			SpellCode:      SpellCode_PaladinExorcism,
			ClassSpellMask: PaladinSpellMaskExorcism,
			ManaCost: core.ManaCostOptions{
				FlatCost: rank.manaCost,
			},

			Cast: core.CastConfig{
				DefaultCast: core.Cast{
					GCD: core.GCDDefault,
				},
				CD: core.Cooldown{
					Timer:    paladin.NewTimer(),
					Duration: time.Second * 15,
				},
			},

			DamageMultiplier: 1,
			ThreatMultiplier: 1,

			BonusCoefficient: 0.429,
			ClientBaseDamage: damage.Range(casterLevel),

			ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
				return target.MobType == proto.MobType_MobTypeDemon || target.MobType == proto.MobType_MobTypeUndead
			},

			ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				spell.CalcAndDealDamage(sim, target, damage.Roll(sim, casterLevel), spell.OutcomeMagicHitAndCrit)
			},
		})

		paladin.exorcism = append(paladin.exorcism, spell)
	}
}
