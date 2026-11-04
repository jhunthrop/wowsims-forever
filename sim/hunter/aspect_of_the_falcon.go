package hunter

import (
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

// Aspect of the Falcon (1.60.1.70009 spell 469145, level 60, 120 mana) is
// the client's "melee and ranged Attack Power by the same amount as their
// highest rank of Aspect of the Hawk": effect 0 is aura 99 (melee attack
// power) and effect 1 aura 124 (ranged attack power), both stating no
// amount of their own. Effect 2 is its own proc (aura 42, Quick Strikes
// 469144): 1.60.1.70755 SpellAuraOptions 239200 gives ProcChance 5 and
// ProcTypeMask 20 (0x4 melee auto done, 0x10 melee-class spell done), so a
// landed melee auto or melee strike opens it at a flat 5%. Falcon's
// SpellClassOptions mask is 0, so Deadly Aspects' chance mods (Hawk's and
// Beast's masks only) never raise it, and no ranged attack procs it.
const (
	aspectOfTheFalconSpellID    int32 = 469145
	aspectOfTheFalconLevel      int   = 60
	aspectOfTheFalconMana             = 120.0
	aspectOfTheFalconAuraLabel        = "Aspect of the Falcon"
	aspectOfTheFalconProcChance       = 0.05
)

func (hunter *Hunter) registerAspectOfTheFalconSpell() {
	if hunter.Level < int32(aspectOfTheFalconLevel) {
		return
	}

	attackPower := hunter.getMaxAspectOfTheHawkAttackPower(hunter.getMaxHawkRank()) * hunter.AspectOfTheHawkAPMultiplier
	actionID := core.ActionID{SpellID: aspectOfTheFalconSpellID}
	quickStrikes := hunter.createQuickStrikesAura()

	aura := hunter.GetOrRegisterAura(core.Aura{
		Label:    aspectOfTheFalconAuraLabel,
		ActionID: actionID,
		Duration: core.NeverExpires,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.AddStatsDynamic(sim, stats.Stats{stats.AttackPower: attackPower, stats.RangedAttackPower: attackPower})
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.AddStatsDynamic(sim, stats.Stats{stats.AttackPower: -attackPower, stats.RangedAttackPower: -attackPower})
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if result.Landed() && spell.ProcMask.Matches(core.ProcMaskMelee) && sim.Proc(aspectOfTheFalconProcChance, "Aspect of the Falcon") {
				quickStrikes.Activate(sim)
			}
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
