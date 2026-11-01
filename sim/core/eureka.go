package core

import "time"

// eurekaModifier is one add-percent modifier of a class's Eureka! row: its
// percent and the client spell ids its class mask reaches.
type eurekaModifier struct {
	Percent int32
	Spells  map[int32]struct{}
}

func (mod eurekaModifier) covers(spell *Spell) bool {
	_, ok := mod.Spells[spell.ActionID.SpellID]
	return ok
}

// eurekaVariant is one class's Eureka! row (eureka_auto_gen.go). Damage is
// SPELLMOD_DAMAGE (misc 0, direct damage and healing), Periodic is
// SPELLMOD_DOT (misc 22) and Cost is SPELLMOD_COST (misc 14).
type eurekaVariant struct {
	SpellID    int32
	DurationMS int32
	CooldownMS int32
	Damage     eurekaModifier
	Periodic   eurekaModifier
	Cost       eurekaModifier
}

// eurekaSpellEffect is what one Eureka! modifier does to one spell; apply is
// called with +1 when the aura starts and -1 when it ends.
type eurekaSpellEffect struct {
	spell *Spell
	apply func(spell *Spell, sign float64)
}

// eurekaEffects lists the changes variant makes to spell. A covered heal
// takes the damage modifier as healing. A covered channel deals its client
// damage through a triggered spell's direct hits (Arcane Missiles, Blizzard,
// Rain of Fire, Hellfire), which the engine ticks, so the damage modifier
// reaches its ticks too.
func eurekaEffects(variant eurekaVariant, spell *Spell) []eurekaSpellEffect {
	var effects []eurekaSpellEffect
	if variant.Damage.covers(spell) {
		bonus := float64(variant.Damage.Percent) / 100
		heal := spell.ProcMask.Matches(ProcMaskSpellHealing)
		channel := spell.Flags.Matches(SpellFlagChanneled)
		effects = append(effects, eurekaSpellEffect{spell, func(spell *Spell, sign float64) {
			if heal {
				spell.DamageMultiplierAdditive += sign * bonus
				return
			}
			spell.ImpactDamageMultiplierAdditive += sign * bonus
			if channel {
				spell.PeriodicDamageMultiplierAdditive += sign * bonus
			}
		}})
	}
	if variant.Periodic.covers(spell) && !(variant.Damage.covers(spell) && spell.Flags.Matches(SpellFlagChanneled)) {
		bonus := float64(variant.Periodic.Percent) / 100
		effects = append(effects, eurekaSpellEffect{spell, func(spell *Spell, sign float64) {
			spell.PeriodicDamageMultiplierAdditive += sign * bonus
		}})
	}
	if variant.Cost.covers(spell) && spell.Cost != nil {
		percent := variant.Cost.Percent
		effects = append(effects, eurekaSpellEffect{spell, func(spell *Spell, sign float64) {
			spell.Cost.Multiplier += int32(sign) * percent
		}})
	}
	return effects
}

// applyEureka registers the character's class row of Eureka! as a major
// cooldown. Each row's modifiers reach only the spells its class masks name
// (eureka_auto_gen.go); the aura has no charges and lasts its full duration.
// A class the client gives no row (no Gnome of that class) gets nothing.
func applyEureka(character *Character) {
	variant, ok := eurekaByClass[character.Class]
	if !ok {
		return
	}

	var effects []eurekaSpellEffect
	character.OnSpellRegistered(func(spell *Spell) {
		effects = append(effects, eurekaEffects(variant, spell)...)
	})

	actionID := ActionID{SpellID: variant.SpellID}
	aura := character.RegisterAura(Aura{
		Label:    "Eureka!",
		ActionID: actionID,
		Duration: time.Duration(variant.DurationMS) * time.Millisecond,
		OnGain: func(_ *Aura, _ *Simulation) {
			for _, effect := range effects {
				effect.apply(effect.spell, 1)
			}
		},
		OnExpire: func(_ *Aura, _ *Simulation) {
			for _, effect := range effects {
				effect.apply(effect.spell, -1)
			}
		},
	})

	spell := character.RegisterSpell(SpellConfig{
		ActionID: actionID,
		Flags:    SpellFlagNoOnCastComplete,
		Cast: CastConfig{
			CD: Cooldown{
				Timer:    character.NewTimer(),
				Duration: time.Duration(variant.CooldownMS) * time.Millisecond,
			},
		},
		ApplyEffects: func(sim *Simulation, _ *Unit, _ *Spell) {
			aura.Activate(sim)
		},
	})

	character.AddMajorCooldown(MajorCooldown{
		Spell:    spell,
		Type:     CooldownTypeDPS,
		SelfBuff: true,
	})
}
