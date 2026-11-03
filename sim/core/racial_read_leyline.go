package core

import "time"

// Read Leyline (client 1259705, "Read Ley Line" in Blizzard's notes) is the
// Alliance Skyborne's second active. It casts Energized (1270842): aura 110
// (power regeneration percent, misc 0 mana) at 50 and aura 88 (health
// regeneration percent) at 50, for 15 s. SpellCooldowns gives a 120000 ms
// recovery and a 1500 ms global cooldown (1.60.1.70753). Aura 110 scales the
// spirit-based mana regeneration, which in the engine is
// PseudoStats.SpiritRegenMultiplier (Innervate's lever); the health half
// changes no damage dealt and is not modelled. Standing at a leyline makes
// the caster Attuned (1259691) for more; a raid boss room has none, so the
// base 50% is the one modelled.
//
// It is registered for an APL to cast and never fired automatically: it
// costs a global cooldown, and inside the five-second rule spirit regen is
// mostly off, so firing it on cooldown would cost a damage dealer a cast for
// little mana.
const (
	readLeylineSpellID        int32 = 1259705
	readLeylineBuffSpellID    int32 = 1270842
	readLeylineDuration             = 15 * time.Second
	readLeylineCooldown             = 2 * time.Minute
	readLeylineRegenIncrement       = 0.5
)

func applyReadLeyline(character *Character) {
	aura := character.RegisterAura(Aura{
		Label:    "Energized",
		ActionID: ActionID{SpellID: readLeylineBuffSpellID},
		Duration: readLeylineDuration,
		OnGain: func(aura *Aura, sim *Simulation) {
			aura.Unit.PseudoStats.SpiritRegenMultiplier += readLeylineRegenIncrement
		},
		OnExpire: func(aura *Aura, sim *Simulation) {
			aura.Unit.PseudoStats.SpiritRegenMultiplier -= readLeylineRegenIncrement
		},
	})

	spell := character.RegisterSpell(SpellConfig{
		ActionID: ActionID{SpellID: readLeylineSpellID},
		Flags:    SpellFlagNoOnCastComplete,
		Cast: CastConfig{
			DefaultCast: Cast{GCD: GCDDefault},
			CD: Cooldown{
				Timer:    character.NewTimer(),
				Duration: readLeylineCooldown,
			},
		},
		ApplyEffects: func(sim *Simulation, _ *Unit, _ *Spell) {
			aura.Activate(sim)
		},
	})

	character.AddMajorCooldown(MajorCooldown{
		Spell:    spell,
		Type:     CooldownTypeMana,
		SelfBuff: true,
		ShouldActivate: func(*Simulation, *Character) bool {
			return false // APL only; see above
		},
	})
}
