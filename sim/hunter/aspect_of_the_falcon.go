package hunter

import (
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

// Aspect of the Falcon (1.60.1.70009 spell 469145, level 60, 120 mana) is
// the client's "melee and ranged Attack Power by the same amount as their
// highest rank of Aspect of the Hawk": effect 0 is aura 99 (melee attack
// power) and effect 1 aura 124 (ranged attack power), both stating no
// amount of their own. The text's Improved Aspect of the Hawk interaction
// has nothing to act on, that talent is not in the client's trees.
const (
	aspectOfTheFalconSpellID int32 = 469145
	aspectOfTheFalconLevel   int   = 60
	aspectOfTheFalconMana          = 120.0
)

func (hunter *Hunter) registerAspectOfTheFalconSpell() {
	if hunter.Level < int32(aspectOfTheFalconLevel) {
		return
	}

	attackPower := hunter.getMaxAspectOfTheHawkAttackPower(hunter.getMaxHawkRank()) * hunter.AspectOfTheHawkAPMultiplier
	actionID := core.ActionID{SpellID: aspectOfTheFalconSpellID}

	aura := hunter.GetOrRegisterAura(core.Aura{
		Label:    "Aspect of the Falcon",
		ActionID: actionID,
		Duration: core.NeverExpires,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.AddStatsDynamic(sim, stats.Stats{stats.AttackPower: attackPower, stats.RangedAttackPower: attackPower})
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.AddStatsDynamic(sim, stats.Stats{stats.AttackPower: -attackPower, stats.RangedAttackPower: -attackPower})
		},
	})
	aura.NewExclusiveEffect("Aspect", true, core.ExclusiveEffect{})

	hunter.AspectOfTheFalcon = hunter.RegisterSpell(core.SpellConfig{
		SpellCode:     SpellCode_HunterAspectOfTheFalcon,
		ActionID:      actionID,
		SpellSchool:   core.SpellSchoolNature,
		Flags:         core.SpellFlagAPL,
		RequiredLevel: aspectOfTheFalconLevel,

		ManaCost: core.ManaCostOptions{
			FlatCost: aspectOfTheFalconMana,
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
