package paladin

import (
	"strconv"
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

// Seal of Command is a spell consisting of:
// - A judgement that has a flat damage roll, and scales with spellpower.
// - A 7ppm on-hit proc with a 1s ICD that deals 70% weapon damage and scales with spellpower.

// Judgement of Command has some unusual behaviour in classic:
// - The judgement operates via a dummy spell, that likely figures out whether to apply
//   half damage if the target is not stunned or not (counter to the tooltip, the base damage only is
//   multiplied by 2 if the target is stunned). These dummy spells are implemented as targetting
//   magic defense type, and have no flags to prevent misses, meaning they roll on spell hit table
//   and can miss. If it succeeds, it calls the "actual" Judgement of Command spell.
// - The actual Judgement of Command has flags to not miss and to avoid block/parry/dodge, but
//   it targets the melee defense type and so crits for double damage.
//   The Seal of Command aura watches for the base Judgement spell, and casts the actual
//   Judgement of Command when it successfully is cast.

// JudgementOfCommandDamage is spellconst/paladin.json's own roll for the
// judgement's ids 20467 through 20966 (rank 5 rolls 339-373 at level 60: a
// centre of 356 at its own level, growing 6.1 a level to level 68), before
// the 0.5 the judgement halves it by unless the target is stunned.
var JudgementOfCommandDamage = [judgementOfCommandRanks + 1]clientdamage.Effect{
	{},
	{Amount: 97, Variance: 0.082474, PerLevel: 5.6, SpellLevel: 20, MaxLevel: 28},
	{Amount: 153, Variance: 0.091503, PerLevel: 6.1, SpellLevel: 30, MaxLevel: 38},
	{Amount: 214, Variance: 0.093458, PerLevel: 5.6, SpellLevel: 40, MaxLevel: 48},
	{Amount: 274, Variance: 0.094891, PerLevel: 6.1, SpellLevel: 50, MaxLevel: 58},
	{Amount: 356, Variance: 0.095506, PerLevel: 6.1, SpellLevel: 60, MaxLevel: 68},
}

const judgementOfCommandRanks = 5

func (paladin *Paladin) registerSealOfCommand() {
	type proc struct {
		spellID int32
	}

	ranks := []struct {
		level        int32
		spellID      int32
		manaCost     float64
		scaleLevel   int32
		proc         proc
		judgeSpellID int32
	}{
		{level: 20, spellID: 20375, manaCost: 65, scaleLevel: 28, proc: proc{spellID: 20424}, judgeSpellID: 20467},
		{level: 30, spellID: 20915, manaCost: 110, scaleLevel: 38, proc: proc{spellID: 20944}, judgeSpellID: 20963},
		{level: 40, spellID: 20918, manaCost: 140, scaleLevel: 48, proc: proc{spellID: 20945}, judgeSpellID: 20964},
		{level: 50, spellID: 20919, manaCost: 180, scaleLevel: 58, proc: proc{spellID: 20946}, judgeSpellID: 20965},
		{level: 60, spellID: 20920, manaCost: 210, scaleLevel: 60, proc: proc{spellID: 20947}, judgeSpellID: 20966},
	}

	ppmm := paladin.AutoAttacks.NewPPMManager(7, core.ProcMaskMelee)

	icd := core.Cooldown{
		Timer:    paladin.NewTimer(),
		Duration: time.Second * 1,
	}

	for i, rank := range ranks {
		rank := rank
		if paladin.Level < rank.level {
			break
		}

		judgeDamage := JudgementOfCommandDamage[i+1]
		casterLevel := int(paladin.Level)

		judgeSpell := paladin.RegisterSpell(core.SpellConfig{
			SpellCode:      SpellCode_PaladinJudgementOfCommand, // used in judgement.go
			ClassSpellMask: PaladinSpellMaskJudgementOfCommand,
			ActionID:       core.ActionID{SpellID: rank.judgeSpellID},
			SpellSchool:    core.SpellSchoolHoly,
			DefenseType:    core.DefenseTypeMelee,
			ProcMask:       core.ProcMaskMeleeMHSpecial,
			Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagNoOnCastComplete,

			// source 1.60.1.70009 client spell data (spellconst/paladin.json):
			// Judgement of Command's spell_level equals the owning Seal
			// of Command rank's own level at every rank. Flagged by
			// paladin.golden.md's "required_level N->0" rows - this
			// SpellConfig never set the field at all.
			RequiredLevel: int(rank.level),

			DamageMultiplier: paladin.getWeaponSpecializationModifier(),
			ThreatMultiplier: 1,
			BonusCoefficient: 0.429,
			ClientBaseDamage: judgeDamage.Range(casterLevel),

			ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				baseDamage := judgeDamage.Roll(sim, casterLevel) * 0.5 // unless stunned

				// Seal of Command requires this spell to act as its intermediary dummy,
				// rolling on the spell hit table. If it succeeds, the actual Judgement of Command rolls on the
				// melee special attack crit/hit table, necessitating two discrete spells.
				// All other judgements are cast directly.
				// Used to decide between spell.OutcomeMeleeSpecialCritOnly and spell.OutcomeAlwaysMiss
				dummyJudgeLanded := paladin.judgement.CalcOutcome(sim, target, paladin.judgement.OutcomeMagicHit).Landed()

				outcomeApplier := core.Ternary(dummyJudgeLanded, spell.OutcomeMeleeSpecialCritOnly, spell.OutcomeAlwaysMiss)
				spell.CalcAndDealDamage(sim, target, baseDamage, outcomeApplier)
			},
		})

		procSpell := paladin.RegisterSpell(core.SpellConfig{
			ClassSpellMask: PaladinSpellMaskSealOfCommandProc,
			ActionID:       core.ActionID{SpellID: rank.proc.spellID},
			SpellSchool:    core.SpellSchoolHoly,
			DefenseType:    core.DefenseTypeMelee,
			ProcMask:       core.ProcMaskMeleeMHSpecial | core.ProcMaskMeleeProc | core.ProcMaskMeleeDamageProc,
			Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagNotAProc,

			DamageMultiplier: 0.7 * paladin.getWeaponSpecializationModifier(),
			ThreatMultiplier: 1,

			BonusCoefficient: 0.29,

			ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				baseDamage := spell.Unit.MHWeaponDamage(sim, spell.MeleeAttackPower(target))
				result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMeleeSpecialHitAndCrit)

				core.StartDelayedAction(sim, core.DelayedActionOptions{
					DoAt: sim.CurrentTime + core.SpellBatchWindow,
					OnAction: func(s *core.Simulation) {
						spell.DealDamage(sim, result)
					},
				})
			},
		})

		aura := paladin.RegisterAura(core.Aura{
			Label:    "Seal of Command" + paladin.Label + strconv.Itoa(i+1),
			ActionID: core.ActionID{SpellID: rank.spellID},
			Duration: time.Second * 30,
			OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
				if !result.Landed() {
					return
				}

				if spell.ProcMask.Matches(core.ProcMaskMeleeWhiteHit) {
					if icd.IsReady(sim) && ppmm.Proc(sim, spell.ProcMask, "seal of command") {
						icd.Use(sim)
						procSpell.Cast(sim, result.Target)
					}
				}
			},
		})

		paladin.aurasSoC = append(paladin.aurasSoC, aura)
		paladin.sealOfCommandProc = procSpell

		paladin.sealOfCommand = paladin.RegisterSpell(core.SpellConfig{
			ClassSpellMask: PaladinSpellMaskSealOfCommandCast,
			ActionID:       aura.ActionID,
			SpellSchool:    core.SpellSchoolHoly,
			Flags:          core.SpellFlagAPL,

			RequiredLevel: int(rank.level),
			Rank:          i + 1,

			// aura's own Duration (30s) matches the client's
			// duration_ms (30000) for this cast's own SpellID at every
			// rank (spellconst/paladin.json) - wiring it through lets
			// compare.go's engineDuration see it instead of reporting
			// 0 for a self-buff that does exist.
			RelatedSelfBuff: aura,

			ManaCost: core.ManaCostOptions{
				FlatCost:   rank.manaCost - paladin.getLibramSealCostReduction(),
				Multiplier: paladin.benediction(),
			},
			Cast: core.CastConfig{
				DefaultCast: core.Cast{
					GCD: core.GCDDefault,
				},
			},

			ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
				paladin.applySeal(aura, judgeSpell, spell, sim)
			},
		})

		paladin.spellsJoC = append(paladin.spellsJoC, judgeSpell)
	}
}
