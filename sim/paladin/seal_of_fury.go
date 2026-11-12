package paladin

import (
	"strconv"
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

// Seal of Fury is Forever's tanking Seal, learned from level 10 on the
// Protection skill line. The client states it as three spells per rank
// (spellconst/paladin.json, ids in sealOfFuryRanks):
//
//   - the seal itself, a 30 s self aura whose effects are a dummy value
//     (the weapon-speed scaling Seal of Righteousness uses, which Seal of
//     Fury does not read: "damage regardless of weapon speed"), a 50 and
//     the id of the judgement;
//   - the proc, a Holy school-damage effect with a flat amount and a 0.1
//     spell-power coefficient, fired by a landed white melee hit;
//   - the judgement, a Holy school-damage effect with a 0.45 coefficient.
//
// Blizzard's Deep Dive recap adds what the tables do not spell out: the
// seal carries "a small absorb worth half the damage with a shield
// equipped" (the 50 above), and Judging it taunts from 10 yards (a taunt
// is no threat model this single-tank engine has). Judgement does not
// consume the seal in Forever.

// sealOfFuryProcDamage is each rank's flat proc damage, the client's own
// effect 2 amount on the proc spells. The client gives no per-level growth
// on the proc (it is "regardless of weapon speed" and flat), so it is one
// number per rank.
var sealOfFuryProcDamage = [sealOfFuryRankCount + 1]clientdamage.Effect{
	{},
	{Amount: 6, SpellLevel: 1},
	{Amount: 9, SpellLevel: 1},
	{Amount: 14, SpellLevel: 1},
	{Amount: 19, SpellLevel: 1},
	{Amount: 25, SpellLevel: 1},
	{Amount: 32, SpellLevel: 1},
	{Amount: 35, SpellLevel: 1},
}

// JudgementOfFuryDamage is spellconst/paladin.json's roll for the
// judgement's ids 1311650 through 20414 (rank 7 rolls 146-160 at level 60,
// a centre of 153 at its own level, growing 3.69 a level to level 64).
var JudgementOfFuryDamage = [sealOfFuryRankCount + 1]clientdamage.Effect{
	{},
	{Amount: 23, Variance: 0.1, PerLevel: 1.71, SpellLevel: 10, MaxLevel: 16},
	{Amount: 37, Variance: 0.1, PerLevel: 2.16, SpellLevel: 18, MaxLevel: 24},
	{Amount: 54, Variance: 0.1, PerLevel: 2.52, SpellLevel: 25, MaxLevel: 31},
	{Amount: 74, Variance: 0.09756097, PerLevel: 2.79, SpellLevel: 34, MaxLevel: 40},
	{Amount: 96, Variance: 0.09756097, PerLevel: 3.42, SpellLevel: 42, MaxLevel: 48},
	{Amount: 123, Variance: 0.087591, PerLevel: 3.69, SpellLevel: 50, MaxLevel: 56},
	{Amount: 153, Variance: 0.087591, PerLevel: 3.69, SpellLevel: 58, MaxLevel: 64},
}

const (
	sealOfFuryRankCount = 7

	// The client's sp_coefficient on the proc and on the judgement.
	sealOfFuryProcCoefficient  = 0.10
	judgementOfFuryCoefficient = 0.429 // client 1.60.1.70425 (0.45 before)

	sealOfFuryDuration = 30 * time.Second

	// sealOfFuryAbsorbShare is the seal's effect 1 (50): "a small absorb
	// worth half the damage with a shield equipped" (Blizzard's Deep Dive
	// recap), read as half of the proc's damage.
	sealOfFuryAbsorbShare = 0.5

	// sealOfFuryAbsorbDuration is the shield's own row (sealOfFuryShieldID,
	// duration_ms 10000). The launch logs spend it on the next hit taken,
	// within a second or two; it only matters if it were shorter than a
	// swing.
	sealOfFuryAbsorbDuration = 10 * time.Second

	// sealOfFuryProcSpellLevel is the proc spells' own spell_level.
	sealOfFuryProcSpellLevel = 1

	// sealOfFuryShieldID is the shield's own spell, "Light's Fury"
	// (spellconst/paladin.json: effect 6, aura 69 school absorb of every
	// school, 10 s, no rank). The launch logs (foreverlogs.gg 2849, a
	// Seal of Fury paladin at 18: 186 applications, 124 absorbs of 4 off a
	// proc of 9 and 8 off a crit of 18) name it on every Seal of Fury
	// proc; the engine had named it with an unused id before.
	sealOfFuryShieldID   = 1310927
	sealOfFuryShieldName = "Light's Fury"
)

type sealOfFuryRank struct {
	level      int32
	spellID    int32
	manaCost   float64
	procID     int32
	judgeSpell int32
}

// sealOfFuryRanks: the seal, its proc and its judgement per rank. The
// seal ids are the trainable's (trainables/paladin.json), the proc and
// judgement ids come from each seal's trigger_spell and effect 2.
var sealOfFuryRanks = [sealOfFuryRankCount]sealOfFuryRank{
	{level: 10, spellID: 1311649, manaCost: 40, procID: 1311647, judgeSpell: 1311650},
	{level: 18, spellID: 1311656, manaCost: 60, procID: 1311654, judgeSpell: 1311655},
	{level: 26, spellID: 20163, manaCost: 90, procID: 20231, judgeSpell: 20183}, // 26 since client 1.60.1.70425
	{level: 34, spellID: 20419, manaCost: 120, procID: 20415, judgeSpell: 20411},
	{level: 42, spellID: 20421, manaCost: 140, procID: 20416, judgeSpell: 20412},
	{level: 50, spellID: 20422, manaCost: 170, procID: 20417, judgeSpell: 20413},
	{level: 58, spellID: 20423, manaCost: 200, procID: 20418, judgeSpell: 20414},
}

func (paladin *Paladin) registerSealOfFury() {
	shield := paladin.newSealOfFuryShield()
	for i, rank := range sealOfFuryRanks {
		if paladin.Level < rank.level {
			break
		}
		paladin.registerSealOfFuryRank(i+1, rank, shield)
	}
}

// newSealOfFuryShield is the one absorb every rank shares: a new proc
// replaces the shield, whatever rank is active.
func (paladin *Paladin) newSealOfFuryShield() *damageAbsorb {
	aura := paladin.RegisterAura(core.Aura{
		Label:    sealOfFuryShieldName,
		ActionID: core.ActionID{SpellID: sealOfFuryShieldID},
		Duration: sealOfFuryAbsorbDuration,
	})
	return newDamageAbsorb(&paladin.Unit, aura, nil)
}

func (paladin *Paladin) registerSealOfFuryRank(rank int, config sealOfFuryRank, shield *damageAbsorb) {
	judgeSpell := paladin.registerJudgementOfFury(rank, config)
	procSpell := paladin.registerSealOfFuryProc(rank, config, shield)

	effect := &sealEffect{
		family: "Seal of Fury",
		apply: func(sim *core.Simulation, target *core.Unit) {
			procSpell.Cast(sim, target)
		},
	}
	aura := paladin.RegisterAura(core.Aura{
		Label:    "Seal of Fury" + paladin.Label + strconv.Itoa(rank),
		ActionID: core.ActionID{SpellID: config.spellID},
		Duration: sealOfFuryDuration,

		OnSpellHitDealt: func(_ *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if isSealSwing(spell, result) {
				effect.tryActivate(sim, result.Target, spell.ProcMask)
			}
		},
	})

	paladin.aurasSoF = append(paladin.aurasSoF, aura)
	paladin.registerSealEffect(aura, effect)
	paladin.spellsJoF = append(paladin.spellsJoF, judgeSpell)

	paladin.sealOfFury = paladin.RegisterSpell(core.SpellConfig{
		ClassSpellMask: PaladinSpellMaskSealOfFuryCast,
		ActionID:       aura.ActionID,
		SpellSchool:    core.SpellSchoolHoly,
		Flags:          core.SpellFlagAPL,

		RequiredLevel: int(config.level),
		Rank:          rank,

		RelatedSelfBuff: aura,

		ManaCost: core.ManaCostOptions{
			FlatCost:   config.manaCost,
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
}

// registerJudgementOfFury registers the judgement Seal of Fury resolves
// to. Like Judgement of Righteousness it rolled the melee table with no
// active defense (SpellCategories DefenseType 2 and SpellMisc Attributes_0
// 0x200000 on every rank, 1.60.1.70707) until the 11 November 2026 hotfix
// made every Judgement a ranged attack (judgement.go).
func (paladin *Paladin) registerJudgementOfFury(rank int, config sealOfFuryRank) *core.Spell {
	damage := JudgementOfFuryDamage[rank]
	casterLevel := int(paladin.Level)

	return paladin.RegisterSpell(core.SpellConfig{
		SpellCode:      SpellCode_PaladinJudgementOfFury,
		ClassSpellMask: PaladinSpellMaskJudgementOfFury,
		ActionID:       core.ActionID{SpellID: config.judgeSpell},
		SpellSchool:    core.SpellSchoolHoly,
		DefenseType:    judgementDefenseType,
		CastType:       judgementCastType,
		ProcMask:       judgementProcMask,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagSuppressWeaponProcs | core.SpellFlagSuppressEquipProcs | core.SpellFlagBinary,

		// The judgement's own row: rank 3 is learned with a level-26 seal
		// but the client files the judgement at 25 (1.60.1.70425).
		RequiredLevel: damage.SpellLevel,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		BonusCoefficient: judgementOfFuryCoefficient,
		ClientBaseDamage: damage.Range(casterLevel),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.CalcAndDealDamage(sim, target, damage.Roll(sim, casterLevel), spell.OutcomeRangedHitAndCritNoBlock)
		},
	})
}

// registerSealOfFuryProc registers the proc spell a landed white hit
// fires: flat Holy damage with the client's coefficient, and, with a
// shield equipped, the absorb worth half of it.
func (paladin *Paladin) registerSealOfFuryProc(rank int, config sealOfFuryRank, shield *damageAbsorb) *core.Spell {
	damage := sealOfFuryProcDamage[rank]
	casterLevel := int(paladin.Level)

	return paladin.RegisterSpell(core.SpellConfig{
		SpellCode:      SpellCode_PaladinSealOfFuryProc,
		ClassSpellMask: PaladinSpellMaskSealOfFuryProc,
		ActionID:       core.ActionID{SpellID: config.procID},
		SpellSchool:    core.SpellSchoolHoly,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagSuppressEquipProcs,

		RequiredLevel: sealOfFuryProcSpellLevel,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		BonusCoefficient: sealOfFuryProcCoefficient,
		ClientBaseDamage: damage.Range(casterLevel),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcAndDealDamage(sim, target, damage.Roll(sim, casterLevel), spell.OutcomeMeleeSpecialCritOnly)
			if result.Landed() && paladin.PseudoStats.CanBlock {
				shield.Grant(sim, spell, sealOfFuryAbsorbShare*result.Damage)
			}
		},
	})
}
