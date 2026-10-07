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

func (rogue *Rogue) registerSinisterStrikeSpell() {
	rank := core.HighestRankAtLevel(sinisterStrikeLearnLevels, rogue.Level)
	if rank == 0 {
		return
	}

	flatDamage := SinisterStrikeDamage[rank]
	casterLevel := int(rogue.Level)
	spellID := sinisterStrikeSpellID[rank]

	rogue.SinisterStrike = rogue.RegisterSpell(core.SpellConfig{
		SpellCode:     SpellCode_RogueSinisterStrike,
		ActionID:      core.ActionID{SpellID: spellID},
		SpellSchool:   core.SpellSchoolPhysical,
		DefenseType:   core.DefenseTypeMelee,
		ProcMask:      core.ProcMaskMeleeMHSpecial,
		Flags:         rogue.builderFlags() | SpellFlagColdBlooded,
		RequiredLevel: sinisterStrikeLearnLevels[rank-1],

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

		DamageMultiplier: 1 + rogue.aggressionBonus(),
		ThreatMultiplier: 1,
		BonusCoefficient: 1,
		ClientBaseDamage: flatDamage.Range(casterLevel),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			rogue.BreakStealth(sim)

			baseDamage := flatDamage.Roll(sim, casterLevel) + spell.Unit.MHNormalizedWeaponDamage(sim, spell.MeleeAttackPower(target))
			result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)

			if result.Landed() {
				rogue.AddComboPoints(sim, 1, target, spell.ComboPointMetrics())
			} else {
				spell.IssueRefund(sim)
			}
		},
	})
}
