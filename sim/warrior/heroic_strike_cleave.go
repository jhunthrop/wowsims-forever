package warrior

import (
	"github.com/wowsims/classic/sim/core"
)

// heroicStrikeRank and cleaveRank are the TOP ranks of the two
// on-next-swing abilities a level-60 warrior casts, as labels into the
// generated rank arrays in constants_auto_gen.go. Heroic Strike's rank
// 9 arrived with AQ, so the phase flag picks the rank and every column
// - id, damage, cost - is then read at that one index rather than
// being a second ternary of typed numbers. Used by
// TestTheForeverFuryRotationQueuesHeroicStrike (a level-60 pinned
// rotation) and as the ceiling heroicStrikeRankForLevel clamps to.
func heroicStrikeRank() int {
	return core.TernaryInt(core.IncludeAQ, HeroicStrikeRanks, HeroicStrikeRanks-1)
}

func cleaveRank() int {
	return CleaveRanks
}

// heroicStrikeRankForLevel is the rank a warrior of level has actually
// learned: rankAtLevel against HeroicStrikeLevel, capped at
// heroicStrikeRank() so the AQ phase flag still wins at 60. Without
// this, registerHeroicStrikeSpell always registered the level-60 rank
// (heroicStrikeRank() ignores the character's level entirely), so a
// levelling character's queued Heroic Strike never matched the rank
// id the ladder's rotation casts at its own level.
func heroicStrikeRankForLevel(level int32) int {
	return min(rankAtLevel(HeroicStrikeLevel[:], level), heroicStrikeRank())
}

// heroicStrikeSpellID and cleaveSpellID are the ids the two
// on-next-swing abilities register under at their TOP rank. They are
// named so a rotation test can check the pinned (level-60) APL against
// the spellbook rather than against a retyped id.
func heroicStrikeSpellID() int32 {
	return HeroicStrikeSpellId[heroicStrikeRank()]
}

func cleaveSpellID() int32 {
	return CleaveSpellId[cleaveRank()]
}

// heroicStrikeRankSpellID reads a Heroic Strike rank's id, correcting
// the one rank where the generator's dedup kept the wrong duplicate:
// rank 3's client row (285, spell_level 16, cost 150) and 25712 (same
// spell_level, cost 0 - a "free" duplicate the client also carries)
// tie on spell_level, and the generator's tie-break kept 25712 in
// HeroicStrikeSpellId[3]. 285 is the id spellranks.json names and the
// one the site's ladder rewrites a levelling rotation's castSpell to,
// so it is the one this package registers.
func heroicStrikeRankSpellID(rank int) int32 {
	if rank == 3 {
		return 285
	}
	return HeroicStrikeSpellId[rank]
}

// heroicStrikeRankManaCost mirrors heroicStrikeRankSpellID for cost:
// the generated HeroicStrikeManaCost[3] is 0, read off the same free
// duplicate (25712) rather than 285's real 150.
func heroicStrikeRankManaCost(rank int) float64 {
	if rank == 3 {
		return 150
	}
	return HeroicStrikeManaCost[rank]
}

func (warrior *Warrior) registerHeroicStrikeSpell(realismICD *core.Cooldown) {
	rank := heroicStrikeRankForLevel(warrior.Level)
	flatDamageBonus := HeroicStrikeBaseDamage[rank][0]
	spellID := heroicStrikeRankSpellID(rank)
	// No known equation, and the client's table has no threat column.
	threat := core.TernaryFloat64(core.IncludeAQ, 173, 145)

	warrior.HeroicStrike = warrior.RegisterSpell(AnyStance, core.SpellConfig{
		ClassSpellMask: WarriorSpellMaskHeroicStrike,
		ActionID:       core.ActionID{SpellID: spellID},
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskMeleeMHSpecial | core.ProcMaskMeleeMHAuto,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagNoOnCastComplete | SpellFlagOffensive,

		RequiredLevel: HeroicStrikeLevel[rank],
		Rank:          rank,

		RageCost: core.RageCostOptions{
			// Improved Heroic Strike's discount is a SpellMod in
			// talents.go; applying it here as well would double it, so
			// this is the client's undiscounted cost.
			Cost:   rageCost(heroicStrikeRankManaCost(rank)),
			Refund: 0.8,
		},

		CritDamageBonus: warrior.impale(),

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		FlatThreatBonus:  threat,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := flatDamageBonus + spell.Unit.MHWeaponDamage(sim, spell.MeleeAttackPower(target))
			result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)

			if !result.Landed() {
				spell.IssueRefund(sim)
			}

			spell.DealDamage(sim, result)
			if warrior.curQueueAura != nil {
				warrior.curQueueAura.Deactivate(sim)
			}
		},
	})
	warrior.HeroicStrikeQueue = warrior.makeQueueSpellsAndAura(warrior.HeroicStrike, realismICD)
}

