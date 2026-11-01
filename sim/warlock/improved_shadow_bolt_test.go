package warlock

import (
	"math"
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// TestImprovedShadowBoltDebuffHasNoChargesAndLastsTwelveSeconds pins the
// Shadow Vulnerability debuff (spell 17794) to the client: SpellDuration
// 12000 ms and no SpellAuraOptions row, so no proc charges. The +20% Shadow
// damage at 5/5 holds for the full 12 s however many spells land.
func TestImprovedShadowBoltDebuffHasNoChargesAndLastsTwelveSeconds(t *testing.T) {
	sim, built, target := newWarlockForDamageTest(t, "--05") // Destruction field 2, rank 5.

	aura := built.ImprovedShadowBoltAuras.Get(target)
	if aura.MaxStacks != 0 {
		t.Errorf("Shadow Vulnerability MaxStacks = %d, want 0 (no proc charges)", aura.MaxStacks)
	}
	if aura.Duration != 12*time.Second || core.ImprovedShadowBoltDuration != 12*time.Second {
		t.Errorf("Shadow Vulnerability duration = %v, want 12s", aura.Duration)
	}

	shadowBolt := built.ShadowBolt[len(built.ShadowBolt)-1]
	at := built.AttackTables[target.UnitIndex][shadowBolt.CastType]
	before := shadowBolt.TargetDamageMultiplier(at, false)
	aura.Activate(sim)
	if got := shadowBolt.TargetDamageMultiplier(at, false) / before; math.Abs(got-1.20) > 1e-9 {
		t.Errorf("Shadow Bolt's target multiplier rose by x%v at 5/5, want x1.20", got)
	}
}

// Shadow Vulnerability is aura 270 (school-mask damage taken from the
// caster), misc 32: only the warlock's own Shadow damage takes it, as the
// talent text says ("from your attacks"). The demon's Lash of Pain and
// every other unit's Shadow damage take nothing.
func TestImprovedShadowBoltReachesOnlyTheWarlocksOwnShadowDamage(t *testing.T) {
	sim, warlock := newPetInheritanceTestWarlock(t, proto.WarlockOptions_Succubus)
	if warlock.Talents.ImprovedShadowBolt != 5 {
		t.Fatalf("Improved Shadow Bolt = %d, want 5", warlock.Talents.ImprovedShadowBolt)
	}
	target := sim.Encounter.TargetUnits[0]
	lash := warlock.Succubus.primaryAbility
	at := warlock.Succubus.AttackTables[target.UnitIndex][lash.CastType]
	before := lash.TargetDamageMultiplier(at, false)
	takenBefore := target.PseudoStats.SchoolDamageTakenMultiplier[stats.SchoolIndexShadow]
	warlock.ImprovedShadowBoltAuras.Get(target).Activate(sim)
	if got := lash.TargetDamageMultiplier(at, false) / before; math.Abs(got-1) > 1e-9 {
		t.Errorf("Lash of Pain's target multiplier moved by x%v under the warlock's Shadow Vulnerability, want x1", got)
	}
	if got := target.PseudoStats.SchoolDamageTakenMultiplier[stats.SchoolIndexShadow] / takenBefore; math.Abs(got-1) > 1e-9 {
		t.Errorf("the target's Shadow damage taken moved by x%v for every attacker, want x1", got)
	}
}
