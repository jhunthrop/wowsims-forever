package hunter

import (
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// Summon Hawk (talents/hunter.json node 104966, Beast Mastery, bool,
// spell 1293241): "Command a hawk to dive-bomb your targeted enemy,
// dealing ${32+($rap*(5/100))} Physical damage and continuing its
// assault for 18 sec. Only 2 hawks can be active at once. Summon Hawk
// shares its cooldown with Arcane Shot."
//
// The instant half is fully specified by the tooltip's own dynamic
// formula (32 flat + 5% of Ranged Attack Power, both corroborated by
// spellconst/hunter.json's own effect 0: amount 32, cost 80 mana,
// category_cooldown_ms 6000 - the same category Arcane Shot's own
// spells carry) and is registered below.
//
// The "continuing its assault for 18 sec" half is NOT modeled. The
// cast spell's own effect 1 (effect code 32, a dummy/trigger effect)
// points at trigger_spell 1312639, which spellconst/hunter.json's
// mined data has no body for at all - no duration, no attack speed, no
// per-swing damage, nothing - because it is the hawk NPC's own kit, not
// a player-spell record the data pipeline captures. Two hawks stacking
// independently is also a second mechanic (a capped-count temporary
// pet) this package has no machinery for, and neither gap can be
// closed with a SpellMod, a Dot, or an invented tick cadence without
// guessing numbers the client data does not give - a core-level
// temporary-pet feature is what would actually model it faithfully.
// SummonHawk's instant hit is real and talented-gated like the rest of
// this file; the sustained portion is the one piece of this talent
// that needs a decision above this lane (see PORTING.md's "no changes
// to sim/core" boundary).
// summonHawkRangedAPCoeff is the tooltip's 5% of Ranged Attack Power, the
// one share the client's table does not carry; the flat, cost and level
// follow the rank (SummonHawkDamage, SummonHawkManaCost, SummonHawkLevel).
const summonHawkRangedAPCoeff = 0.05

func (hunter *Hunter) getSummonHawkConfig(rank int, timer *core.Timer) core.SpellConfig {
	flatDamage := SummonHawkDamage[rank]
	casterLevel := int(hunter.Level)
	return core.SpellConfig{
		SpellCode:     SpellCode_HunterSummonHawk,
		ActionID:      core.ActionID{SpellID: SummonHawkSpellId[rank]},
		SpellSchool:   core.SpellSchoolPhysical,
		DefenseType:   core.DefenseTypeRanged,
		ProcMask:      core.ProcMaskRangedSpecial,
		Flags:         core.SpellFlagMeleeMetrics | core.SpellFlagAPL | SpellFlagShot,
		CastType:      proto.CastType_CastTypeRanged,
		Rank:          rank,
		RequiredLevel: SummonHawkLevel[rank],
		MissileSpeed:  24,

		ManaCost: core.ManaCostOptions{
			FlatCost: SummonHawkManaCost[rank],
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			IgnoreHaste: true, // Hunter GCD is locked at 1.5s
			CD: core.Cooldown{
				Timer: timer,
				// Base category_cooldown_ms is 6000, the same as Arcane
				// Shot's own base. Not reduced by Improved Arcane Shot:
				// that talent's own rank text names only "your Arcane
				// Shot spell," not the shared category, and the two
				// abilities sharing a timer does not make Summon Hawk's
				// own Cooldown.Duration field the same variable.
				Duration: time.Second * 6,
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return hunter.DistanceFromTarget >= core.MinRangedAttackDistance
		},

		CritDamageBonus: hunter.mortalShots(),

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		ClientBaseDamage: flatDamage.Range(casterLevel),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := flatDamage.Roll(sim, casterLevel) + summonHawkRangedAPCoeff*spell.RangedAttackPower(target, false)
			result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeRangedHitAndCrit)

			spell.WaitTravelTime(sim, func(s *core.Simulation) {
				spell.DealDamage(sim, result)
			})
		},
	}
}

func (hunter *Hunter) registerSummonHawkSpell(arcaneShotTimer *core.Timer) {
	if !hunter.Talents.SummonHawk {
		return
	}
	rank := core.HighestRankAtLevel(SummonHawkLevel[1:], hunter.Level)
	if rank == 0 {
		return
	}

	hunter.SummonHawk = hunter.GetOrRegisterSpell(hunter.getSummonHawkConfig(rank, arcaneShotTimer))
}
