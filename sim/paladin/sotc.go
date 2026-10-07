package paladin

import (
	"strconv"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

func (paladin *Paladin) registerSealOfTheCrusader() {
	type judge struct {
		spellID int32
		bonus   float64
	}

	var ranks = []struct {
		level      int32
		spellID    int32
		manaCost   float64
		scaleLevel int32
		ap         float64
		scale      float64
		judge      judge
	}{
		{level: 6, spellID: 21082, manaCost: 25, scaleLevel: 12, ap: 31, scale: 0.7, judge: judge{spellID: 21183, bonus: 20}},
		{level: 12, spellID: 20162, manaCost: 40, scaleLevel: 20, ap: 51, scale: 1.1, judge: judge{spellID: 20188, bonus: 30}},
		{level: 22, spellID: 20305, manaCost: 65, scaleLevel: 30, ap: 94, scale: 1.7, judge: judge{spellID: 20300, bonus: 50}},
		{level: 32, spellID: 20306, manaCost: 90, scaleLevel: 40, ap: 145, scale: 2, judge: judge{spellID: 20301, bonus: 80}},
		{level: 42, spellID: 20307, manaCost: 125, scaleLevel: 50, ap: 221, scale: 2.2, judge: judge{spellID: 20302, bonus: 110}},
		{level: 52, spellID: 20308, manaCost: 160, scaleLevel: 60, ap: 306, scale: 2.4, judge: judge{spellID: 20303, bonus: 140}},
	}

	// FOREVER: Improved Seal of the Crusader is not in the client's trees.
	// improvedSotC := []float64{1, 1.05, 1.1, 1.15}[paladin.Talents.ImprovedSealOfTheCrusader]
	improvedSotC := 1.0

	var libramAp, libramBonus float64
	if paladin.Ranged().ID == LibramOfFervor {
		libramAp, libramBonus = libramOfFervorBonuses()
	}

	for i, rank := range ranks {
		rank := rank
		if paladin.Level < rank.level {
			break
		}

		debuffs := paladin.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
			return core.JudgementOfTheCrusaderAura(&paladin.Unit, target, improvedSotC, libramBonus)
		})

		judgeSpell := paladin.RegisterSpell(core.SpellConfig{
			ActionID:    core.ActionID{SpellID: rank.judge.spellID},
			SpellSchool: core.SpellSchoolHoly,
			DefenseType: core.DefenseTypeMagic,
			ProcMask:    core.ProcMaskEmpty,
			Flags:       core.SpellFlagMeleeMetrics,

			// source 1.60.1.70009 client spell data (spellconst/paladin.json):
			// Judgement of the Crusader's spell_level equals the
			// owning Seal of the Crusader rank's own level at every
			// rank. Flagged by paladin.golden.md's "required_level
			// N->0" rows - this SpellConfig never set the field at
			// all.
			//
			// Its duration_ms (40000, every rank) is NOT wired here:
			// the debuff it applies is core.JudgementOfTheCrusaderAura
			// (sim/core/debuffs.go), which hardcodes Duration to 10s -
			// a stale-vanilla-literal bug in sim/core, out of scope
			// for this lane (never edit sim/core). Wiring RelatedSelfBuff
			// to that aura would trade today's "missing aura duration"
			// mismatch for a numeric one (40000->10000) this lane
			// cannot fix. Reported as a core follow-up.
			RequiredLevel: int(rank.level),

			ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				spell.CalcAndDealOutcome(sim, target, spell.OutcomeAlwaysHit)
				debuffs.Get(target).Activate(sim)
			},
		})

		ap := rank.ap + rank.scale*float64(min(paladin.Level, rank.scaleLevel)-rank.level)

		aura := paladin.RegisterAura(core.Aura{
			Label:    "Seal of the Crusader" + paladin.Label + strconv.Itoa(i+1),
			ActionID: core.ActionID{SpellID: rank.spellID},
			Duration: time.Second * 30,
			OnGain: func(_ *core.Aura, sim *core.Simulation) {
				paladin.MultiplyMeleeSpeed(sim, 1.4)
				paladin.AutoAttacks.MHAuto().DamageMultiplier /= 1.4
				paladin.AddStatDynamic(sim, stats.AttackPower, ap*improvedSotC+libramAp)
			},
			OnExpire: func(_ *core.Aura, sim *core.Simulation) {
				paladin.MultiplyMeleeSpeed(sim, 1/1.4)
				paladin.AutoAttacks.MHAuto().DamageMultiplier *= 1.4
				paladin.AddStatDynamic(sim, stats.AttackPower, -(ap*improvedSotC + libramAp))
			},
		})

		paladin.aurasSotC = append(paladin.aurasSotC, aura)

		paladin.RegisterSpell(core.SpellConfig{
			ClassSpellMask: PaladinSpellMaskSealOfTheCrusaderCast,
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

		paladin.spellsJotC = append(paladin.spellsJotC, judgeSpell)
	}
}
