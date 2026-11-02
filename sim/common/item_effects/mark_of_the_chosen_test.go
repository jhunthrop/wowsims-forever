package item_effects_test

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// Mark of the Chosen (17774): 21969 procs 21970 on 2% of the melee hits taken
// (SpellAuraOptions 132914: ProcChance 2, ProcCategoryRecovery 120000), and
// 21970 is aura 29 (all stats), base 21, for 1 minute. Wowhead Forever:
// "increasing all stats by 21 for 1 min (Proc chance: 2%, 2m cooldown)". The
// engine gave Era's 25 with no cooldown, so a proc could land again and
// refresh the buff.
func TestMarkOfTheChosenIsTwentyOneStatsOnATwoMinuteCooldown(t *testing.T) {
	sim, character := buildCharacterWithWeapon(t, proto.Class_ClassWarrior, &proto.Player_Warrior{Warrior: &proto.Warrior{Options: &proto.Warrior_Options{}}}, proto.ItemSlot_ItemSlotTrinket1, 17774)

	effect := character.GetAuraByID(core.ActionID{SpellID: 21970})
	if effect == nil {
		t.Fatal("no Mark of the Chosen effect aura")
	}
	// The character's stamina multipliers (Kings and the like) scale any
	// flat stamina: measure them on a known 100 first.
	before := character.GetStat(stats.Stamina)
	character.AddStatDynamic(sim, stats.Stamina, 100)
	perPoint := (character.GetStat(stats.Stamina) - before) / 100
	character.AddStatDynamic(sim, stats.Stamina, -100)
	effect.Activate(sim)
	if got := (character.GetStat(stats.Stamina) - before) / perPoint; math.Abs(got-21) > 1e-6 {
		t.Errorf("Mark of the Chosen raises stamina by %v, want 21", got)
	}
	effect.Deactivate(sim)

	trigger := character.GetAura("Mark of the Chosen")
	if trigger == nil {
		t.Fatal("no Mark of the Chosen trigger aura")
	}
	attacker := sim.Encounter.TargetUnits[0]
	swing := attacker.GetOrRegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{OtherID: proto.OtherAction_OtherActionAttack},
		SpellSchool: core.SpellSchoolPhysical,
		ProcMask:    core.ProcMaskMeleeMHAuto,
	})
	result := &core.SpellResult{Target: &character.Unit, Outcome: core.OutcomeHit}
	gains := 0
	for i := 0; i < 2000; i++ {
		trigger.OnSpellHitTaken(trigger, sim, swing, result)
		if effect.IsActive() {
			gains++
			effect.Deactivate(sim)
		}
	}
	if gains != 1 {
		t.Errorf("2000 hits taken at one moment gave %d procs, want 1 (2 minute proc cooldown)", gains)
	}
}
