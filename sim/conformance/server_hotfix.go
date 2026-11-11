package conformance

// VerdictServerHotfix is the verdict of a row that matches the client in
// every field but one Blizzard changed server side: the hotfix notes state
// the new value and neither the client nor its hotfix push carries it yet.
const VerdictServerHotfix = "server hotfix"

// serverHotfixCooldowns are the cooldowns the engine takes from Blizzard's
// hotfix notes ahead of the client, keyed "<class>/<spell>". A row reads as
// a server hotfix only when the engine holds exactly this value; any other
// difference stays a mismatch. Remove an entry once the client carries it.
var serverHotfixCooldowns = map[string]struct {
	cooldownMS int32
	reason     string
}{
	"shaman/Stormstrike": {8000, "Blizzard's 10 November hotfix (t/2403040): cooldown increased to 8 seconds; client 1.60.1.70888 and its pushes still carry 6000"},
}
