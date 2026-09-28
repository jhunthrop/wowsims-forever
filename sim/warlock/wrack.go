package warlock

import (
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

// Wrack is Affliction's level-60 capstone (talents/warlock.json node
// 105909, prereq Siphon Life): an instant, no-cooldown DoT that also
// makes the target take more damage from the warlock's other Shadow
// damage-over-time effects for its duration. Talent text: "Tears the
// target apart from within, dealing 36 Shadow damage every 1 sec and
// increasing the damage they take from your other Shadow damage over
// time effects by 10%. Lasts 6 sec."
//
// Client numbers (spellconst/warlock.json spells["1316697"], corroborated
// by wowhead's Forever page):
//   - cast_time_ms 0, gcd_ms 1500, cooldown_ms 0, cost 200 (mana),
//     spell_level 40, duration_ms 6000.
//   - effect 0: aura 3 (PERIODIC_DAMAGE), amount 36, sp_coefficient
//     0.143, period_ms 1000 -- "36 Shadow damage every 1 sec" exactly.
//   - effect 1: aura 271, amount 10, period_ms 0 -- a flat, non-periodic
//     10, matching the "+10%" in the talent text.
//
// Engine limitation: the fork's existing damage-taken-multiplier
// convention (core's Improved Shadow Bolt / Shadow Weaving debuffs) is a
// school-wide SchoolDamageTakenMultiplier on the target, with no
// narrower "only this caster's periodic Shadow effects, not this one"
// scope. Wrack's vulnerability aura is implemented the same way: it
// increases ALL Shadow damage the target takes (direct and periodic,
// from any source) by 10% while it is up, which is broader than the
// talent text's "your other Shadow DoTs" wording. This mirrors the
// fork's only precedent for this kind of effect; see report-warlock.md.
const WrackTickBaseDamage = 36.0
const WrackTickCoefficient = 0.14300000668
const WrackNumTicks = 6
const WrackTickLength = time.Second * 1
const WrackDuration = WrackTickLength * WrackNumTicks
const WrackManaCost = 200.0
const WrackVulnerabilityBonus = 0.10
const WrackSpellID = 1316697

func (warlock *Warlock) newWrackVulnerabilityAura(target *core.Unit) *core.Aura {
	return target.GetOrRegisterAura(core.Aura{
		Label:    "Wrack Vulnerability-" + warlock.Label,
		ActionID: core.ActionID{SpellID: WrackSpellID, Tag: 1},
		Duration: WrackDuration,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.PseudoStats.SchoolDamageTakenMultiplier[stats.SchoolIndexShadow] *= 1 + WrackVulnerabilityBonus
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.PseudoStats.SchoolDamageTakenMultiplier[stats.SchoolIndexShadow] /= 1 + WrackVulnerabilityBonus
		},
	})
}

func (warlock *Warlock) registerWrackSpell() {
	if !warlock.Talents.Wrack {
		return
	}

	warlock.WrackVulnerabilityAuras = warlock.NewEnemyAuraArray(warlock.newWrackVulnerabilityAura)

	config := core.SpellConfig{
		ActionID:      core.ActionID{SpellID: WrackSpellID},
		SpellSchool:   core.SpellSchoolShadow,
		SpellCode:     SpellCode_WarlockWrack,
		ProcMask:      core.ProcMaskSpellDamage,
		DefenseType:   core.DefenseTypeMagic,
		Flags:         core.SpellFlagAPL | core.SpellFlagResetAttackSwing | core.SpellFlagPureDot | WarlockFlagAffliction,
		RequiredLevel: 40,

		ManaCost: core.ManaCostOptions{
			FlatCost: WrackManaCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
		},

		CritDamageBonus: 0,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "Wrack-" + warlock.Label,
			},

			NumberOfTicks:    WrackNumTicks,
			TickLength:       WrackTickLength,
			BonusCoefficient: WrackTickCoefficient,

			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
				dot.Snapshot(target, WrackTickBaseDamage, isRollover)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcOutcome(sim, target, spell.OutcomeMagicHitNoHitCounter)
			if result.Landed() {
				dot := spell.Dot(target)
				dot.Apply(sim)
				warlock.WrackVulnerabilityAuras.Get(target).Activate(sim)
			}
			spell.DealOutcome(sim, result)
		},
		ExpectedTickDamage: func(sim *core.Simulation, target *core.Unit, spell *core.Spell, useSnapshot bool) *core.SpellResult {
			if useSnapshot {
				dot := spell.Dot(target)
				return dot.CalcSnapshotDamage(sim, target, dot.Spell.OutcomeExpectedMagicAlwaysHit)
			} else {
				return spell.CalcPeriodicDamage(sim, target, WrackTickBaseDamage, spell.OutcomeExpectedMagicAlwaysHit)
			}
		},
	}

	warlock.Wrack = warlock.GetOrRegisterSpell(config)
}
