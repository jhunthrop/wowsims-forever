package conformance

import (
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/spellconst"
)

// VerdictUnmodeledDuration is the verdict of a row whose every field matches
// and whose duration column is read through a report-side reading below
// instead of being compared as numbers.
const VerdictUnmodeledDuration = "unmodeled-duration"

// Client SpellEffect.Effect and EffectAura ids of the effects whose duration
// no sim number reads: a movement slow, a stun, a root, a fear, a confusion,
// a silence, an interrupt's school lockout, and the Shadowburn marker aura
// that grants a Soul Shard if the target dies (this fork's Shadowburn costs
// mana and no Soul Shard resource exists).
const (
	effectInterruptCast   int32 = 68
	auraConfuse           int32 = 5
	auraFear              int32 = 7
	auraStun              int32 = 12
	auraRoot              int32 = 26
	auraSilence           int32 = 27
	auraIncreaseSpeed     int32 = 31
	auraDecreaseSpeed     int32 = 33
	auraChannelDeathItem  int32 = 86
	reasonUnmodeledEffect       = "the client's duration belongs to a control or movement effect (slow, speed buff, stun, root, fear, silence, interrupt lockout, Shadowburn shard marker) no sim number reads, so the engine registers no aura for it by design"
	reasonFullBuffsAura         = "the engine aura on the target is the same object core.FullBuffs.Debuffs already made permanent with core.MakePermanent, so its registered duration cannot be read here (the spell's own constructor sets the client's duration)"
)

// durationPayloadSpells maps a cast spell whose own duration_ms is not the
// duration of anything it does to the client spell that carries the
// payload. A trap's duration_ms (60000 on every rank) is its armed lifetime
// on the ground; the burn, DoT or freeze it leaves lives on a separate
// "<Trap> Effect" spell per rank. The engine resolves the trap on cast and
// models only the payload, so the payload's duration is the comparable one.
//
// Build 1.60.1.70291 renamed the Freezing and Immolation payload spells to
// the trap's own name ("Freezing Trap Effect" became "Freezing Trap"), so
// for those two the payload is the same-rank spell of that name with another
// id; Explosive Trap's payload keeps its "Effect" name.
var durationPayloadSpells = map[string]string{
	"Freezing Trap":   "Freezing Trap",
	"Immolation Trap": "Immolation Trap",
	"Explosive Trap":  "Explosive Trap Effect",
}

// unsimulatedDurations names, per class and spell, the client durations of
// effects with no consequence in a sim result, with the reason each has no
// engine aura. They read as unmodeled-duration like the generic control
// effects but keep their own explanation. The key is "<class slug>/<client
// spell name>".
var unsimulatedDurations = map[string]string{
	"shaman/Earth Shock": "interrupt lock-out of the target's school, never applied: the sim's targets never cast",
	"shaman/Frost Shock": "movement snare on the target, never applied: the sim's targets neither move nor flee",

	"hunter/Sniper Shot": "the 10 s range increase (a range spell mod, aura 107) has no reader: the sim's hunter never moves and has no range band",

	"warrior/Hamstring":     "the 15 s movement-speed snare has no reader in a sim with no movement",
	"warrior/Piercing Howl": "the 6 s daze (see piercing_howl.go) has no reader in a sim with no movement",
	"warrior/Bloodthirst":   "the 10 s movement-speed buff has no reader in a sim with no movement",
	"warrior/Mortal Strike": "the 10 s healing-reduction debuff has no reader: no healer on the target",
	"warrior/Pummel":        "the school lockout after an interrupt has no reader: the sim's targets never cast",
}

// durationSpellFor returns the client spell whose duration_ms the engine is
// compared against: the cast spell itself, or its payload spell of the same
// rank when durationPayloadSpells names one.
func durationSpellFor(clientClass spellconst.Class, cast spellconst.Spell) spellconst.Spell {
	payloadName, ok := durationPayloadSpells[cast.Name]
	if !ok {
		return cast
	}
	for _, candidate := range clientClass.Ranks(payloadName) {
		if candidate.Rank == cast.Rank && candidate.ID != cast.ID {
			return candidate
		}
	}
	return cast
}

var unmodeledControlAuras = map[int32]bool{
	auraConfuse: true, auraFear: true, auraStun: true, auraRoot: true,
	auraSilence: true, auraIncreaseSpeed: true, auraDecreaseSpeed: true, auraChannelDeathItem: true,
}

// durationReading returns the explanation when row's duration column is an
// artefact of the report rather than a disagreement, and "" when the
// numbers are to be compared:
//
//   - the engine found no aura and the spell is named in unsimulatedDurations
//     (that row's own reason);
//   - every duration-bearing client effect is a control or movement effect
//     the sim does not model and the engine's number (if any) differs
//     (reasonUnmodeledEffect);
//   - the engine's only matching aura sits on the target and was made
//     permanent by the full-buffs debuff set (reasonFullBuffsAura).
//
// An aura on the caster that carries NeverExpires is not read this way: that
// is the engine's own registration and a real disagreement (Evocation).
func durationReading(row Row, client spellconst.Spell, spell *core.Spell, siblingIDs map[int32]bool) string {
	if client.DurationMS <= 0 {
		return ""
	}
	if reason := unsimulatedDurations[row.ClassSlug+"/"+client.Name]; reason != "" && !row.EngineDurationFound {
		return reason
	}
	// A control effect's duration is never what an engine aura or Dot found
	// on the spell measures (Pounce's 2 s stun beside its 18 s bleed), so
	// the numbers are only compared when they happen to agree.
	if onlyUnmodeledControlEffects(client) && row.EngineDurationMS != client.DurationMS {
		return reasonUnmodeledEffect
	}
	if row.EngineDurationMS == -1 && permanentOnTarget(spell, siblingIDs) {
		return reasonFullBuffsAura
	}
	return ""
}

// onlyUnmodeledControlEffects is true when the spell has at least one
// control effect and no other aura effect that could carry the duration.
func onlyUnmodeledControlEffects(client spellconst.Spell) bool {
	control := false
	for _, e := range client.Effects {
		switch {
		case e.Effect == effectInterruptCast:
			control = true
		case e.Effect == effectApplyAura && unmodeledControlAuras[e.Aura]:
			control = true
		case e.Effect == effectApplyAura && e.Aura != 0:
			return false
		}
	}
	return control
}

// permanentOnTarget reports whether the spell's current target carries a
// matching aura that is NeverExpires.
func permanentOnTarget(spell *core.Spell, siblingIDs map[int32]bool) bool {
	if spell.Unit == nil || spell.Unit.CurrentTarget == nil {
		return false
	}
	aura := matchingAura(spell.Unit.CurrentTarget.GetAuras(), siblingIDs)
	return aura != nil && aura.Duration == core.NeverExpires
}
