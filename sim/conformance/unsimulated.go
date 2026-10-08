package conformance

// VerdictDurationNotSimulated is a row whose every field matches except a
// client duration that belongs to an effect the engine deliberately does
// not simulate, so there is no engine aura to read it from.
const VerdictDurationNotSimulated = "not-simulated"

// unsimulatedDurations names, per class and spell, the client durations
// that describe an effect with no consequence in a sim result: the
// engine has no aura to carry them and faking one would only satisfy
// this report. The key is "<class slug>/<client spell name>".
var unsimulatedDurations = map[string]string{
	"shaman/Earth Shock": "interrupt lock-out of the target's school, never applied: the sim's targets never cast",
	"shaman/Frost Shock": "movement snare on the target, never applied: the sim's targets neither move nor flee",
}

// unsimulatedDurationReason returns why the client's duration for the
// spell has no engine counterpart, or "" when it should have one.
func unsimulatedDurationReason(classSlug, spellName string) string {
	return unsimulatedDurations[classSlug+"/"+spellName]
}
