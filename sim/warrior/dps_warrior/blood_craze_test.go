package dpswarrior

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

// Blood Craze has three independent, deterministic triggers - no RNG
// of its own - so each is checkable by forcing exactly the SpellResult
// that trigger reads, then draining the 6s heal's periodic ticks.
func TestBloodCrazeHealsOnEachTrigger(t *testing.T) {
	// Rank 3 is 3% of max health over 6 sec (bloodCrazeHealthPercent).
	const rank = 3

	t.Run("crit taken", func(t *testing.T) {
		war, sim := buildWarriorAndSimForCostTest(t, talentStringWithRank(t, emptyWarriorTalents, "blood_craze", rank))
		target := sim.Encounter.TargetUnits[0]

		// Reset leaves every unit at full health (health.go), so there
		// is no headroom for GainHealth to fill until some is spent.
		war.RemoveHealth(sim, war.MaxHealth()*0.5)
		before := war.CurrentHealth()
		// Any spell the target casts at the warrior that crits serves:
		// Rend's own outcome applier is MH-only, so the target's
		// Whirlwind-shaped multi-target special stands in as "a spell
		// hitting the warrior", with its crit chance driven to certain
		// by DrawDebugMultiplier-free CalcAndDealDamage under a fixed
		// seed is unreliable, so this drives the dedicated outcome
		// applier for a guaranteed crit instead.
		dummySpell := target.GetOrRegisterSpell(core.SpellConfig{
			ActionID:    core.ActionID{OtherID: 1},
			SpellSchool: core.SpellSchoolPhysical,
			ProcMask:    core.ProcMaskMeleeMHSpecial,
			DefenseType: core.DefenseTypeMelee,
		})
		result := &core.SpellResult{
			Target: &war.Unit,
			Damage: 1,
		}
		result.Outcome = core.OutcomeCrit
		war.Unit.OnSpellHitTaken(sim, dummySpell, result)

		drainBloodCrazeTicks(sim)
		if after := war.CurrentHealth(); after <= before {
			t.Errorf("CurrentHealth after a crit-taken trigger = %v, want more than %v", after, before)
		}
	})

	t.Run("bloodthirst landed", func(t *testing.T) {
		talents := talentStringWithRank(t, emptyWarriorTalents, "blood_craze", rank)
		talents = talentStringWithRank(t, talents, "bloodthirst", 1)
		war, sim := buildWarriorAndSimForCostTest(t, talents)
		if war.Bloodthirst == nil {
			t.Fatal("a warrior with the bloodthirst talent has no Bloodthirst spell registered")
		}
		war.RemoveHealth(sim, war.MaxHealth()*0.5)
		before := war.CurrentHealth()
		war.Bloodthirst.ApplyEffects(sim, sim.Encounter.TargetUnits[0], war.Bloodthirst.Spell)
		drainBloodCrazeTicks(sim)
		if after := war.CurrentHealth(); after <= before {
			t.Errorf("CurrentHealth after Bloodthirst lands = %v, want more than %v", after, before)
		}
	})

	t.Run("hit over 20 percent max health", func(t *testing.T) {
		war, sim := buildWarriorAndSimForCostTest(t, talentStringWithRank(t, emptyWarriorTalents, "blood_craze", rank))
		target := sim.Encounter.TargetUnits[0]

		war.RemoveHealth(sim, war.MaxHealth()*0.5)
		before := war.CurrentHealth()
		dummySpell := target.GetOrRegisterSpell(core.SpellConfig{
			ActionID:    core.ActionID{OtherID: 2},
			SpellSchool: core.SpellSchoolPhysical,
			ProcMask:    core.ProcMaskMeleeMHSpecial,
			DefenseType: core.DefenseTypeMelee,
		})
		result := &core.SpellResult{
			Target: &war.Unit,
			Damage: war.MaxHealth() * 0.25,
		}
		result.Outcome = core.OutcomeHit
		war.Unit.OnSpellHitTaken(sim, dummySpell, result)

		drainBloodCrazeTicks(sim)
		if after := war.CurrentHealth(); after <= before {
			t.Errorf("CurrentHealth after a >20%%-max-health hit = %v, want more than %v", after, before)
		}
	})
}

// drainBloodCrazeTicks steps the sim forward far enough that a 6s,
// 3-tick Blood Craze heal (blood_craze.go) finishes.
func drainBloodCrazeTicks(sim *core.Simulation) {
	for i := 0; i < 20 && sim.CurrentTime < sim.Duration; i++ {
		if sim.Step() {
			return
		}
	}
}
