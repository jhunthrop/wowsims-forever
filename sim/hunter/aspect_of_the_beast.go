package hunter

import (
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

// Aspect of the Beast is Forever's trainable melee aspect (1.60.1.70707 spells
// 13161, 1299445, 1299446, 1299447 at levels 30, 40, 50 and 60): aura 120
// (untrackable), aura 99 melee attack power 50/70/90/110 and a proc trigger to
// Quick Strikes 1299448 whose chance comes only from Deadly Aspects ("While
// Aspect of the Beast is active, all melee auto attacks have a ...% chance").
// It costs 50/70/90/110 mana. Wowhead Forever 1299447: "increasing Melee
// Attack Power by 110". Only one Aspect can be active.
const aspectOfTheBeastAuraLabel = "Aspect of the Beast"

// aspectOfTheBeastAttackPower is each rank's melee attack power (the aura 99
// base points), index 0 unused.
var aspectOfTheBeastAttackPower = [AspectOfTheBeastRanks + 1]float64{0, 50, 70, 90, 110}

func (hunter *Hunter) registerAspectOfTheBeastSpell() {
	rank := core.HighestRankAtLevel(AspectOfTheBeastLevel[1:], hunter.Level)
	if rank == 0 {
		return
	}

	attackPower := aspectOfTheBeastAttackPower[rank]
	actionID := core.ActionID{SpellID: AspectOfTheBeastSpellId[rank]}

	aura := hunter.GetOrRegisterAura(core.Aura{
		Label:    aspectOfTheBeastAuraLabel,
		ActionID: actionID,
		Duration: core.NeverExpires,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.AddStatsDynamic(sim, stats.Stats{stats.AttackPower: attackPower})
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.AddStatsDynamic(sim, stats.Stats{stats.AttackPower: -attackPower})
		},
	})
	aura.NewExclusiveEffect("Aspect", true, core.ExclusiveEffect{})

	hunter.AspectOfTheBeast = hunter.RegisterSpell(core.SpellConfig{
		SpellCode:     SpellCode_HunterAspectOfTheBeast,
		ActionID:      actionID,
		SpellSchool:   core.SpellSchoolNature,
		Flags:         core.SpellFlagAPL,
		RequiredLevel: AspectOfTheBeastLevel[rank],
		Rank:          rank,

		ManaCost: core.ManaCostOptions{
			FlatCost: AspectOfTheBeastManaCost[rank],
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return !aura.IsActive()
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			aura.Activate(sim)
		},

		RelatedSelfBuff: aura,
	})
}
