package rogue

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// sinisterStrikeLearnLevels are Sinister Strike's eight rank learn levels;
// source: 1.60.1.70009 client spell data ("Sinister Strike", ranks 1-8; the
// level-20 rank-0 ids are internal copies, not player ranks).
var sinisterStrikeLearnLevels = []int{1, 6, 14, 22, 30, 38, 46, 54}

// sinisterStrikeSpellID is Sinister Strike's rank -> spell id, index 0
// unused.
var sinisterStrikeSpellID = [9]int32{0, 1752, 1757, 1758, 1759, 1760, 8621, 11293, 11294}

// sinisterStrikeFlatDamageBonus is Sinister Strike's rank -> flat damage
// bonus, index 0 unused. Only ranks 4, 6, 7 and 8 have a tuned value in
// this file; ranks 1-3 carry rank 4's bonus backward, and rank 5 carries
// rank 4's bonus forward, until real numbers are sourced.
var sinisterStrikeFlatDamageBonus = [9]float64{0, 15, 15, 15, 15, 15, 33, 52, 68}

func (rogue *Rogue) registerSinisterStrikeSpell() {
	rank := core.HighestRankAtLevel(sinisterStrikeLearnLevels, rogue.Level)
	if rank == 0 {
		return
	}

	flatDamageBonus := sinisterStrikeFlatDamageBonus[rank]
	spellID := sinisterStrikeSpellID[rank]

	rogue.SinisterStrike = rogue.RegisterSpell(core.SpellConfig{
		SpellCode:   SpellCode_RogueSinisterStrike,
		ActionID:    core.ActionID{SpellID: spellID},
		SpellSchool: core.SpellSchoolPhysical,
		DefenseType: core.DefenseTypeMelee,
		ProcMask:    core.ProcMaskMeleeMHSpecial,
		Flags:       rogue.builderFlags(),

		EnergyCost: core.EnergyCostOptions{
			Cost:   []float64{45, 42, 40}[rogue.Talents.ImprovedSinisterStrike],
			Refund: 0.8,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: time.Second,
			},
			IgnoreHaste: true,
		},

		CritDamageBonus: rogue.lethality(),

		DamageMultiplier: []float64{1, 1.02, 1.04, 1.06}[rogue.Talents.Aggression],
		ThreatMultiplier: 1,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			rogue.BreakStealth(sim)

			baseDamage := flatDamageBonus + spell.Unit.MHNormalizedWeaponDamage(sim, spell.MeleeAttackPower(target))
			result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)

			if result.Landed() {
				rogue.AddComboPoints(sim, 1, target, spell.ComboPointMetrics())
			} else {
				spell.IssueRefund(sim)
			}
		},
	})
}
