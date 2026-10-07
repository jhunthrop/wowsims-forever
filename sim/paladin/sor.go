package paladin

import (
	"strconv"
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

// JudgementOfRighteousnessDamage is spellconst/paladin.json's own roll for
// the judgement's ids 20187 through 20286 (rank 8 rolls 162-178 at level 60:
// a centre of 170 at its own level, growing 4.1 a level to level 64). The
// client states the same 0.5 spell-power coefficient on every rank; the
// downranked 0.144-0.462 coefficients the Classic port carried are gone.
var JudgementOfRighteousnessDamage = [judgementOfRighteousnessRanks + 1]clientdamage.Effect{
	{},
	{Amount: 15, Variance: 0.077, PerLevel: 1.8, SpellLevel: 1, MaxLevel: 7},
	{Amount: 26, Variance: 0.076923, PerLevel: 1.9, SpellLevel: 10, MaxLevel: 16},
	{Amount: 41, Variance: 0.097561, PerLevel: 2.4, SpellLevel: 18, MaxLevel: 24},
	{Amount: 60, Variance: 0.1, PerLevel: 2.8, SpellLevel: 26, MaxLevel: 32},
	{Amount: 82, Variance: 0.097561, PerLevel: 3.1, SpellLevel: 34, MaxLevel: 40},
	{Amount: 107, Variance: 0.093458, PerLevel: 3.8, SpellLevel: 42, MaxLevel: 48},
	{Amount: 137, Variance: 0.087591, PerLevel: 4.1, SpellLevel: 50, MaxLevel: 56},
	{Amount: 170, Variance: 0.094118, PerLevel: 4.1, SpellLevel: 58, MaxLevel: 64},
}

const (
	judgementOfRighteousnessRanks       = 8
	judgementOfRighteousnessCoefficient = 0.5
)

func (paladin *Paladin) registerSealOfRighteousness() {
	type proc struct {
		spellID int32
		value   float64
		scale   float64
		coeff   float64
	}

	var ranks = []struct {
		level        int32
		spellID      int32
		manaCost     float64
		scaleLevel   int32
		proc         proc
		judgeSpellID int32
	}{
		{level: 1, spellID: 20154, manaCost: 20, scaleLevel: 7, proc: proc{spellID: 25742, value: 108, scale: 18, coeff: 0.029}, judgeSpellID: 20187},
		{level: 10, spellID: 20287, manaCost: 40, scaleLevel: 16, proc: proc{spellID: 25740, value: 216, scale: 17, coeff: 0.063}, judgeSpellID: 20280},
		{level: 18, spellID: 20288, manaCost: 60, scaleLevel: 24, proc: proc{spellID: 25739, value: 352, scale: 23, coeff: 0.093}, judgeSpellID: 20281},
		{level: 26, spellID: 20289, manaCost: 90, scaleLevel: 32, proc: proc{spellID: 25738, value: 541, scale: 31, coeff: 0.1}, judgeSpellID: 20282},
		{level: 34, spellID: 20290, manaCost: 120, scaleLevel: 40, proc: proc{spellID: 25737, value: 785, scale: 37, coeff: 0.1}, judgeSpellID: 20283},
		{level: 42, spellID: 20291, manaCost: 140, scaleLevel: 48, proc: proc{spellID: 25736, value: 1082, scale: 41, coeff: 0.1}, judgeSpellID: 20284},
		{level: 50, spellID: 20292, manaCost: 170, scaleLevel: 56, proc: proc{spellID: 25735, value: 1407, scale: 47, coeff: 0.1}, judgeSpellID: 20285},
		{level: 58, spellID: 20293, manaCost: 200, scaleLevel: 60, proc: proc{spellID: 25713, value: 1786, scale: 47, coeff: 0.1}, judgeSpellID: 20286},
	}

	improvedSoR := paladin.improvedSoR()

	for i, rank := range ranks {
		rank := rank
		if paladin.Level < rank.level {
			break
		}

		/*
		 * Seal of Righteousness is a Spell/Aura that when active makes the paladin capable of procing
		 * two different SpellIDs depending on a paladin's casted spell or melee swing.
		 *
		 * (Judgement of Righteousness):
		 *   - Deals flat damage that is affected by Improved SoR talent, and
		 *     has a spellpower scaling that is unaffected by that talent.
		 *   - Targets magic defense and rolls to hit and crit.
		 *
		 * (Seal of Righteousness):
		 *   - Procs from white hits.
		 *   - Cannot miss or be dodged/parried/blocked if the underlying white hit lands.
		 *   - Deals damage that is a function of weapon speed, and spellpower.
		 *   - Has 0.85 scale factor on base damage if using 1h, 1.2 if using 2h.
		 *   - Calculates damage including spellpower scaling but ignoring damage multipliers,
		 *      then feeds that value as base damage into the proc spell.
		 */

		judgeDamage := JudgementOfRighteousnessDamage[i+1]
		casterLevel := int(paladin.Level)

		judgeSpell := paladin.RegisterSpell(core.SpellConfig{
			SpellCode:      SpellCode_PaladinJudgementOfRighteousness,
			ClassSpellMask: PaladinSpellMaskJudgementOfRighteousness,
			ActionID:       core.ActionID{SpellID: rank.judgeSpellID},
			SpellSchool:    core.SpellSchoolHoly,
			DefenseType:    core.DefenseTypeMagic,
			ProcMask:       core.ProcMaskSpellDamage,
			Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagSuppressWeaponProcs | core.SpellFlagSuppressEquipProcs | core.SpellFlagBinary,

			// source 1.60.1.70009 client spell data (spellconst/paladin.json):
			// Judgement of Righteousness's spell_level equals the
			// owning Seal of Righteousness rank's own level at every
			// rank. Flagged by paladin.golden.md's "required_level
			// N->0" rows - this SpellConfig never set the field at all.
			RequiredLevel: int(rank.level),

			DamageMultiplier: 1,
			ThreatMultiplier: 1,

			BonusCoefficient: judgementOfRighteousnessCoefficient,
			ClientBaseDamage: judgeDamage.Range(casterLevel),

			ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				baseDamage := judgeDamage.Roll(sim, casterLevel) * improvedSoR
				spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)
			},
		})

		value := 0.01 * (rank.proc.value + rank.proc.scale*float64(min(paladin.Level, rank.scaleLevel)-rank.level))

		coeff := rank.proc.coeff
		damage := value * 0.85 * paladin.MainHand().SwingSpeed
		if paladin.has2hEquipped() {
			coeff = rank.proc.coeff * 1.1 // from testing in SoD
			damage = value * 1.2 * paladin.MainHand().SwingSpeed
		}

		procSpell := paladin.RegisterSpell(core.SpellConfig{
			ClassSpellMask: PaladinSpellMaskSealOfRighteousnessProc,
			ActionID:       core.ActionID{SpellID: rank.proc.spellID},
			SpellSchool:    core.SpellSchoolHoly,
			DefenseType:    core.DefenseTypeMelee,
			ProcMask:       core.ProcMaskMeleeMHSpecial,                                   //changed to ProcMaskMeleeMHSpecial, to allow procs from weapons/oils which do proc from SoR,
			Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagSuppressEquipProcs, // but Wild Strikes does not proc, nor equip procs

			// source 1.60.1.70009 client spell data: every Seal of
			// Righteousness proc id (25713/2573x/2574x) carries
			// spell_level 1, independent of the owning rank. Flagged by
			// paladin.golden.md's "required_level 1->0" rows.
			RequiredLevel: 1,

			//BonusCritRating: paladin.holyCrit(), // TODO to be tested, but unlikely

			DamageMultiplier: paladin.getWeaponSpecializationModifier(),
			ThreatMultiplier: 1,

			BonusCoefficient: coeff,

			ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				// effectively scales with coeff x 2, and damage dealt multipliers affect half the damage taken bonus
				baseDamage := damage*improvedSoR + spell.BonusCoefficient*(spell.GetBonusDamage(target)+target.GetSchoolBonusDamageTaken(spell))
				spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeSpecialCritOnly)
			},
		})

		aura := paladin.RegisterAura(core.Aura{
			Label:    "Seal of Righteousness" + paladin.Label + strconv.Itoa(i+1),
			ActionID: core.ActionID{SpellID: rank.spellID},
			Duration: time.Second * 30,

			OnSpellHitDealt: func(_ *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
				if !result.Landed() {
					return
				}
				if spell.ProcMask.Matches(core.ProcMaskMeleeWhiteHit) {
					procSpell.Cast(sim, result.Target)
				}
			},
		})

		paladin.aurasSoR = append(paladin.aurasSoR, aura)
		paladin.sealOfRighteousnessProc = procSpell

		paladin.sealOfRighteousness = paladin.RegisterSpell(core.SpellConfig{
			ClassSpellMask: PaladinSpellMaskSealOfRighteousnessCast,
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

		paladin.spellsJoR = append(paladin.spellsJoR, judgeSpell)
	}
}
