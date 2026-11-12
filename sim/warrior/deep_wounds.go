package warrior

import (
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

func (warrior *Warrior) applyDeepWounds() {
	if warrior.Talents.DeepWounds == 0 {
		return
	}

	spellID := deepWoundsSpellIDs[rankIndex(warrior.Talents.DeepWounds, deepWoundsSpellIDs[:])]

	warrior.DeepWounds = warrior.RegisterSpell(AnyStance, core.SpellConfig{
		SpellCode:   SpellCode_WarriorDeepWounds,
		ActionID:    core.ActionID{SpellID: spellID},
		SpellSchool: core.SpellSchoolPhysical,
		ProcMask:    core.ProcMaskEmpty,
		Flags:       core.SpellFlagNoOnCastComplete | core.SpellFlagPassiveSpell,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: 1,

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "Deep Wounds",
			},
			NumberOfTicks: deepWoundsTicks,
			TickLength:    time.Second * 3,
			// Forever's Deep Wound (the server's 412613, what the talent's
			// 12162 trigger deals each tick) is flagged Can't Crit in the
			// client (SpellMisc Attributes_2 0x20000000 in 1.60.1.70790
			// and 70888), the one bleed that is: its ticks never crit,
			// where Forever's other bleeds roll at the melee crit chance.
			// The launch logs read the same: 0 critical ticks in 146 Deep
			// Wound ticks over 17 warriors whose Rend ticks crit 13 times
			// in 193 (foreverlogs.gg 2815 to 2853).
			NeverCrits: true,

			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				attackTable := warrior.AttackTables[target.UnitIndex][proto.CastType_CastTypeMainHand]
				dot.SnapshotAttackerMultiplier = dot.Spell.AttackerDamageMultiplier(attackTable, true) // Double dips on attackers mods
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			// A refresh keeps the running tick timer (forever-bugs #234);
			// procDeepWounds has already rolled the damage over.
			dot := spell.Dot(target)
			if dot.IsActive() {
				// Four ticks again on the running timer, the bleed ending
				// on its last one.
				dot.ApplyOrRefresh(sim)
				dot.Aura.UpdateExpires(sim, dot.NextTickAt()+dot.TickPeriod()*time.Duration(deepWoundsTicks-1))
			} else {
				dot.Apply(sim)
			}
			spell.CalcAndDealOutcome(sim, target, spell.OutcomeAlwaysHitNoHitCounter)
		},
	})

	core.MakePermanent(warrior.RegisterAura(core.Aura{
		Label: "Deep Wounds Talent",
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if spell.ProcMask.Matches(core.ProcMaskEmpty) || !spell.SpellSchool.Matches(core.SpellSchoolPhysical) {
				return
			}

			// Ravager doesn't proc Deep Wounds
			if spell.ActionID.SpellID == 9633 {
				return
			}

			if result.Outcome.Matches(core.OutcomeCrit) {
				warrior.procDeepWounds(sim, result.Target, spell.IsOH())
			}
		},
	}))
}

// deepWoundsTicks is the bleed's four 3 s ticks ("over 12 sec").
const deepWoundsTicks = 4

// procDeepWounds starts or rolls the bleed after a crit. Blizzard on
// forever-bugs #234 (2026-10-05), describing Forever's Deep Wounds against
// vanilla's: it "Rolls over its damage when refreshed", "Doesn't reset its
// tick timer when it is refreshed" and "Doesn't scale with Attack Power"
// ("it scales with your weapon damage and not with your Attack Power at
// all"). The engine used to read the average weapon damage with attack
// power in it, replace the running bleed and restart its ticks.
func (warrior *Warrior) procDeepWounds(sim *core.Simulation, target *core.Unit, isOh bool) {
	dot := warrior.DeepWounds.Dot(target)

	weapon, castType, auto := warrior.AutoAttacks.MH(), proto.CastType_CastTypeMainHand, warrior.AutoAttacks.MHAuto()
	if isOh {
		weapon, castType, auto = warrior.AutoAttacks.OH(), proto.CastType_CastTypeOffHand, warrior.AutoAttacks.OHAuto()
	}
	adm := auto.AttackerDamageMultiplier(warrior.AttackTables[target.UnitIndex][castType], true)
	newDamage := deepWoundsTotal(weapon, isOh, warrior.Talents.DeepWounds) * adm

	var tickDamage float64
	var ticksRemaining int32
	if dot.IsActive() {
		tickDamage, ticksRemaining = dot.SnapshotBaseDamage, dot.MaxTicksRemaining()
	}
	dot.SnapshotBaseDamage = deepWoundsRolledTick(tickDamage, ticksRemaining, newDamage)
	dot.SnapshotAttackerMultiplier = 1

	warrior.DeepWounds.Cast(sim, target)
}

// deepWoundsTotal is the bleed one crit adds before the attacker
// multiplier: 20% a point of the hand's average weapon damage, the off
// hand at half, and no attack power (forever-bugs #234).
func deepWoundsTotal(weapon *core.Weapon, offHand bool, points int32) float64 {
	total := weapon.AverageDamage() * 0.2 * float64(points)
	if offHand {
		total *= 0.5
	}
	return total
}

// deepWoundsRolledTick is the tick after a crit: what the running bleed
// has left plus the new bleed, over four ticks again.
func deepWoundsRolledTick(tickDamage float64, ticksRemaining int32, newTotal float64) float64 {
	return (tickDamage*float64(ticksRemaining) + newTotal) / deepWoundsTicks
}