func (warrior *Warrior) registerCleaveSpell(realismICD *core.Cooldown) {
	rank := cleaveRank()
	flatDamageBonus := CleaveBaseDamage[rank][0]
	spellID := cleaveSpellID()
	// No known equation, and the client's table has no threat column.
	threat := 100.0

	// FOREVER: the client's Improved Cleave is a rage discount, not a
	// damage bonus (see applyDeclarativeTalents); the vanilla
	// multiplier that stood here is gone rather than renamed.

	// Pool-sized ceiling, live-bounded loop; see registerWhirlwindSpell.
	results := make([]*core.SpellResult, min(2, len(warrior.Env.Encounter.AllTargetUnits)))

	warrior.Cleave = warrior.RegisterSpell(AnyStance, core.SpellConfig{
		ClassSpellMask: WarriorSpellMaskCleave,
		ActionID:       core.ActionID{SpellID: spellID},
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskMeleeMHSpecial | core.ProcMaskMeleeMHAuto,
		Flags:          core.SpellFlagMeleeMetrics | SpellFlagOffensive,

		RequiredLevel: CleaveLevel[rank],
		Rank:          rank,

		RageCost: core.RageCostOptions{
			// Improved Cleave's discount is a SpellMod in talents.go.
			Cost: rageCost(CleaveManaCost[rank]),
		},

		CritDamageBonus: warrior.impale(),

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		FlatThreatBonus:  threat,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			numHits := min(len(results), len(sim.Encounter.TargetUnits))
			for idx := 0; idx < numHits; idx++ {
				baseDamage := flatDamageBonus + spell.Unit.MHWeaponDamage(sim, spell.MeleeAttackPower(target))
				results[idx] = spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)
				target = sim.Environment.NextTargetUnit(target)
			}

			for _, result := range results[:numHits] {
				spell.DealDamage(sim, result)
			}

			if warrior.curQueueAura != nil {
				warrior.curQueueAura.Deactivate(sim)
			}
		},
	})
	warrior.CleaveQueue = warrior.makeQueueSpellsAndAura(warrior.Cleave, realismICD)
}

func (warrior *Warrior) makeQueueSpellsAndAura(srcSpell *WarriorSpell, realismICD *core.Cooldown) *WarriorSpell {
	isQueueQueued := false

	queueAura := warrior.RegisterAura(core.Aura{
		Label:    "HS/Cleave Queue Aura-" + srcSpell.ActionID.String(),
		ActionID: srcSpell.ActionID.WithTag(1),
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			isQueueQueued = false
		},
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			if warrior.curQueueAura != nil {
				warrior.curQueueAura.Deactivate(sim)
			}
			warrior.PseudoStats.DisableDWMissPenalty = true
			warrior.curQueueAura = aura
			warrior.curQueuedAutoSpell = srcSpell
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			warrior.PseudoStats.DisableDWMissPenalty = false
			warrior.curQueueAura = nil
			warrior.curQueuedAutoSpell = nil
		},
	})

	queueSpell := warrior.RegisterSpell(AnyStance, core.SpellConfig{
		ActionID: srcSpell.ActionID.WithTag(1),
		// Same Rank as srcSpell (not left at the zero value): this spell
		// shares srcSpell's SpellID (just a different ActionID.Tag, which
		// sim/conformance's rowFor does not look at), so without this the
		// two registrations produced two report rows for the same
		// (spec, level, SpellID) - srcSpell's real one and this queue
		// helper's all-zero one (no Cost/RequiredLevel of its own) - and
		// the queue helper's showed up as a bogus "no cost block"
		// registration gap. Matching Rank collapses them onto one row via
		// collectRows' own (spec, level, spellID, rank) dedup; it cannot
		// be SpellFlagPassiveSpell instead (see sim/warrior/execute.go's
		// identical note) because this spell IS cast directly by the APL
		// and its own Casts count is real.
		Rank:  srcSpell.Rank,
		Flags: core.SpellFlagMeleeMetrics | core.SpellFlagAPL | core.SpellFlagCastTimeNoGCD,

		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			// GetCurrentCost, not DefaultCast.Cost: the rage discounts
			// from Improved Heroic Strike and Improved Cleave are
			// SpellMods now, and a mod writes Cost.FlatModifier rather
			// than the default cast. Gating on the undiscounted number
			// would queue the ability less often than the talent says.
			return warrior.curQueueAura == nil &&
				!isQueueQueued &&
				warrior.CurrentRage() >= srcSpell.Cost.GetCurrentCost() &&
				!warrior.IsCasting(sim) &&
				realismICD.IsReady(sim)
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			if realismICD.IsReady(sim) {
				isQueueQueued = true
				realismICD.Use(sim)
				sim.AddPendingAction(&core.PendingAction{
					NextActionAt: sim.CurrentTime + realismICD.Duration,
					OnAction: func(sim *core.Simulation) {
						queueAura.Activate(sim)
						isQueueQueued = false
					},
				})
			}
		},
	})

	return queueSpell
}

func (warrior *Warrior) TryHSOrCleave(sim *core.Simulation, mhSwingSpell *core.Spell) *core.Spell {
	if !warrior.curQueueAura.IsActive() {
		return mhSwingSpell
	}

	if !warrior.curQueuedAutoSpell.CanCast(sim, warrior.CurrentTarget) {
		warrior.curQueueAura.Deactivate(sim)
		return mhSwingSpell
	}

	return warrior.curQueuedAutoSpell.Spell
}
