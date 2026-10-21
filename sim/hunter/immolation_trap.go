package hunter

import (
	"strconv"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

func (hunter *Hunter) getImmolationTrapConfig(rank int, timer *core.Timer) core.SpellConfig {
	// The client's duration_ms is 60000 for every rank -- the trap's
	// armed lifetime on the ground, the same quantity
	// explosive_trap.go's and freezing_trap.go's comments describe --
	// but this spell's Dot IS exposed to compare.go's engineDuration
	// (it is a real per-target Dot, not AOE), so the conformance report
	// shows a genuine number here instead of a missing one: 15000ms,
	// this engine's 5-tick, 3s-period burn length (NumberOfTicks *
	// TickLength below). Both 60000 and 15000 are real, correctly
	// implemented numbers; they are just not the same quantity (armed
	// lifetime vs. burn length), so this is left as a documented
	// semantic mismatch rather than forced to match.
	//
	// Ids, mana cost and level match spellconst/hunter.json's own
	// spells table exactly (13795/14302/14303/14304/14305; the old
	// 409521-409530 ids do not exist in the client at all). The cast
	// spell's own effect 104 (trigger spell) points at a server-side
	// script spell (164638/164872/164873/164874/164875) with no entry
	// anywhere in spellconst, but the actual dot damage lives in a
	// separate, real "Immolation Trap Effect" spell per rank
	// (13797/14298/14299/14300/14301, spell_level matching): a 5-tick,
	// 3s-period periodic-damage aura (effect 6, aura 3) of 21/43/68/
	// 102/138 per tick - ImmolationTrapTickDamage
	// (client_damage.go) and TickLength (fixed from 1.5s to the real 3s below) both now match
	// exactly. Corroborated on Wowhead's Forever pages (spell=13797/
	// 14301: "22 every 3 seconds" / "139 every 3 seconds", 15s/5-tick
	// duration).
	spellId := [6]int32{0, 13795, 14302, 14303, 14304, 14305}[rank]
	tickDamage := ImmolationTrapTickDamage[rank]
	casterLevel := int(hunter.Level)
	manaCost := [6]float64{0, 50, 90, 135, 190, 245}[rank]
	level := [6]int{0, 16, 26, 36, 46, 56}[rank]

	return core.SpellConfig{
		SpellCode:     SpellCode_HunterImmolationTrap,
		ActionID:      core.ActionID{SpellID: spellId},
		SpellSchool:   core.SpellSchoolFire,
		DefenseType:   core.DefenseTypeMagic,
		ProcMask:      core.ProcMaskSpellDamage,
		Flags:         core.SpellFlagAPL | SpellFlagTrap,
		Rank:          rank,
		RequiredLevel: level,
		MissileSpeed:  24,

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},
		Cast: core.CastConfig{
			CD: core.Cooldown{
				Timer: timer,
				// spellconst's category_cooldown_ms is 30000 for every
				// rank (confirmed on Wowhead's Forever pages: "Cooldown:
				// 30 seconds"), not the old 15s - this is the shared
				// "Traps" category cooldown, not a per-rank value.
				Duration: time.Second * 30,
			},
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			IgnoreHaste: true, // Hunter GCD is locked at 1.5s
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		ClientBaseDamage: tickDamage.Range(casterLevel),

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "ImmolationTrap" + hunter.Label + strconv.Itoa(rank),
				Tag:   "ImmolationTrap",
			},
			NumberOfTicks: 5,
			TickLength:    time.Second * 3,

			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
				dot.Snapshot(target, tickDamage.Roll(sim, casterLevel), isRollover)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			if hunter.DistanceFromTarget > 5 {
				return
			}
			// Traps gain no benefit from hit bonuses except for the Trap Mastery talent, since this is a unique interaction this is my workaround
			spellHit := spell.Unit.GetStat(stats.Hit) + target.PseudoStats.BonusSpellHitRatingTaken
			spell.Unit.AddStatDynamic(sim, stats.Hit, spellHit*-1)
			result := spell.CalcOutcome(sim, target, spell.OutcomeMagicHitNoHitCounter)
			spell.Unit.AddStatDynamic(sim, stats.Hit, spellHit)
			spell.WaitTravelTime(sim, func(s *core.Simulation) {
				spell.DealOutcome(sim, result)
				if result.Landed() {
					spell.Dot(target).Apply(sim)
				}
			})
		},
	}
}

func (hunter *Hunter) registerImmolationTrapSpell(timer *core.Timer) {
	maxRank := 5
	for i := 1; i <= maxRank; i++ {
		config := hunter.getImmolationTrapConfig(i, timer)

		if config.RequiredLevel <= int(hunter.Level) {
			hunter.ImmolationTrap = hunter.GetOrRegisterSpell(config)
		}
	}
}
