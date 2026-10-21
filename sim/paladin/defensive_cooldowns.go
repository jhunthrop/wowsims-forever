package paladin

import (
	"math"
	"time"

	"github.com/wowsims/classic/sim/core"
)

// The paladin's emergency defensives: Templar's Bulwark (a Protection
// talent), Divine Protection and Divine Shield. All three apply
// Forbearance (forbearance.go) and share the five minute cooldown Sacred
// Duty shortens by 30 and 60 seconds.
//
// Divine Protection and Divine Shield are registered under their Classic
// Era ids (498/5573 and 642/1020). The client table also carries Season
// of Discovery ids for Divine Protection (458312/458371, a 50% damage
// reduction for 9/12 s); the trainables list both but Holy Shield's live
// talent text names the Era spell, so the Era spells are taken as the live
// ones (assumption: unconfirmed).
const (
	defensiveCooldown = 5 * time.Minute

	// Templar's Bulwark (node 105625, spell 1311015): "When activated, this
	// ability grants you an absorb shield equal to 100% of your maximum
	// health for 8 sec. Applies Forbearance for 1 min. Cannot be cast
	// while Forbearance is active." 110 mana, no global cooldown.
	templarsBulwarkActionID    = 1311015
	templarsBulwarkLevel       = 30
	templarsBulwarkManaCost    = 110
	templarsBulwarkDuration    = 8 * time.Second
	templarsBulwarkAbsorbShare = 1.0
)

// immunityRank is one rank of an immunity cooldown, as the client states
// it (spellconst/paladin.json: school immunity for all schools).
type immunityRank struct {
	level    int32
	spellID  int32
	manaCost float64
	duration time.Duration
}

var (
	// divineProtectionRanks: "Protects the paladin from all damage for
	// the duration, but cannot attack or use physical abilities" -
	// effects: pacify (aura 25) and immunity to physical and every other
	// school (aura 39, misc 1 and 126).
	divineProtectionRanks = []immunityRank{
		{level: 6, spellID: 498, manaCost: 15, duration: 6 * time.Second},
		{level: 18, spellID: 5573, manaCost: 35, duration: 8 * time.Second},
	}

	// divineShieldRanks: immunity to every school, with the paladin's
	// damage dealt reduced by 50% (aura 79, amount -50).
	divineShieldRanks = []immunityRank{
		{level: 34, spellID: 642, manaCost: 75, duration: 10 * time.Second},
		{level: 50, spellID: 1020, manaCost: 110, duration: 12 * time.Second},
	}
)

const divineShieldDamageDealtMultiplier = 0.5

func (paladin *Paladin) registerDefensiveCooldowns() {
	paladin.registerTemplarsBulwark()
	paladin.registerDivineProtection()
	paladin.registerDivineShield()
}

// defensiveCooldownDuration is the five minute cooldown less Sacred Duty.
func (paladin *Paladin) defensiveCooldownDuration() time.Duration {
	return defensiveCooldown - sacredDutyCooldownReductionPerRank*time.Duration(paladin.Talents.SacredDuty)
}

// newDefensiveCooldown registers one Forbearance-bound defensive cooldown
// and returns its spell. The effect runs on cast.
func (paladin *Paladin) newDefensiveCooldown(config core.SpellConfig, effect core.ApplySpellResults) *core.Spell {
	config.Flags |= core.SpellFlagAPL | SpellFlag_Forbearance
	config.Cast.CD = core.Cooldown{
		Timer:    paladin.NewTimer(),
		Duration: paladin.defensiveCooldownDuration(),
	}
	config.ApplyEffects = effect
	return paladin.RegisterSpell(config)
}

func (paladin *Paladin) registerTemplarsBulwark() {
	if !paladin.Talents.TemplarsBulwark || paladin.Level < templarsBulwarkLevel {
		return
	}

	aura := paladin.RegisterAura(core.Aura{
		Label:    "Templar's Bulwark",
		ActionID: core.ActionID{SpellID: templarsBulwarkActionID},
		Duration: templarsBulwarkDuration,
	})
	shield := newDamageAbsorb(&paladin.Unit, aura, nil)

	paladin.newDefensiveCooldown(core.SpellConfig{
		ActionID:        core.ActionID{SpellID: templarsBulwarkActionID},
		SpellSchool:     core.SpellSchoolHoly,
		RequiredLevel:   templarsBulwarkLevel,
		RelatedSelfBuff: aura,
		ManaCost:        core.ManaCostOptions{FlatCost: templarsBulwarkManaCost},
	}, func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
		shield.Grant(sim, spell, templarsBulwarkAbsorbShare*paladin.MaxHealth())
	})
}

func (paladin *Paladin) registerDivineProtection() {
	rank, ok := highestImmunityRank(divineProtectionRanks, paladin.Level)
	if !ok {
		return
	}

	aura := paladin.newImmunityAura("Divine Protection", rank, core.Aura{
		OnGain: func(_ *core.Aura, sim *core.Simulation) {
			paladin.AutoAttacks.CancelAutoSwing(sim)
		},
		OnExpire: func(_ *core.Aura, sim *core.Simulation) {
			paladin.AutoAttacks.EnableAutoSwing(sim)
		},
	})
	paladin.registerImmunityCooldown(rank, aura)
}

func (paladin *Paladin) registerDivineShield() {
	rank, ok := highestImmunityRank(divineShieldRanks, paladin.Level)
	if !ok {
		return
	}

	aura := paladin.newImmunityAura("Divine Shield", rank, core.Aura{
		OnGain: func(aura *core.Aura, _ *core.Simulation) {
			aura.Unit.PseudoStats.DamageDealtMultiplier *= divineShieldDamageDealtMultiplier
		},
		OnExpire: func(aura *core.Aura, _ *core.Simulation) {
			aura.Unit.PseudoStats.DamageDealtMultiplier /= divineShieldDamageDealtMultiplier
		},
	})
	paladin.registerImmunityCooldown(rank, aura)
}

func highestImmunityRank(ranks []immunityRank, level int32) (immunityRank, bool) {
	for i := len(ranks) - 1; i >= 0; i-- {
		if level >= ranks[i].level {
			return ranks[i], true
		}
	}
	return immunityRank{}, false
}

// newImmunityAura is the timed aura of an immunity cooldown, with the
// extra effects the particular spell carries in hooks.
func (paladin *Paladin) newImmunityAura(label string, rank immunityRank, hooks core.Aura) *core.Aura {
	hooks.Label = label
	hooks.ActionID = core.ActionID{SpellID: rank.spellID}
	hooks.Duration = rank.duration
	return paladin.RegisterAura(hooks)
}

// registerImmunityCooldown registers the cast of an immunity: an absorb
// that never runs out for the aura's duration.
func (paladin *Paladin) registerImmunityCooldown(rank immunityRank, aura *core.Aura) {
	immunity := newDamageAbsorb(&paladin.Unit, aura, nil)

	paladin.newDefensiveCooldown(core.SpellConfig{
		ActionID:        aura.ActionID,
		SpellSchool:     core.SpellSchoolHoly,
		RequiredLevel:   int(rank.level),
		RelatedSelfBuff: aura,
		ManaCost:        core.ManaCostOptions{FlatCost: rank.manaCost},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{GCD: core.GCDDefault},
		},
	}, func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
		immunity.Grant(sim, spell, math.Inf(1))
	})
}
