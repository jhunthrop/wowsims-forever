package paladin

import (
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

// lightsVigilRank is one rank of Light's Vigil: the cast the player learns
// (castID) and the heal it later gives a party (healID).
type lightsVigilRank struct {
	castID   int32
	healID   int32
	level    int
	manaCost float64
	heal     clientdamage.Effect
}

// lightsVigilRanks are trainables "Light's Vigil" (levels 40, 50, 60; the
// Holy capstone talent grants the spell).
var lightsVigilRanks = []lightsVigilRank{
	{castID: 1310911, healID: 1310912, level: 40, manaCost: 730,
		heal: clientdamage.Effect{Amount: 324, Variance: 0.056213, PerLevel: 1.2, SpellLevel: 40, MaxLevel: 49}},
	{castID: 1311590, healID: 1311591, level: 50, manaCost: 1000,
		heal: clientdamage.Effect{Amount: 487, Variance: 0.056213, PerLevel: 1.5, SpellLevel: 50, MaxLevel: 59}},
	{castID: 1311595, healID: 1311596, level: 60, manaCost: 1340,
		heal: clientdamage.Effect{Amount: 704, Variance: 0.056213, PerLevel: 1.8, SpellLevel: 60, MaxLevel: 69}},
}

const (
	lightsVigilCastTime    = 1500 * time.Millisecond
	lightsVigilCooldown    = 6 * time.Second
	lightsVigilDuration    = 30 * time.Second
	lightsVigilCoefficient = 0.143
)

// lightsVigilState is the one Light's Vigil a paladin may keep up: the
// client allows one per party, and a healer's own casts are what a sim
// can see, so a new cast moves the old one.
type lightsVigilState struct {
	aura   *core.Aura
	target *core.Unit
	heal   *core.Spell
}

// registerLightsVigil registers Light's Vigil (talent node 1310911): "Applies
// Light's Vigil to the target for 30 sec. Your next Holy Shock cast on them
// triggers no cooldown and causes enemy targets to suffer 175 to 189 Holy
// damage and refund 75% of Light's Vigil's Mana cost, or allied targets to
// heal their party for 326 to 344."
//
// Only the allied half is modelled, because it is the healer's. The enemy
// half (damage and the mana refund) belongs to a spec that casts Holy
// Shock on enemies and does not use this spell.
func (paladin *Paladin) registerLightsVigil() {
	if !paladin.Talents.LightsVigil || !paladin.isHealer() {
		return
	}

	paladin.lightsVigil.aura = paladin.RegisterAura(core.Aura{
		Label:    "Light's Vigil",
		ActionID: core.ActionID{SpellID: lightsVigilRanks[0].castID},
		Duration: lightsVigilDuration,
		OnExpire: func(*core.Aura, *core.Simulation) {
			paladin.lightsVigil.target = nil
		},
	})

	cd := core.Cooldown{Timer: paladin.NewTimer(), Duration: lightsVigilCooldown}
	for _, rank := range lightsVigilRanks {
		if paladin.Level < int32(rank.level) {
			break
		}
		paladin.registerLightsVigilRank(rank, cd)
	}
}

func (paladin *Paladin) registerLightsVigilRank(rank lightsVigilRank, cd core.Cooldown) {
	heal := paladin.registerLightsVigilHeal(rank)

	cast := paladin.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: rank.castID},
		SpellSchool: core.SpellSchoolHoly,
		DefenseType: core.DefenseTypeMagic,
		ProcMask:    core.ProcMaskEmpty,
		Flags:       core.SpellFlagHelpful | core.SpellFlagAPL,

		RequiredLevel: rank.level,

		SpellCode:         SpellCode_PaladinLightsVigil,
		ClassSpellMask:    PaladinSpellMaskLightsVigilCast,
		PushbackReduction: paladin.spiritualFocusPushbackReduction(PaladinSpellMaskLightsVigilCast),

		ManaCost: core.ManaCostOptions{FlatCost: rank.manaCost},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{GCD: core.GCDDefault, CastTime: lightsVigilCastTime},
			CD:          cd,
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, _ *core.Spell) {
			paladin.lightsVigil.target = target
			paladin.lightsVigil.heal = heal
			paladin.lightsVigil.aura.Activate(sim)
		},
	})
	paladin.markIlluminating(heal, cast.Cost.BaseCost)
}

func (paladin *Paladin) registerLightsVigilHeal(rank lightsVigilRank) *core.Spell {
	return paladin.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: rank.healID},
		SpellSchool: core.SpellSchoolHoly,
		DefenseType: core.DefenseTypeMagic,
		ProcMask:    core.ProcMaskSpellHealing,
		Flags:       core.SpellFlagHelpful | core.SpellFlagNoOnCastComplete,

		RequiredLevel: rank.level,

		SpellCode:      SpellCode_PaladinLightsVigilHeal,
		ClassSpellMask: PaladinSpellMaskLightsVigilHeal,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: lightsVigilCoefficient,
		ClientBaseDamage: rank.heal.Range(int(paladin.Level)),
		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.CalcAndDealHealing(sim, target, rank.heal.Roll(sim, int(paladin.Level)), spell.OutcomeHealingCrit)
		},
	})
}

// triggerLightsVigil is called when a Holy Shock cast heals target. If
// Light's Vigil is on that target it is consumed: the Holy Shock goes
// back off cooldown and the target's whole party is healed.
func (paladin *Paladin) triggerLightsVigil(sim *core.Simulation, target *core.Unit, holyShock *core.Spell) {
	state := &paladin.lightsVigil
	if state.aura == nil || !state.aura.IsActive() || state.target != target {
		return
	}
	state.aura.Deactivate(sim)
	holyShock.CD.Timer.Reset()

	party := paladin.Env.Raid.GetPlayerParty(target)
	for _, member := range party.Players {
		state.heal.Cast(sim, &member.GetCharacter().Unit)
	}
}
