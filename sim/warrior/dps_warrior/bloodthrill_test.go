package dpswarrior

import "testing"

// TestBloodthrillOpensItsOwnOverpowerWindow exercises Bloodthrill's
// trigger end to end: a Main Hand hit landing on a Rend-afflicted
// target has a per-rank chance to activate BloodthrillAura, and
// Overpower's own ExtraCastCondition must read that aura, not only the
// vanilla dodge-proc OverpowerAura (overpower.go).
//
// Rend doubles as both the affliction and the Main Hand hit: calling
// its ApplyEffects once applies the Dot, and every call after is itself
// a landed ProcMaskMeleeMHSpecial hit against a target its own Dot is
// now ticking on.
//
// Rank 5 is 20% a landed hit (bloodthrillProcChance), so 300 trials
// without a single proc has probability 0.8^300, far below any flake
// budget.
func TestBloodthrillOpensItsOwnOverpowerWindow(t *testing.T) {
	talents := talentStringWithRank(t, emptyWarriorTalents, "bloodthrill", 5)
	war, sim := buildWarriorAndSimForCostTest(t, talents)
	target := sim.Encounter.TargetUnits[0]

	if war.BloodthrillAura == nil {
		t.Fatal("a warrior with Bloodthrill talented has no BloodthrillAura registered")
	}
	if war.Overpower.CanCast(sim, target) {
		t.Fatal("Overpower is castable before either Overpower aura has activated")
	}

	for i := 0; i < 300 && !war.BloodthrillAura.IsActive(); i++ {
		war.Rend.ApplyEffects(sim, target, war.Rend.Spell)
	}

	if !war.BloodthrillAura.IsActive() {
		t.Fatal("BloodthrillAura never activated across 300 Main Hand hits on a Rend-afflicted target at rank 5 (20%)")
	}
	if !war.Overpower.CanCast(sim, target) {
		t.Error("Overpower is not castable while BloodthrillAura is active")
	}
}

// Without the talent, BloodthrillAura is nil and Overpower's extra cast
// condition must not panic on the nil check added for it.
func TestBloodthrillAuraIsNilWithoutTheTalent(t *testing.T) {
	war, sim := buildWarriorAndSimForCostTest(t, emptyWarriorTalents)
	if war.BloodthrillAura != nil {
		t.Error("a warrior with no points in Bloodthrill has a BloodthrillAura")
	}

	target := sim.Encounter.TargetUnits[0]
	if war.Overpower.CanCast(sim, target) {
		t.Error("Overpower is castable with neither proc aura active")
	}
}
