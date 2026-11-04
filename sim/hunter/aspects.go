package hunter

import (
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
	"strconv"
	"time"
)

// AspectOfTheHawkAuraTag is the Aura.Tag every rank of Aspect of the Hawk
// shares, so a caller outside this file (talents.go's Deadly Aspects)
// can test "is Aspect of the Hawk active" without knowing which rank's
// spell id is current.
const AspectOfTheHawkAuraTag = "AspectOfTheHawk"

// Utility function to create an Improved Hawk Aura
func (hunter *Hunter) createImprovedHawkAura(auraLabel string, actionID core.ActionID) *core.Aura {
	bonusMultiplier := 1.3
	return hunter.GetOrRegisterAura(core.Aura{
		Label:    auraLabel,
		ActionID: actionID,
		Duration: time.Second * 12,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.MultiplyRangedSpeed(sim, bonusMultiplier)
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.MultiplyRangedSpeed(sim, 1/bonusMultiplier)
		},
	})
}

// quickStrikesSpellID is Quick Strikes (469144), Aspect of the Falcon's
// proc: aura 319 at 30 (melee attack speed), duration index 29 (12 sec).
const quickStrikesSpellID int32 = 469144

func (hunter *Hunter) createQuickStrikesAura() *core.Aura {
	bonusMultiplier := 1.3
	return hunter.GetOrRegisterAura(core.Aura{
		Label:    "Quick Strikes",
		ActionID: core.ActionID{SpellID: quickStrikesSpellID},
		Duration: time.Second * 12,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.MultiplyMeleeSpeed(sim, bonusMultiplier)
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.MultiplyMeleeSpeed(sim, 1/bonusMultiplier)
		},
	})
}

// getMaxAspectOfTheHawkAttackPower is the rank's ranged attack power: the
// client's aura-124 base points on spells 13165/14318/14319/14320/14321/
// 14322/25296, read from the generated AspectOfTheHawkBaseDamage. The
// 1.60.1.70753 hotfix push (2026-11-03) set ranks 1-6 to 20/35/50/70/90/110
// (from 70707's 15/22/30/38/47/55); rank 7 (25296, AQ) stays 120.
func (hunter *Hunter) getMaxAspectOfTheHawkAttackPower(rank int) float64 {
	if rank < 1 || rank > AspectOfTheHawkRanks {
		return 0.0
	}
	return AspectOfTheHawkBaseDamage[rank][0]
}

// getMaxHawkRank is the rank a hunter casts: the strongest rank learned by
// their level, not the highest. Through client 1.60.1.70685 rank 6 (55)
// was weaker than rank 5 (90); since 70707 rank 5 is 47, so the strongest
// is again the highest learned. The comparison stays so a future inferior
// rank is skipped from the data alone.
func (hunter *Hunter) getMaxHawkRank() int {
	maxRank := core.TernaryInt(core.IncludeAQ, 7, 6)

	best := 1
	for rank := 1; rank <= maxRank; rank++ {
		config := hunter.getAspectOfTheHawkSpellConfig(rank)
		if config.RequiredLevel > int(hunter.Level) {
			continue
		}
		if hunter.getMaxAspectOfTheHawkAttackPower(rank) > hunter.getMaxAspectOfTheHawkAttackPower(best) {
			best = rank
		}
	}
	return best
}

func (hunter *Hunter) getAspectOfTheHawkSpellConfig(rank int) core.SpellConfig {
	var impHawkAura *core.Aura
	// FOREVER: Improved Aspect of the Hawk is not in the client's trees, so
	// the proc never fires and impHawkAura stays nil.
	// improvedHawkProcChance := 0.01 * float64(hunter.Talents.ImprovedAspectOfTheHawk)
	improvedHawkProcChance := 0.0

	spellIds := [8]int32{0, 13165, 14318, 14319, 14320, 14321, 14322, 25296}
	levels := [8]int{0, 10, 18, 28, 38, 48, 58, 60}

	spellId := spellIds[rank]
	level := levels[rank]

	/*
		if hunter.Talents.ImprovedAspectOfTheHawk > 0 {
			impHawkAura = hunter.createImprovedHawkAura(
				"Quick Shots",
				core.ActionID{SpellID: 6150},
			)
		}
	*/
	// Use utility function to get the attack power based on rank
	rap := hunter.getMaxAspectOfTheHawkAttackPower(rank)

	actionID := core.ActionID{SpellID: spellId}
	aspectOfTheHawkAura := hunter.GetOrRegisterAura(core.Aura{
		Label: "Aspect of the Hawk" + strconv.Itoa(rank),
		// Tag, not the "Aspect" ExclusiveEffect category below (a
		// different namespace - NewExclusiveEffect only groups mutually
		// exclusive aspects, it does not set Aura.Tag): Deadly Aspects
		// (talents.go) reads this tag with HasActiveAuraWithTag to gate
		// its Auto Shot proc on "while Aspect of the Hawk is active."
		Tag:      AspectOfTheHawkAuraTag,
		ActionID: actionID,
		Duration: core.NeverExpires,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.AddStatDynamic(sim, stats.RangedAttackPower, rap*hunter.AspectOfTheHawkAPMultiplier)
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.AddStatDynamic(sim, stats.RangedAttackPower, -rap*hunter.AspectOfTheHawkAPMultiplier)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if !spell.ProcMask.Matches(core.ProcMaskRangedAuto) {
				return
			}

			if impHawkAura != nil && sim.Proc(improvedHawkProcChance, "Imp Aspect of the Hawk") {
				impHawkAura.Activate(sim)
			}
		},
	})

	aspectOfTheHawkAura.NewExclusiveEffect("Aspect", true, core.ExclusiveEffect{})

	return core.SpellConfig{
		ActionID:      actionID,
		Flags:         core.SpellFlagAPL,
		Rank:          rank,
		RequiredLevel: level,

		ManaCost: core.ManaCostOptions{
			FlatCost: AspectOfTheHawkManaCost[rank],
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return !aspectOfTheHawkAura.IsActive()
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			aspectOfTheHawkAura.Activate(sim)
		},
	}
}

func (hunter *Hunter) registerAspectOfTheHawkSpell() {
	hunter.AspectOfTheHawkAPMultiplier = 1.0
	maxRank := hunter.getMaxHawkRank()
	config := hunter.getAspectOfTheHawkSpellConfig(maxRank)
	hunter.GetOrRegisterSpell(config)
}
