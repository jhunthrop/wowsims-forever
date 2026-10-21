package warlock

import (
	"math"
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
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

	before := target.PseudoStats.SchoolDamageTakenMultiplier[stats.SchoolIndexShadow]
	aura.Activate(sim)
	got := target.PseudoStats.SchoolDamageTakenMultiplier[stats.SchoolIndexShadow] / before
	if math.Abs(got-1.20) > 1e-9 {
		t.Errorf("Shadow damage taken multiplier rose by x%v at 5/5, want x1.20", got)
	}
}
