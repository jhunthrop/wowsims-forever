package paladin

import (
	"strconv"
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

// Seal of Light and Seal of Wisdom, read from the 1.60.1.70009 client
// (data/builds/1.60.1.70009/raw).
//
// Seal of Light, ranks 1-4 (cast spells 20165 / 20347 / 20348 / 20349, spell
// levels 30 / 40 / 50 / 60, 110 / 140 / 180 / 210 mana, 1.5 s GCD, 30 s):
// effect 0 is aura 42 (proc trigger) with ProcChance 100 and ProcTypeMask_0
// 20 (a melee auto attack or a melee ability); the heal is spells 20167 /
// 20333 / 20334 / 20340, effect 10 (heal), base points 39 / 53 / 76 / 94,
// no coefficient. Effect 2 is the judgement, aura 4 (dummy) naming spells
// 20185 / 20344 / 20345 / 20346 (Judgement of Light, 40 s).
//
// Seal of Wisdom, ranks 1-3 (cast spells 20166 / 20356 / 20357, spell levels
// 38 / 48 / 58, 135 / 170 / 200 mana, 1.5 s GCD, 30 s): the same aura 42, the
// mana is spells 20168 / 20350 / 20351, effect 30 (energize), base points
// 50 / 71 / 90. Its judgement is spells 20186 / 20354 / 20355 (Judgement of
// Wisdom, 40 s).
//
// The debuffs those judgements lay are core.JudgementOfLightAura and
// core.JudgementOfWisdomAura, the same auras the raid debuffs make
// permanent.

// UtilitySealProcsPerMinute is how often a melee attack triggers Seal of
// Light or Seal of Wisdom. Assumption, not a client row: the auras state
// ProcChance 100 with no procs-per-minute row (Seal of Command states the
// same 100 and the engine gives it 7 a minute from the server's script), so
// this is the 15 a minute vanilla's seals ran on.
const UtilitySealProcsPerMinute = 15.0

// utilitySealRank is one rank of Seal of Light or Seal of Wisdom.
type utilitySealRank struct {
	castID      int32
	level       int32
	manaCost    float64
	judgementID int32
	// procID is the spell that carries the amount (the heal or the mana).
	procID int32
	// amount is the heal or the mana one trigger returns.
	amount float64
}

// utilitySeal is what Light and Wisdom differ in; the rest of the seal is
// shared.
type utilitySeal struct {
	name  string
	mask  uint64
	ranks []utilitySealRank
	// judgementAura lays the seal's judgement on a target.
	judgementAura func(target *core.Unit) *core.Aura
	// newProc says what one trigger does for a rank, given the PPM label.
	newProc func(paladin *Paladin, rank utilitySealRank) func(sim *core.Simulation)
}

var sealOfLight = utilitySeal{
	name: "Seal of Light",
	mask: PaladinSpellMaskSealOfLightCast,
	ranks: []utilitySealRank{
		{castID: 20165, level: 30, manaCost: 110, judgementID: 20185, procID: 20167, amount: 39},
		{castID: 20347, level: 40, manaCost: 140, judgementID: 20344, procID: 20333, amount: 53},
		{castID: 20348, level: 50, manaCost: 180, judgementID: 20345, procID: 20334, amount: 76},
		{castID: 20349, level: 60, manaCost: 210, judgementID: 20346, procID: 20340, amount: 94},
	},
	judgementAura: core.JudgementOfLightAura,
	newProc:       newSealOfLightProc,
}

var sealOfWisdom = utilitySeal{
	name: "Seal of Wisdom",
	mask: PaladinSpellMaskSealOfWisdomCast,
	ranks: []utilitySealRank{
		{castID: 20166, level: 38, manaCost: 135, judgementID: 20186, procID: 20168, amount: 50},
		{castID: 20356, level: 48, manaCost: 170, judgementID: 20354, procID: 20350, amount: 71},
		{castID: 20357, level: 58, manaCost: 200, judgementID: 20355, procID: 20351, amount: 90},
	},
	judgementAura: core.JudgementOfWisdomAura,
	newProc:       newSealOfWisdomProc,
}

func (paladin *Paladin) registerUtilitySeals() {
	paladin.registerUtilitySeal(sealOfLight)
	paladin.registerUtilitySeal(sealOfWisdom)
}

func (paladin *Paladin) registerUtilitySeal(seal utilitySeal) {
	ppmm := paladin.AutoAttacks.NewPPMManager(UtilitySealProcsPerMinute, core.ProcMaskMelee)

	for i, rank := range seal.ranks {
		if paladin.Level < rank.level {
			break
		}

		judgementDebuffs := paladin.NewEnemyAuraArray(seal.judgementAura)
		judgeSpell := paladin.registerUtilityJudgement(rank, judgementDebuffs)
		trigger := seal.newProc(paladin, rank)

		aura := paladin.RegisterAura(core.Aura{
			Label:    seal.name + paladin.Label + strconv.Itoa(i+1),
			ActionID: core.ActionID{SpellID: rank.castID},
			Duration: sealDuration,
			OnSpellHitDealt: func(_ *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
				if !result.Landed() || !spell.ProcMask.Matches(core.ProcMaskMelee) {
					return
				}
				if ppmm.Proc(sim, spell.ProcMask, seal.name) {
					trigger(sim)
				}
			},
		})

		paladin.registerSealCastSpell(seal.mask, aura, rank.level, i+1, rank.manaCost, judgeSpell)
		paladin.spellsJoUtility = append(paladin.spellsJoUtility, judgeSpell)
		paladin.aurasUtilitySeals = append(paladin.aurasUtilitySeals, aura)
	}
}

// sealDuration is the client's duration_ms (30000) on every seal's cast.
const sealDuration = 30 * time.Second

// registerUtilityJudgement is the spell Judgement resolves to while the
// seal is up. It rolls on the spell hit table and lays the judgement on a
// hit; it deals no damage.
func (paladin *Paladin) registerUtilityJudgement(rank utilitySealRank, debuffs core.AuraArray) *core.Spell {
	return paladin.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: rank.judgementID},
		SpellSchool: core.SpellSchoolHoly,
		DefenseType: core.DefenseTypeMagic,
		ProcMask:    core.ProcMaskEmpty,
		Flags:       core.SpellFlagMeleeMetrics | core.SpellFlagNoOnCastComplete,

		// Each judgement's spell level is its seal rank's (Judgement of
		// Light 30 / 40 / 50 / 60, Wisdom 38 / 48 / 58).
		RequiredLevel: int(rank.level),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			if spell.CalcAndDealOutcome(sim, target, spell.OutcomeMagicHit).Landed() {
				debuffs.Get(target).Activate(sim)
			}
		},
	})
}

