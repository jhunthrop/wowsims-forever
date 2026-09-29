package hunter

import (
	"strconv"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

func (hunter *Hunter) getExplosiveTrapConfig(rank int, timer *core.Timer) core.SpellConfig {
	// Ids, mana cost and level match spellconst/hunter.json's own
	// spells table exactly (13813/14316/14317; the old
	// 409532/409534/409535 ids do not exist in the client at all). The
	// cast spell's own effect 104 (trigger spell) points at a
	// server-side script spell (164839/164879/164880) with no entry
	// anywhere in spellconst, but the actual damage lives in a
	// separate, real "Explosive Trap Effect" spell per rank
	// (13812/14314/14315, spell_level matching): effect index 0 is a
	// flat, non-random instant hit (effect 2, sp/ap coefficient both
	// 0) of 115/163/229, and effect index 1 is the 10-tick, 2s-period
	// AoE dot of 15/24/33 per tick - both corroborated on Wowhead's
	// Forever pages (spell=13812/14315: "School Damage ... Value: 116"
	// / "230", "16 every 2 seconds" / "34 every 2 seconds"). The old
	// code rolled a min/max range (104-135/145-193/208-265) that
	// approximated but never matched this flat value; instantDamage
	// below replaces it.
	spellId := [4]int32{0, 13813, 14316, 14317}[rank]
	dotDamage := [4]float64{0, 15, 24, 33}[rank]
	instantDamage := [4]float64{0, 115, 163, 229}[rank]
	manaCost := [4]float64{0, 275, 395, 520}[rank]
	level := [4]int{0, 34, 44, 54}[rank]

	return core.SpellConfig{
		SpellCode:     SpellCode_HunterExplosiveTrap,
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

		Dot: core.DotConfig{
			IsAOE: true,
			Aura: core.Aura{
				Label: "ExplosiveTrap" + hunter.Label + strconv.Itoa(rank),
				Tag:   "ExplosiveTrap",
			},
			NumberOfTicks: 10,
			TickLength:    time.Second * 2,

			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
				dot.Snapshot(target, dotDamage, isRollover)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				for _, aoeTarget := range sim.Encounter.TargetUnits {
					// Explosive Trap DoT only does damage if the target does not have an immolation trap ticking on them
					if !aoeTarget.HasActiveAuraWithTag("ImmolationTrap") {
						dot.CalcAndDealPeriodicSnapshotDamage(sim, aoeTarget, dot.OutcomeTick)
					}
				}
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			if hunter.DistanceFromTarget > 5 {
				return
			}

			spell.WaitTravelTime(sim, func(s *core.Simulation) {
				curTarget := target
				// Read at detonation, beside the AoE cap multiplier
				// below: a target timeline changes the count mid-fight,
				// and the old registration-time read both missed later
				// adds and wrapped surplus hits onto the survivors.
				numHits := len(sim.Encounter.TargetUnits)
				// Traps gain no benefit from hit bonuses except for the Trap Mastery talent, since this is a unique interaction this is my workaround
				spellHit := spell.Unit.GetStat(stats.Hit) + target.PseudoStats.BonusSpellHitRatingTaken
				spell.Unit.AddStatDynamic(sim, stats.Hit, spellHit*-1)
				for hitIndex := 0; hitIndex < numHits; hitIndex++ {
					baseDamage := instantDamage * sim.Encounter.AOECapMultiplier()
					spell.CalcAndDealDamage(sim, curTarget, baseDamage, spell.OutcomeMagicHitAndCrit)
					curTarget = sim.Environment.NextTargetUnit(curTarget)
				}
				spell.Unit.AddStatDynamic(sim, stats.Hit, spellHit)
				spell.AOEDot().ApplyOrReset(sim)
			})
		},
	}
}

func (hunter *Hunter) registerExplosiveTrapSpell(timer *core.Timer) {
	maxRank := 3
	for i := 1; i <= maxRank; i++ {
		config := hunter.getExplosiveTrapConfig(i, timer)

		if config.RequiredLevel <= int(hunter.Level) {
			hunter.ExplosiveTrap = hunter.GetOrRegisterSpell(config)
		}
	}
}
