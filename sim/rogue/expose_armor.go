package rogue

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// exposeArmorLearnLevels are Expose Armor's five rank learn levels; source:
// 1.60.1.70009 client spell data ("Expose Armor", ranks 1-5).
var exposeArmorLearnLevels = []int{14, 26, 36, 46, 56}

// exposeArmorSpellID is Expose Armor's rank -> spell id, index 0 unused.
var exposeArmorSpellID = [6]int32{0, 8647, 8649, 8650, 11197, 11198}

// exposeArmorArpenPerCombo is Expose Armor's rank -> armor pen per combo
// point, index 0 unused. Rank 2 has no tuned value in this file; it
// carries rank 1's number forward until a real one is sourced.
var exposeArmorArpenPerCombo = [6]float64{0, 80, 80, 210, 275, 340}

func (rogue *Rogue) registerExposeArmorSpell() {
	rogue.ExposeArmorAuras = rogue.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
		return core.ExposeArmorAura(target, rogue.Talents.ImprovedExposeArmor)
	})

	rank := core.HighestRankAtLevel(exposeArmorLearnLevels, rogue.Level)
	if rank == 0 {
		return
	}

	spellID := exposeArmorSpellID[rank]
	arpenPerCombo := exposeArmorArpenPerCombo[rank]

	arpenPerCombo *= []float64{1, 1.25, 1.5}[rogue.Talents.ImprovedExposeArmor]

	// share ExtraCastCondition() state with ApplyEffects()
	var arpen float64
	var eaAura *core.Aura

	rogue.ExposeArmor = rogue.RegisterSpell(core.SpellConfig{
		SpellCode:    SpellCode_RogueExposeArmor,
		ActionID:     core.ActionID{SpellID: spellID},
		SpellSchool:  core.SpellSchoolPhysical,
		DefenseType:  core.DefenseTypeMelee,
		ProcMask:     core.ProcMaskMeleeMHSpecial,
		Flags:        rogue.finisherFlags(),
		MetricSplits: 6,

		EnergyCost: core.EnergyCostOptions{
			Cost:   25,
			Refund: 0,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: time.Second,
			},
			IgnoreHaste: true,
			ModifyCast: func(sim *core.Simulation, spell *core.Spell, cast *core.Cast) {
				spell.SetMetricsSplit(spell.Unit.ComboPoints())
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			if rogue.ComboPoints() == 0 {
				return false
			}

			eaAura = rogue.ExposeArmorAuras.Get(target)
			arpen = float64(rogue.ComboPoints()) * arpenPerCombo

			if curActive := eaAura.ExclusiveEffects[0].Category.GetActiveEffect(); curActive != nil {
				return arpen >= curActive.Priority
			}
			return true
		},

		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			rogue.BreakStealth(sim)

			result := spell.CalcOutcome(sim, target, spell.OutcomeMeleeSpecialHit)
			if result.Landed() {
				eaAura.ExclusiveEffects[0].Priority = arpen
				eaAura.Activate(sim)
				rogue.SpendComboPoints(sim, spell)
			} else {
				spell.IssueRefund(sim)
			}
			spell.DealOutcome(sim, result)
		},

		RelatedAuras: []core.AuraArray{rogue.ExposeArmorAuras},
	})
	rogue.Finishers = append(rogue.Finishers, rogue.ExposeArmor)
}
