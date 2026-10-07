package paladin

import (
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

// HolyShockDamage is spellconst/paladin.json's own roll for the damage
// spells ids 25912, 25911 and 25902 that the cast ids below (20473, 20929,
// 20930) trigger: 182, 258 and 348 at their own levels (40, 48, 56) and
// 7.5-7.9% wide, with no per-level growth. The Classic tooltip rolls these
// replaced (204-220, 279-301, 365-395) sat about 12% above them.
var HolyShockDamage = [holyShockRanks + 1]clientdamage.Effect{
	{},
	{Amount: 182, Variance: 0.075472, SpellLevel: 40},
	{Amount: 258, Variance: 0.075862, SpellLevel: 48},
	{Amount: 348, Variance: 0.078947, SpellLevel: 56},
}

const holyShockRanks = 3

func (paladin *Paladin) registerHolyShock() {
	if !paladin.Talents.HolyShock {
		return
	}

	ranks := []struct {
		level    int32
		spellID  int32
		manaCost float64
	}{
		{level: 40, spellID: 20473, manaCost: 225},
		{level: 48, spellID: 20929, manaCost: 275},
		{level: 56, spellID: 20930, manaCost: 325},
	}

	for i, rank := range ranks {
		rank := rank
		if paladin.Level < rank.level {
			break
		}

		damage := HolyShockDamage[i+1]
		casterLevel := int(paladin.Level)

		paladin.RegisterSpell(core.SpellConfig{
			ActionID:    core.ActionID{SpellID: rank.spellID},
			SpellSchool: core.SpellSchoolHoly,
			DefenseType: core.DefenseTypeMagic,
			ProcMask:    core.ProcMaskSpellDamage,
			Flags:       core.SpellFlagMeleeMetrics | core.SpellFlagAPL,

			RequiredLevel: int(rank.level),
			Rank:          i + 1,

			SpellCode:      SpellCode_PaladinHolyShock,
			ClassSpellMask: PaladinSpellMaskHolyShock,

			ManaCost: core.ManaCostOptions{
				FlatCost: rank.manaCost,
			},

			Cast: core.CastConfig{
				DefaultCast: core.Cast{
					GCD: core.GCDDefault,
				},
				CD: core.Cooldown{
					Timer: paladin.NewTimer(),
					// category_cooldown_ms is 10000 for every rank's
					// actual cast spell (spellconst/paladin.json ids
					// 20473/20929/20930): source 1.60.1.70009 client
					// spell data. 30s was Classic's original Holy
					// Shock cooldown; Forever's client shortened it to
					// 10s and this literal was never updated. Flagged
					// by sim/core/testdata/conformance/paladin.golden.md's
					// "cooldown_ms 10000->30000" row.
					Duration: time.Second * 10,
				},
			},

			DamageMultiplier: 1,
			ThreatMultiplier: 1,
			BonusCoefficient: 0.429,
			ClientBaseDamage: damage.Range(casterLevel),

			ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				spell.CalcAndDealDamage(sim, target, damage.Roll(sim, casterLevel), spell.OutcomeMagicHitAndCrit)
			},
		})
	}
}
