package warlock

import (
	"slices"
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

// Wrack is Affliction's level-60 capstone (talents/warlock.json node
// 105909, prereq Siphon Life): a 6 second, no-cooldown channel. Since the
// 10 Nov hotfix its text reads: "Tears the target apart from within for 6
// sec, inflicting Shadow damage every 1 sec. While channeling, your Siphon
// Life, Corruption, and Bane of Agony effects on the target deal their
// periodic damage 100% faster."
//
// Numbers (spellconst/warlock.json spells["1316697"] for the rest):
//   - cast_time_ms 0, gcd_ms 1500, cooldown_ms 0, cost 200 (mana),
//     spell_level 40, duration_ms 6000.
//   - effect 0: aura 3 (PERIODIC_DAMAGE), period 1000 ms. The 10 Nov
//     hotfix ("Wrack damage reduced by 30%") is push 113766 on 1.60.1.70888
//     (SpellEffect 1349556): 25.2 a tick at the spell's level 40 plus 1.75
//     a caster level up to SpellLevels.MaxLevel 60 (60.2 a tick at 60),
//     EffectBonusCoefficient 0.2002. Each is 70% of the 70755 push's row
//     (36, 2.5, 0.286). The active build's generated tables (70790) still
//     carry the old row, so the tick is stated here until the client move.
//   - effect 1: the same push turns it from aura 271 (+10% damage taken
//     from the caster on Corruption and Bane of Agony) into a dummy at 100:
//     the speed-up. "Wrack no longer increases the damage of the Warlock's
//     periodic effects on the target."
//
// Wrack is a channel: SpellMisc Attributes_1 0x4004, the bits Drain Life
// and Mind Flay carry, and Wowhead's Forever tooltip reads "Channeled (6 sec
// cast)". The caster casts nothing else for its six ticks.
//
// The speed-up runs the clock of the warlock's own Siphon Life, Corruption
// and Bane of Agony on the channelled target at twice the sim's while the
// channel lasts. The hotfix's note: "the time it skips is kept after
// Wrack's channel ends or is interrupted. For example, if you channel Wrack
// for 1 second during a 3-second Corruption tick, Corruption advances 2
// seconds instead of 1, so its next tick lands 1 second later instead of
// 2." Each DoT therefore ends sooner and deals the same ticks.
var WrackTickDamage = clientdamage.Effect{Amount: 25.2, PerLevel: 1.75, SpellLevel: 40, MaxLevel: 60}

// WrackTickCoefficient is effect 0's EffectBonusCoefficient (push 113766).
const WrackTickCoefficient = 0.2002

const WrackNumTicks = 6
const WrackTickLength = time.Second * 1
const WrackDuration = WrackTickLength * WrackNumTicks
const WrackSpellID = 1316697

// WrackDoTClockRate is effect 1's "100% faster".
const WrackDoTClockRate = 2.0

// wrackHastenedDoTs are the warlock's DoT spells the channel speeds up.
func (warlock *Warlock) wrackHastenedDoTs() []*core.Spell {
	return slices.Concat(warlock.SiphonLife, warlock.Corruption, warlock.CurseOfAgony)
}

// setWrackClockRate runs the hastened DoTs' clocks on target at rate.
func (warlock *Warlock) setWrackClockRate(sim *core.Simulation, target *core.Unit, rate float64) {
	for _, spell := range warlock.wrackHastenedDoTs() {
		spell.Dot(target).SetClockRate(sim, rate)
	}
}

func (warlock *Warlock) registerWrackSpell() {
	if !warlock.Talents.Wrack {
		return
	}

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
				OnGain: func(aura *core.Aura, sim *core.Simulation) {
					warlock.setWrackClockRate(sim, aura.Unit, WrackDoTClockRate)
				},
				OnExpire: func(aura *core.Aura, sim *core.Simulation) {
					warlock.setWrackClockRate(sim, aura.Unit, 1)
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
					// identical hook for the talent's wording.
					dot.SnapshotAttackerMultiplier *= warlock.soulSiphonMultiplier(target)
				}
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				// The per-tick read (Dynamic) re-took the plain snapshot;
				// Soul Siphon's count is read at the tick too.
				dot.SnapshotAttackerMultiplier *= warlock.soulSiphonMultiplier(target)
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcOutcome(sim, target, spell.OutcomeMagicHitNoHitCounter)
			if result.Landed() {
				dot := spell.Dot(target)
				dot.Apply(sim)
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
