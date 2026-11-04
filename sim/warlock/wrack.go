package warlock

import (
	"slices"
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

// Wrack is Affliction's level-60 capstone (talents/warlock.json node
// 105909, prereq Siphon Life): a 6 second, no-cooldown channel that also
// makes the target take more damage from the warlock's Corruption and Bane
// of Agony while it lasts. Talent text: "Tears the
// target apart from within, dealing 36 Shadow damage every 1 sec and
// increasing the damage they take from your other Shadow damage over
// time effects by 10%. Lasts 6 sec."
//
// Client numbers (spellconst/warlock.json spells["1316697"], corroborated
// by wowhead's Forever page):
//   - cast_time_ms 0, gcd_ms 1500, cooldown_ms 0, cost 200 (mana),
//     spell_level 40, duration_ms 6000.
//   - effect 0: aura 3 (PERIODIC_DAMAGE), amount 36 at the spell's level
//     40 plus EffectRealPointsPerLevel 2.5 a caster level up to
//     SpellLevels.MaxLevel 60 (86 a tick at 60), EffectBonusCoefficient
//     0.286, period_ms 1000. The 70755 hotfix push doubled the
//     coefficient (0.143 -> 0.286) and added the per-level growth (0 ->
//     2.5); the tick is read from the generated Wrack* tables.
//   - effect 1: aura 271, amount 10, period_ms 0 -- a flat, non-periodic
//     10, matching the "+10%" in the talent text.
//
// Wrack is a channel: SpellMisc Attributes_1 0x4004, the bits Drain Life
// and Mind Flay carry, and Wowhead's Forever tooltip reads "Channeled (6 sec
// cast)". The caster casts nothing else for its six ticks, and its
// vulnerability is the channel's own aura, so it ends with the channel.
//
// The vulnerability is aura 271 (damage taken from the caster) on class mask
// 0x402: Corruption (0x2) and Bane of Agony (0x400). The text's "your other
// Shadow damage over time effects" resolves to those two; Siphon Life, Bane
// of Doom, Shadow Bolt and other casters' Shadow damage take nothing from it.
// WrackTickDamage is effect 0's per-tick amount at the caster's level.
var WrackTickDamage = clientdamage.FromTable(WrackBaseDamage[:], WrackPointsPerLevel[:], WrackLevel[:], WrackMaxLevel[:])[1]

// WrackTickCoefficient is effect 0's EffectBonusCoefficient.
var WrackTickCoefficient = WrackSpellCoeff[1]

const WrackNumTicks = 6
const WrackTickLength = time.Second * 1
const WrackDuration = WrackTickLength * WrackNumTicks
const WrackVulnerabilityBonus = 0.10
const WrackSpellID = 1316697

// wrackVulnerableSpellCodes are the spells in the vulnerability's class mask.
var wrackVulnerableSpellCodes = []int32{SpellCode_WarlockCorruption, SpellCode_WarlockCurseOfAgony}

func (warlock *Warlock) newWrackVulnerabilityAura(target *core.Unit) *core.Aura {
	return target.GetOrRegisterAura(core.Aura{
		Label:    "Wrack Vulnerability-" + warlock.Label,
		ActionID: core.ActionID{SpellID: WrackSpellID, Tag: 1},
		Duration: WrackDuration,
	})
}

// applyWrackVulnerability makes the warlock's own Corruption and Bane of
// Agony take the vulnerability on every target, read when each tick lands.
func (warlock *Warlock) applyWrackVulnerability() {
	warlock.Env.RegisterPostFinalizeEffect(func() {
		for _, target := range warlock.Env.Encounter.AllTargetUnits {
			for _, at := range warlock.AttackTables[target.UnitIndex] {
				previous := at.DamageDoneByCasterMultiplier
				at.DamageDoneByCasterMultiplier = func(spell *core.Spell, at *core.AttackTable) float64 {
					multiplier := 1.0
					if previous != nil {
						multiplier = previous(spell, at)
					}
					if slices.Contains(wrackVulnerableSpellCodes, spell.SpellCode) && warlock.WrackVulnerabilityAuras.Get(at.Defender).IsActive() {
						multiplier *= 1 + WrackVulnerabilityBonus
					}
					return multiplier
				}
			}
		}
	})
}

func (warlock *Warlock) registerWrackSpell() {
	if !warlock.Talents.Wrack {
		return
	}

	warlock.WrackVulnerabilityAuras = warlock.NewEnemyAuraArray(warlock.newWrackVulnerabilityAura)
	warlock.applyWrackVulnerability()

	casterLevel := int(warlock.Level)
	tickDamage := WrackTickDamage.Center(casterLevel)

	config := core.SpellConfig{
		ActionID:      core.ActionID{SpellID: WrackSpellID},
		SpellSchool:   core.SpellSchoolShadow,
		SpellCode:     SpellCode_WarlockWrack,
		ProcMask:      core.ProcMaskSpellDamage,
		DefenseType:   core.DefenseTypeMagic,
		Flags:         core.SpellFlagAPL | core.SpellFlagResetAttackSwing | core.SpellFlagChanneled | core.SpellFlagPureDot | WarlockFlagAffliction,
		RequiredLevel: 40,
		// One rank, growing 2.5 a tick each level to MaxLevel 60: never below
		// the caster, so the low-rank penalty (core.SpellLevelPenalty) leaves it whole.

		ManaCost: core.ManaCostOptions{
			FlatCost: WrackManaCost[1],
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
		},

		CritDamageBonus: 0,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		ClientBaseDamage: WrackTickDamage.Range(casterLevel),
		BonusCoefficient: WrackTickCoefficient, // the report compares the spell's, which a pure DoT never reads

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "Wrack-" + warlock.Label,
				OnExpire: func(aura *core.Aura, sim *core.Simulation) {
					warlock.WrackVulnerabilityAuras.Get(aura.Unit).Deactivate(sim)
				},
			},

			NumberOfTicks:    WrackNumTicks,
			TickLength:       WrackTickLength,
			BonusCoefficient: WrackTickCoefficient,

			Dynamic: true,

			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
				dot.Snapshot(target, tickDamage, isRollover)
				if !isRollover {
					// Soul Siphon (talents.go): see drain_life.go's
					// identical hook for the talent's wording. Wrack is
					// itself an Affliction effect, so soulSiphonMultiplier
					// excludes it from its own count of "other" effects.
					dot.SnapshotAttackerMultiplier *= warlock.soulSiphonMultiplier(target, dot.Spell)
				}
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				// The per-tick read (Dynamic) re-took the plain snapshot;
				// Soul Siphon's count is read at the tick too.
				dot.SnapshotAttackerMultiplier *= warlock.soulSiphonMultiplier(target, dot.Spell)
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
				return spell.CalcPeriodicDamage(sim, target, tickDamage, spell.OutcomeExpectedMagicAlwaysHit)
			}
		},
	}

	warlock.Wrack = warlock.GetOrRegisterSpell(config)
}
