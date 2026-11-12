package warlock

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

// TestBaneOfHavocTakesTheGlobalCooldown pins the 10 Nov hotfix row for Bane
// of Havoc (1225228): push 113766 on client 1.60.1.70888 adds SpellCooldowns
// 102518 (StartRecoveryTime 1500) and SpellCategories 174286
// (StartRecoveryCategory 133), the ordinary 1.5 s global cooldown. The 70790
// tables carried no cooldown row at all, which is why the cast was GCD-less
// here. Sim-accuracy 2026-11-12, warlocks pass 3.
func TestBaneOfHavocTakesTheGlobalCooldown(t *testing.T) {
	_, built, _, _ := newWarlockForDamageTestMultiTarget(t, "--0000000000001") // Destruction field 13.

	if built.BaneOfHavoc == nil {
		t.Fatal("level-60 warlock with the Bane of Havoc talent has no Bane of Havoc spell")
	}
	if got, want := built.BaneOfHavoc.DefaultCast.GCD, core.GCDDefault; got != want {
		t.Errorf("Bane of Havoc DefaultCast.GCD = %v, want %v (push 113766, SpellCooldowns 102518)", got, want)
	}
}

// TestCurseOfShadowIsNotRegistered: Forever's client has no Curse of Shadow.
// Neither 17862 nor 17937 has a row in any spell table of 1.60.1.70790 or
// 70888 (Curse of the Elements covers every magic school), so a level-60
// warlock must not be able to cast one. Sim-accuracy 2026-11-12, warlocks
// pass 3.
func TestCurseOfShadowIsNotRegistered(t *testing.T) {
	_, built, _ := newBareWarlockForDamageTest(t)

	for _, id := range []int32{17862, 17937} {
		if built.GetSpell(core.ActionID{SpellID: id}) != nil {
			t.Errorf("level-60 warlock registers Curse of Shadow %d, a spell the Forever client does not have", id)
		}
	}
}