// registerSealCastSpell registers the cast of a seal whose effect is its
// aura: the client's cost and 1.5 s GCD, the aura as the spell's own buff
// (30 s), and applySeal on cast.
func (paladin *Paladin) registerSealCastSpell(mask uint64, aura *core.Aura, level int32, rank int, manaCost float64, judge *core.Spell) *core.Spell {
	return paladin.RegisterSpell(core.SpellConfig{
		ClassSpellMask: mask,
		ActionID:       aura.ActionID,
		SpellSchool:    core.SpellSchoolHoly,
		Flags:          core.SpellFlagAPL,

		RequiredLevel: int(level),
		Rank:          rank,

		RelatedSelfBuff: aura,

		ManaCost: core.ManaCostOptions{
			FlatCost:   manaCost - paladin.getLibramSealCostReduction(),
			Multiplier: paladin.benediction(),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
			paladin.applySeal(aura, judge, spell, sim)
		},
	})
}

// newSealOfLightProc registers the heal spell and returns the trigger.
func newSealOfLightProc(paladin *Paladin, rank utilitySealRank) func(sim *core.Simulation) {
	heal := clientdamage.Effect{Amount: rank.amount}
	healSpell := paladin.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: rank.procID},
		SpellSchool: core.SpellSchoolHoly,
		DefenseType: core.DefenseTypeMagic,
		ProcMask:    core.ProcMaskSpellHealing,
		Flags:       core.SpellFlagHelpful | core.SpellFlagNoOnCastComplete,

		RequiredLevel: int(rank.level),

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		ClientBaseDamage: heal.Range(int(paladin.Level)),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.CalcAndDealHealing(sim, target, rank.amount, spell.OutcomeHealing)
		},
	})
	return func(sim *core.Simulation) { healSpell.Cast(sim, &paladin.Unit) }
}

// newSealOfWisdomProc returns the trigger that restores the mana. The
// metrics are filed under the client's energize spell (not the cast, whose
// id already carries the cast's mana spend).
func newSealOfWisdomProc(paladin *Paladin, rank utilitySealRank) func(sim *core.Simulation) {
	metrics := paladin.NewManaMetrics(core.ActionID{SpellID: rank.procID})
	return func(sim *core.Simulation) { paladin.AddMana(sim, rank.amount, metrics) }
}
