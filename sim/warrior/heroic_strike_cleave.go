package warrior

import (
	"github.com/wowsims/classic/sim/core"
)

// heroicStrikeRank and cleaveRank are the ranks of the two on-next-swing
// abilities a level-60 warrior casts, as labels into the generated rank
// arrays in constants_auto_gen.go. Heroic Strike's rank 9 arrived with
// AQ, so the phase flag picks the rank and every column - id, damage,
// cost - is then read at that one index rather than being a second
// ternary of typed numbers.
func heroicStrikeRank() int {
	return core.TernaryInt(core.IncludeAQ, HeroicStrikeRanks, HeroicStrikeRanks-1)
}

func cleaveRank() int {
	return CleaveRanks
}

// heroicStrikeSpellID and cleaveSpellID are the ids the two
// on-next-swing abilities register under. They are named so a rotation
// test can check the pinned APL against the spellbook rather than
// against a retyped id.
func heroicStrikeSpellID() int32 {
	return HeroicStrikeSpellId[heroicStrikeRank()]
}

func cleaveSpellID() int32 {
	return CleaveSpellId[cleaveRank()]
}

func (warrior *Warrior) registerHeroicStrikeSpell(realismICD *core.Cooldown) {
	rank := heroicStrikeRank()
	flatDamageBonus := HeroicStrikeBaseDamage[rank][0]
	spellID := heroicStrikeSpellID()
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
			Cost:   rageCost(HeroicStrikeManaCost[rank]),
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

	results := make([]*core.SpellResult, min(int32(2), warrior.Env.GetNumTargets()))

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
			for idx := range results {
				baseDamage := flatDamageBonus + spell.Unit.MHWeaponDamage(sim, spell.MeleeAttackPower(target))
				results[idx] = spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)
				target = sim.Environment.NextTargetUnit(target)
			}

			for _, result := range results {
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
		Flags:    core.SpellFlagMeleeMetrics | core.SpellFlagAPL | core.SpellFlagCastTimeNoGCD,

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
