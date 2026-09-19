package core

import (
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
)

// multidotSimAndSpell builds a timelineSim and returns the FakeAgent's
// dottable "fakedot" spell (registered by dot_test.go's init), which is
// the only spell available inside sim/core tests with a real Dot config.
func multidotSimAndSpell(t *testing.T, timeline []*proto.TargetCountAt) (*Simulation, *Spell) {
	t.Helper()
	sim := timelineSim(t, timeline)
	fa, ok := sim.Raid.Parties[0].Players[0].(*FakeAgent)
	if !ok {
		t.Fatalf("player is a %T, want *FakeAgent", sim.Raid.Parties[0].Players[0])
	}
	if fa.Spell == nil {
		t.Fatal("FakeAgent.Spell is nil; expected the fakedot spell to be registered by Initialize")
	}
	return sim, fa.Spell
}

// A Multidot APL action's maxDots is fixed once, at construction time. If
// the timeline later shrinks the active prefix below that count, IsReady
// must not walk off the end of the shrunken TargetUnits slice: it has to
// stay bounded by however many targets are active right now, not just by
// maxDots.
func TestMultidotIsReadySurvivesAShrinkingPrefix(t *testing.T) {
	sim, spell := multidotSimAndSpell(t, []*proto.TargetCountAt{
		{AtSeconds: 0, Count: 3},
		{AtSeconds: 40, Count: 1},
	})

	action := &APLActionMultidot{
		spell: spell,
		// As if this were configured by newActionMultidot while 3 targets
		// were the pool's ceiling.
		maxDots:    3,
		maxOverlap: &APLValueConst{valType: proto.APLValueType_ValueTypeDuration},
	}

	sim.Encounter.SetActiveTargetCount(sim, 1)
	if got := len(sim.Encounter.TargetUnits); got != 1 {
		t.Fatalf("active count after shrinking = %d, want 1", got)
	}

	// Put the dot up on the one remaining target with plenty of remaining
	// duration, so IsReady's "is this target ready to (re)dot" check is
	// false at i=0 and the loop walks on to i=1 - exactly where an
	// unbounded loop indexes past the end of the shrunken TargetUnits
	// slice (len 1) instead of stopping.
	dot := spell.Dot(sim.Encounter.TargetUnits[0])
	dot.Apply(sim)
	if !dot.IsActive() {
		t.Fatal("dot did not apply; test setup is broken")
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("IsReady panicked against a shrunken prefix (maxDots=3, active=1): %v", r)
		}
	}()
	action.IsReady(sim)
}

// newActionMultidot must clamp maxDots against the target POOL, not the
// timeline's initial active count: a dungeon that starts at one target and
// grows later must still let a Multidot action reach every target the
// fight can ever present, and must not warn just because the fight starts
// small.
func TestMultidotMaxDotsIsClampedToThePoolNotTheInitialActiveCount(t *testing.T) {
	sim := timelineSim(t, []*proto.TargetCountAt{
		{AtSeconds: 0, Count: 1},
		{AtSeconds: 40, Count: 3},
	})
	fa, ok := sim.Raid.Parties[0].Players[0].(*FakeAgent)
	if !ok {
		t.Fatalf("player is a %T, want *FakeAgent", sim.Raid.Parties[0].Players[0])
	}
	if got := sim.GetNumTargets(); got != 1 {
		t.Fatalf("active count at construction = %d, want the timeline's initial 1", got)
	}

	rot := &APLRotation{unit: &fa.Character.Unit}
	impl := rot.newActionMultidot(&proto.APLActionMultidot{
		SpellId: ActionID{SpellID: 42}.ToProto(),
		MaxDots: 3,
	})
	action, ok := impl.(*APLActionMultidot)
	if !ok {
		t.Fatalf("newActionMultidot returned %T, want *APLActionMultidot", impl)
	}
	if action.maxDots != 3 {
		t.Errorf("maxDots = %d, want 3 (the pool size); it must not be clamped to the initial active count of 1", action.maxDots)
	}
	if len(rot.curWarnings) != 0 {
		t.Errorf("curWarnings = %v, want none: a fight that merely starts small should not warn", rot.curWarnings)
	}
}

// Asking for more dots than the fight can EVER present (more than the
// pool, not just more than the current active count) still deserves a
// warning, and still gets clamped down to what's possible.
func TestMultidotWarnsWhenMaxDotsExceedsThePool(t *testing.T) {
	sim := timelineSim(t, []*proto.TargetCountAt{
		{AtSeconds: 0, Count: 1},
		{AtSeconds: 40, Count: 3},
	})
	fa, ok := sim.Raid.Parties[0].Players[0].(*FakeAgent)
	if !ok {
		t.Fatalf("player is a %T, want *FakeAgent", sim.Raid.Parties[0].Players[0])
	}

	rot := &APLRotation{unit: &fa.Character.Unit}
	impl := rot.newActionMultidot(&proto.APLActionMultidot{
		SpellId: ActionID{SpellID: 42}.ToProto(),
		MaxDots: 5,
	})
	action, ok := impl.(*APLActionMultidot)
	if !ok {
		t.Fatalf("newActionMultidot returned %T, want *APLActionMultidot", impl)
	}
	if action.maxDots != 3 {
		t.Errorf("maxDots = %d, want 3 (clamped to the pool size)", action.maxDots)
	}
	if len(rot.curWarnings) == 0 {
		t.Error("want a validation warning: 5 dots requested but the pool never has more than 3 targets")
	}
}
