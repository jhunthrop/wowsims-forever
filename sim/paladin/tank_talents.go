package paladin

import (
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

// The Protection tree, as the live client states it (talents/paladin.json,
// build 1.60.1.70707). Every constant quotes the rank text it came from;
// where the text leaves a number out, the constant says "unconfirmed" and
// what was assumed.

// ForeverProtectionTalents is the fixed tank input the regression suite
// and the level smoke run, so a change in the golden is attributable to
// the engine rather than to a build edit. It is the build the site's
// Protection paladin guide gives, written against the 1.60.1.70707 tree:
// Holy, Protection and Retribution 17 characters each (TalentTreeSizes).
// Spend:
//
//	Holy 10:        Divine Strength 5, Improved Seals 5
//	Protection 33:  Divine Intellect 1, Redoubt 5, Precision 3, Sacred
//	                Duty 2, Templar's Bulwark 1, Shield Specialization 2,
//	                Toughness 5, Swift Judgement 1, Improved Righteous
//	                Fury 3, Anticipation 5, Holy Shield 1, Iron Creed 3,
//	                Shieldsworn 1
//	Retribution 8:  Deflection 5, Conviction 3
//
// Shieldsworn is the 31-point capstone and needs Holy Shield. Deflection
// (+5% parry, the only parry a paladin can take), Sacred Duty and Iron
// Creed are the survival picks past the core; the 70707 tree cut Iron
// Creed and Shield Specialization's ranks and removed Improved Seal of
// Fury and One-Handed Weapon Specialization, and those points moved to
// Shieldsworn, Conviction (crit on every attack and spell), Improved Seals
// and Divine Intellect (TestForeverProtectionTalentsAreLegal checks the
// tiers and prerequisites against the client's tree).
const ForeverProtectionTalents = "05005000000000000-15302102513501031-50030000000000000"

const (
	// Anticipation (node 105636): "Increases your Defense Skill by 4" at
	// rank 1, 20 at rank 5. Defense feeds dodge, parry, block and crit
	// reduction through the character's defense dependencies.
	anticipationDefensePerRank = 4.0

	// Toughness (node 105630): "Increases your armor value from items by
	// 2%" a rank, 10% at rank 5.
	toughnessArmorPctPerRank = 0.02

	// Precision (node 105638): "Improves your chance to hit by 1%" a rank.
	precisionHitPctPerRank = 1.0

	// Sacred Duty (node 105632): "Reduces the cooldown of your Divine
	// Shield, Divine Protection, and Templar's Bulwark spells by 30 sec,
	// and reduces the duration of Forbearance by 5 sec", 60 and 10 sec at
	// rank 2 (client 1.60.1.70707; it gave 2% Stamina a rank before).
	sacredDutyCooldownReductionPerRank    = 30 * time.Second
	sacredDutyForbearanceReductionPerRank = 5 * time.Second

	// Shield Specialization (node 110874, 2 ranks): "Increases the amount
	// of damage absorbed by your shield by 15%, and gives your blocks a 50%
	// chance to restore 4% of your maximum Mana. May only occur once every
	// 3 sec." 30% and 100% at rank 2 (client 1.60.1.70707; 10% and 33% a
	// rank over 3 ranks before).
	shieldSpecializationAbsorbPerRank = 0.15
	shieldSpecializationManaFraction  = 0.04 // "restore 4% of your maximum Mana" (1.60.1.70425; 6% before)
	shieldSpecializationManaICD       = 3 * time.Second
	shieldSpecializationActionID      = 20150

	// Redoubt (node 105626): "Damaging melee attacks against you have a
	// 10% chance to increase your chance to block by 6%. Lasts 10 sec or
	// 5 blocks.", 30% at rank 5 (client 1.60.1.70707; 4% a rank before).
	// It triggers on any damaging melee hit, not only a crit.
	redoubtProcChance         = 0.10
	redoubtBlockChancePerRank = 6.0
	redoubtDuration           = 10 * time.Second
	redoubtBlocks             = 5
	redoubtActionID           = 20134

	// Reckoning (node 105627): "Gives you a 8% chance to gain an extra
	// attack after Blocking a melee attack and a 20% chance to gain an
	// extra attack after being the victim of a non-periodic critical
	// strike", 40% and 100% at rank 5.
	reckoningBlockChancePerRank = 0.08
	reckoningCritChancePerRank  = 0.20
	reckoningActionID           = 20178
	reckoningInternalCooldown   = 1500 * time.Millisecond

	// Iron Creed (node 110879, 3 ranks): "Increases the threat generated
	// by your Holy Strike ability by 10%. While Righteous Fury is active,
	// Holy Strike also reduces your damage taken by 2% for 6 sec.", 30%
	// and 6% at rank 3 (client 1.60.1.70707; 5% and 2% a rank over 5 ranks
	// before). 1311033 is the id of the trigger the client names.
	ironCreedThreatPerRank      = 0.10
	ironCreedDamageTakenPerRank = 0.02
	ironCreedDuration           = 6 * time.Second
	ironCreedActionID           = 1311033
)

var (
	// shieldSpecializationManaChance is the block-to-mana chance by rank.
	shieldSpecializationManaChance = [3]float64{0, 0.50, 1.00}
)

// applyProtectionTalents applies the Protection tree's passive effects.
// Righteous Fury's own talent (Improved Righteous Fury) is read by
// righteous_fury.go, and Holy Shield, Templar's Bulwark and Swift
// Judgement by their abilities.
func (paladin *Paladin) applyProtectionTalents() {
	t := paladin.Talents

	paladin.AddStat(stats.Hit, precisionHitPctPerRank*float64(t.Precision)*core.HitRatingPerHitChance)
	if t.Toughness > 0 {
		paladin.ApplyEquipScaling(stats.Armor, 1.0+toughnessArmorPctPerRank*float64(t.Toughness))
	}
	paladin.AddStat(stats.Defense, anticipationDefensePerRank*float64(t.Anticipation))
	paladin.forbearanceDurationMod -= sacredDutyForbearanceReductionPerRank * time.Duration(t.SacredDuty)

	// Shield Specialization's absorb is the shield's block value, which
	// BlockValue() scales for item block value only (Strength's share is
	// not "damage absorbed by your shield").
	paladin.PseudoStats.BlockValueMultiplier += shieldSpecializationAbsorbPerRank * float64(t.ShieldSpecialization)

	paladin.applyShieldSpecializationMana()
	paladin.applyRedoubt()
	paladin.applyReckoning()
	paladin.applyIronCreed()
}

func (paladin *Paladin) applyShieldSpecializationMana() {
	rank := paladin.Talents.ShieldSpecialization
	if rank == 0 {
		return
	}

	manaMetrics := paladin.NewManaMetrics(core.ActionID{SpellID: shieldSpecializationActionID})
	chance := shieldSpecializationManaChance[rank]
	restore := func(sim *core.Simulation) {
		paladin.AddMana(sim, shieldSpecializationManaFraction*paladin.MaxMana(), manaMetrics)
	}
	core.MakeProcTriggerAura(&paladin.Unit, core.ProcTrigger{
		Name:       "Shield Specialization",
		ActionID:   core.ActionID{SpellID: shieldSpecializationActionID},
		Callback:   core.CallbackOnSpellHitTaken,
		Outcome:    core.OutcomeBlock,
		ProcChance: chance,
		ICD:        shieldSpecializationManaICD,
		Handler: func(sim *core.Simulation, _ *core.Spell, _ *core.SpellResult) {
			restore(sim)
		},
	})
	// Shieldsworn's shield strike rolls the same chance outside the
	// internal cooldown.
	paladin.shieldSpecializationOnStrike = func(sim *core.Simulation) {
		if sim.Proc(chance, "Shield Specialization (Shieldsworn)") {
			restore(sim)
		}
	}
}

// applyRedoubt: any damaging melee hit has a chance to give the block
// bonus, which a refresh brings back to five blocks. One aura owns the
// proc and the spending, so a hit never spends a block of the bonus it
// has just granted.
func (paladin *Paladin) applyRedoubt() {
	rank := paladin.Talents.Redoubt
	if rank == 0 {
		return
	}

	blockBonus := redoubtBlockChancePerRank * float64(rank) * core.BlockRatingPerBlockChance

	paladin.redoubtAura = paladin.RegisterAura(core.Aura{
		Label:     "Redoubt",
		ActionID:  core.ActionID{SpellID: redoubtActionID},
		Duration:  redoubtDuration,
		MaxStacks: redoubtBlocks,
		OnGain: func(_ *core.Aura, sim *core.Simulation) {
			paladin.AddStatDynamic(sim, stats.Block, blockBonus)
		},
		OnExpire: func(_ *core.Aura, sim *core.Simulation) {
			paladin.AddStatDynamic(sim, stats.Block, -blockBonus)
		},
	})

	paladin.RegisterAura(core.Aura{
		Label:    "Redoubt Trigger",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitTaken: func(_ *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if result.DidBlock() && paladin.redoubtAura.IsActive() {
				paladin.redoubtAura.RemoveStack(sim)
			}
			if spell.ProcMask.Matches(core.ProcMaskMelee) && result.Landed() && sim.Proc(redoubtProcChance, "Redoubt") {
				paladin.redoubtAura.Activate(sim)
				paladin.redoubtAura.SetStacks(sim, redoubtBlocks)
			}
		},
	})
}

// applyReckoning grants the extra attack on two triggers: blocking a
// melee attack, and any non-periodic critical strike taken (a periodic
// tick reaches OnPeriodicDamageTaken, not OnSpellHitTaken).
func (paladin *Paladin) applyReckoning() {
	rank := float64(paladin.Talents.Reckoning)
	if rank == 0 {
		return
	}

	procID := core.ActionID{SpellID: reckoningActionID}
	// "Cannot occur more often than once every 1.5 sec", for both triggers
	// together (the text added in build 1.60.1.70291).
	icd := core.Cooldown{Timer: paladin.NewTimer(), Duration: reckoningInternalCooldown}
	grantExtraAttack := func(sim *core.Simulation, spell *core.Spell, _ *core.SpellResult) {
		if !icd.IsReady(sim) {
			return
		}
		icd.Use(sim)
		paladin.AutoAttacks.ExtraMHAttack(sim, 1, procID, spell.ActionID)
	}

	core.MakeProcTriggerAura(&paladin.Unit, core.ProcTrigger{
		Name:       "Reckoning Block Trigger",
		Callback:   core.CallbackOnSpellHitTaken,
		Outcome:    core.OutcomeBlock,
		ProcMask:   core.ProcMaskMelee,
		ProcChance: reckoningBlockChancePerRank * rank,
		Handler:    grantExtraAttack,
	})
	core.MakeProcTriggerAura(&paladin.Unit, core.ProcTrigger{
		Name:       "Reckoning Crit Trigger",
		Callback:   core.CallbackOnSpellHitTaken,
		Outcome:    core.OutcomeCrit,
		ProcChance: reckoningCritChancePerRank * rank,
		Handler:    grantExtraAttack,
	})
}

// applyIronCreed: Holy Strike's threat, and the damage-taken window it
// opens while Righteous Fury is active (the aura is only built for a
// paladin that has Righteous Fury on).
func (paladin *Paladin) applyIronCreed() {
	rank := float64(paladin.Talents.IronCreed)
	if rank == 0 {
		return
	}

	paladin.AddStaticMod(core.SpellModConfig{
		Kind:       core.SpellMod_Threat_Pct,
		ClassMask:  PaladinSpellMaskHolyStrike,
		FloatValue: 1 + ironCreedThreatPerRank*rank,
	})

	if !paladin.Options.RighteousFury {
		return
	}
	paladin.ironCreedAura = paladin.newDamageTakenReductionAura(
		"Iron Creed", core.ActionID{SpellID: ironCreedActionID}, ironCreedDuration, ironCreedDamageTakenPerRank*rank)
}

// newDamageTakenReductionAura is a timed "all damage taken reduced by
// reduction" buff (Iron Creed, Divine Protection).
func (paladin *Paladin) newDamageTakenReductionAura(label string, actionID core.ActionID, duration time.Duration, reduction float64) *core.Aura {
	multiplier := 1 - reduction
	return paladin.RegisterAura(core.Aura{
		Label:    label,
		ActionID: actionID,
		Duration: duration,
		OnGain: func(aura *core.Aura, _ *core.Simulation) {
			aura.Unit.PseudoStats.DamageTakenMultiplier *= multiplier
		},
		OnExpire: func(aura *core.Aura, _ *core.Simulation) {
			aura.Unit.PseudoStats.DamageTakenMultiplier /= multiplier
		},
	})
}

// activateIronCreed opens Iron Creed's damage-taken window; it is a no-op
// without the talent or without Righteous Fury.
func (paladin *Paladin) activateIronCreed(sim *core.Simulation) {
	if paladin.ironCreedAura != nil {
		paladin.ironCreedAura.Activate(sim)
	}
}
