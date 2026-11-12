package paladin

import (
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// The attack table, defense type and proc mask every damage judgement
// shares since the 11 November 2026 hotfix made Judgement a ranged attack
// (see registerJudgement).
const (
	judgementCastType    = proto.CastType_CastTypeRanged
	judgementDefenseType = core.DefenseTypeRanged
	judgementProcMask    = core.ProcMaskRangedSpecial
)

func (paladin *Paladin) registerJudgement() {
	// Judgement functions as a dummy spell in vanilla.
	// It rolls on the spell hit table and can only miss or hit.
	// Individual seals have their own effects that this spell triggers,
	// that are handled in the implementations of the seal auras.
	//
	// Forever: "Judgement no longer consumes Seals" (Blizzard's Deep Dive
	// recap), so the seal stays up and Judgement only triggers its effect.
	//
	// Blizzard's 11 November 2026 hotfix (t/2403040): "Judgement changed to
	// now be considered a Ranged Attack. This means it no longer counts as
	// a Melee attack for the purposes of item and effect procs and uses the
	// Paladin's Ranged Hit chance." Server side: client 1.60.1.70888 still
	// files the judgements on the melee table. So this ability and every
	// judgement it triggers (judgementCastType, judgementDefenseType and
	// judgementProcMask below) roll the paladin's ranged attack table, the
	// base skill of a class with no ranged weapon, never a weapon racial,
	// and carry a ranged proc mask that no melee proc reads: Windfury
	// Totem, weapon enchants and Hand of Justice no longer fire on one.
	paladin.judgement = paladin.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: 20271},
		ClassSpellMask: PaladinSpellMaskJudgement,
		SpellSchool:    core.SpellSchoolHoly,
		ProcMask:       core.ProcMaskEmpty,
		CastType:       judgementCastType,
		// core.SpellFlagPassiveSpell used to sit here too. That flag
		// tells core/metrics_aggregator.go's addSpellMetrics to drop
		// the spell's OWN Casts count ("applied as a result of another
		// spell"), which is correct for the seal-specific judgement
		// spells this ApplyEffects triggers (castSpecificJudgement,
		// below) but wrong for THIS wrapper: it is the ability the
		// rotation names directly (SpellFlagAPL, right beside it), so
		// its Casts count is a real player cast, not a side effect.
		// The identical, identically-caused bug is sim/warrior/execute.go's.
		Flags: core.SpellFlagMeleeMetrics | core.SpellFlagAPL | core.SpellFlagNoOnCastComplete | core.SpellFlagCastTimeNoGCD,

		// RequiredLevel 4: source 1.60.1.70009 client spell data
		// (spellconst/paladin.json, spell 20271 "Judgement"), spell_level 4.
		//
		// Cost: 6% of base mana, from the same client row's cost_pct
		// (SpellPower.PowerCostPct). The flat `cost` column is 0 for
		// every percent-priced spell, and a 2026-10-06 conformance fix
		// read that zero as "free" and removed the 0.06 BaseCost this
		// spell had always carried; spellconst now carries cost_pct and
		// the paladin golden flagged the free Judgement the next day.
		RequiredLevel: 4,
		ManaCost: core.ManaCostOptions{
			BaseCost:   JudgementManaCostPct[0] / 100,
			Multiplier: paladin.benediction(),
		},

		Cast: core.CastConfig{
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    paladin.NewTimer(),
				Duration: time.Second * (10 - time.Duration(paladin.Talents.ImprovedJudgement)),
			},
		},
		ExtraCastCondition: func(_ *core.Simulation, _ *core.Unit) bool {
			return paladin.currentSeal.IsActive()
		},
		ApplyEffects: func(sim *core.Simulation, target *core.Unit, _ *core.Spell) {
			paladin.currentJudgement.Cast(sim, target)
			paladin.trySanctifiedJudgementManaReturn(sim)
		},
	})
}
