package item_effects

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Kindling Stave is a Forever item with a Forever-authored equip proc. The
// 1.60.1.70707 client binds item 11750 (ItemXItemEffect -> ItemEffect,
// TriggerType 1, equip) to spell 1300781 "Ignition": aura 42 on ProcTypeMask
// 0x10000 (deal harmful spell), ProcChance 100, no SpellProcsPerMinute row,
// ProcCategoryRecovery 10 ms, triggering 1300782: school damage 70 at
// Variance 0.3 (59.5 to 80.5, Wowhead Forever's "60 to 81 Fire damage"), no
// spell-power coefficient. The text says "a chance": the rate is the
// server's, read as one a minute like every unstated "chance on" bonus
// (core.MakeHarmfulSpellProcAura). The "3 times as much damage to Plants"
// half meets no raid target.
const (
	KindlingStave = 11750

	kindlingStaveIgnitionSpellID = 1300782
	kindlingStaveIgnitionDamage  = 70.0
	kindlingStaveIgnitionSpread  = 0.3
	kindlingStaveIgnitionICD     = 10 * time.Millisecond
)

func init() {
	core.NewItemEffect(KindlingStave, func(agent core.Agent) {
		character := agent.GetCharacter()

		low := kindlingStaveIgnitionDamage * (1 - kindlingStaveIgnitionSpread/2)
		high := kindlingStaveIgnitionDamage * (1 + kindlingStaveIgnitionSpread/2)
		ignition := character.RegisterSpell(core.SpellConfig{
			ActionID:         core.ActionID{SpellID: kindlingStaveIgnitionSpellID},
			SpellSchool:      core.SpellSchoolFire,
			DefenseType:      core.DefenseTypeMagic,
			ProcMask:         core.ProcMaskEmpty,
			Flags:            core.SpellFlagNoOnCastComplete | core.SpellFlagPassiveSpell,
			DamageMultiplier: 1,
			ThreatMultiplier: 1,
			ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				spell.CalcAndDealDamage(sim, target, sim.Roll(low, high), spell.OutcomeMagicHitAndCrit)
			},
		})

		core.MakeHarmfulSpellProcAura(&character.Unit, core.HarmfulSpellProc{
			Name:     "Ignition",
			ActionID: core.ActionID{SpellID: 1300781},
			PPM:      core.UnstatedProcsPerMinute,
			ICD:      kindlingStaveIgnitionICD,
			Handler:  func(sim *core.Simulation, target *core.Unit) { ignition.Cast(sim, target) },
		})
	})
}
