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
	auraDecreaseSpeed     int32 = 33
	auraChannelDeathItem  int32 = 86
	reasonUnmodeledEffect       = "the client's duration belongs to a control effect (slow, stun, root, fear, silence, interrupt lockout, Shadowburn shard marker) no sim number reads, so the engine registers no aura for it by design"
	reasonFullBuffsAura         = "the engine aura on the target is the same object core.FullBuffs.Debuffs already made permanent with core.MakePermanent, so its registered duration cannot be read here (the spell's own constructor sets the client's duration)"
)

// durationReadingClasses are the client class slugs whose goldens have
// adopted these readings. A class joins by adding its slug here and
// regenerating its golden; the other classes' goldens keep the plain
// numeric comparison until their lanes make that choice.
var durationReadingClasses = map[string]bool{"mage": true, "warlock": true}

var unmodeledControlAuras = map[int32]bool{
	auraConfuse: true, auraFear: true, auraStun: true, auraRoot: true,
	auraSilence: true, auraDecreaseSpeed: true, auraChannelDeathItem: true,
}

// durationReading returns the explanation when row's duration column is an
// artefact of the report rather than a disagreement, and "" when the
// numbers are to be compared:
//
//   - the engine found no aura and every duration-bearing client effect is a
//     control effect the sim does not model (reasonUnmodeledEffect);
//   - the engine's only matching aura sits on the target and was made
//     permanent by the full-buffs debuff set (reasonFullBuffsAura).
//
// An aura on the caster that carries NeverExpires is not read this way: that
// is the engine's own registration and a real disagreement (Evocation).
func durationReading(row Row, client spellconst.Spell, spell *core.Spell, siblingIDs map[int32]bool) string {
	if !durationReadingClasses[row.ClassSlug] || client.DurationMS <= 0 {
		return ""
	}
	if !row.EngineDurationFound {
		if onlyUnmodeledControlEffects(client) {
			return reasonUnmodeledEffect
		}
		return ""
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
