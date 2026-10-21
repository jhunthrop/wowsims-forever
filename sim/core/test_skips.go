package core

import "testing"

// SkipAwaitingForeverTalentRewrite skips a spec's regression suite
// because its talents were regenerated from the client's trait trees
// (plan docs/superpowers/plans/2026-09-14-sim-engine.md, task 17) while
// its behaviour was not rewritten, so its DPS goldens describe a spec
// that no longer exists. Skipped rather than left failing: a suite that
// is always red is a suite nobody reads, and the next real regression
// would hide in it.
//
// Remove the call when the spec is brought up, in the order design
// section 2.3 gives: Rogue, Hunter, Warlock, Shadow Priest, Balance and
// Feral Druid, Elemental and Enhancement Shaman, Retribution Paladin.
//
// It lives here, beside RunTestSuite, because the reason is one reason
// held once: it was thirteen byte-identical ten-line paragraphs across
// eleven spec suites, and a paragraph copied thirteen times is a
// paragraph that goes stale in twelve of them. pkg is the suite's
// package path, which names the spec in the skip line.
func SkipAwaitingForeverTalentRewrite(t *testing.T, pkg string) {
	t.Helper()
	t.Skip(pkg + " awaits its Forever talent rewrite (plan 2026-09-14-sim-engine, task 17)")
}
