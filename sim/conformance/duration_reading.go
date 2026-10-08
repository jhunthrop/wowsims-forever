package conformance

import "github.com/wowsims/classic/sim/core/spellconst"

// Two readings of the client's duration_ms that rowFor applies before the
// comparison, so a number the client states about something other than
// what the engine models is not scored as an engine defect.

// durationPayloadSpells maps a cast spell whose own duration_ms is not the
// duration of anything it does to the client spell that carries the
// payload. A trap's duration_ms (60000 on every rank) is its armed
// lifetime on the ground before something walks over it; the burn, the
// explosion's DoT or the freeze it leaves lives on a separate "<Trap>
// Effect" spell per rank (Immolation Trap Effect 15000, Explosive Trap
// Effect 20000, Freezing Trap Effect 10000/15000/20000). The engine
// resolves the trap on cast and models only the payload, so the payload's
// duration is the one comparable with the engine's Dot.
var durationPayloadSpells = map[string]string{
	"Freezing Trap":   "Freezing Trap Effect",
	"Immolation Trap": "Immolation Trap Effect",
	"Explosive Trap":  "Explosive Trap Effect",
}

// Effects of the client's SpellEffect table that change how a unit moves,
// whether it can act, or what it may cast: the stun, speed-up and slow
// auras, and the interrupt effect. The engine has no movement, its
// encounter targets are never crowd controlled and cast nothing, so a
// duration that belongs only to these has nothing in the engine to be
// compared with; it is utility, not a simmed quantity.
const (
	auraStun          int32 = 12
	auraIncreaseSpeed int32 = 31
	auraDecreaseSpeed int32 = 33
	effectInterrupt   int32 = 68
)

func isUnsimmedMovementAura(aura int32) bool {
	switch aura {
	case auraStun, auraIncreaseSpeed, auraDecreaseSpeed:
		return true
	default:
		return false
	}
}

// durationSpellFor returns the client spell whose duration_ms the engine
// is compared against: the cast spell itself, or its payload spell of the
// same rank when durationPayloadSpells names one.
func durationSpellFor(clientClass spellconst.Class, cast spellconst.Spell) spellconst.Spell {
	payloadName, ok := durationPayloadSpells[cast.Name]
	if !ok {
		return cast
	}
	for _, candidate := range clientClass.Ranks(payloadName) {
		if candidate.Rank == cast.Rank {
			return candidate
		}
	}
	return cast
}

// durationIsUnsimmedMovement is true when the spell applies at least one
// movement, crowd-control or interrupt effect and no aura of any other
// kind (Wing Clip's snare, Strider Kick's speed buff, Freezing Trap's
// freeze, Kick's lockout), i.e. its duration_ms describes nothing the
// engine models. A spell with any other aura keeps its duration.
func durationIsUnsimmedMovement(spell spellconst.Spell) bool {
	sawControl := false
	for _, effect := range spell.Effects {
		switch {
		case effect.Effect == effectInterrupt:
			sawControl = true
		case effect.Aura == 0:
		case isUnsimmedMovementAura(effect.Aura):
			sawControl = true
		default:
			return false
		}
	}
	return sawControl
}

// clientDurationMS is the duration_ms rowFor scores: the payload spell's
// when there is one, and 0 ("the client states none") when the preset opts
// into skipping durations that are only movement or crowd control.
func clientDurationMS(clientClass spellconst.Class, cast spellconst.Spell, skipUnsimmedMovement bool) int32 {
	source := durationSpellFor(clientClass, cast)
	if skipUnsimmedMovement && durationIsUnsimmedMovement(source) {
		return 0
	}
	return source.DurationMS
}
