package dpswarrior

import (
	"testing"
	"time"
)

// The client's durations and rank numbers for the abilities the
// conformance report compares: Battle Shout lasts 3 minutes, Sweeping
// Strikes 20 seconds, and Booming Voice takes 5% a point off the shouts'
// rage cost instead of lengthening them.
func TestBattleShoutLastsThreeMinutesAndBoomingVoiceDiscountsIt(t *testing.T) {
	plain := buildWarriorForCostTest(t, emptyWarriorTalents)
	if got := plain.BattleShout.Cost.GetCurrentCost(); got != 10 {
		t.Fatalf("Battle Shout costs %v rage with no talents, want 10", got)
	}

	boomed := buildWarriorForCostTest(t, talentStringWithRank(t, emptyWarriorTalents, "booming_voice", 5))
	if got := boomed.BattleShout.Cost.GetCurrentCost(); got != 7.5 {
		t.Errorf("Battle Shout costs %v rage at Booming Voice 5, want 7.5 (25%% off)", got)
	}
	if got := boomed.DemoralizingShout.Cost.GetCurrentCost(); got != 7.5 {
		t.Errorf("Demoralizing Shout costs %v rage at Booming Voice 5, want 7.5 (25%% off)", got)
	}
	if got := boomed.BattleShout.RelatedAuras[0][0].Duration; got != 3*time.Minute {
		t.Errorf("Battle Shout lasts %v at Booming Voice 5, want 3m0s", got)
	}
}

func TestChargeGrantsTheClientsRageOnlyBeforeTheFight(t *testing.T) {
	war, sim := buildWarriorAndSimForCostTest(t, emptyWarriorTalents)
	if war.Charge == nil {
		t.Fatal("a level 60 warrior has no Charge")
	}
	if war.Charge.RequiredLevel != 46 || war.Charge.Rank != 3 {
		t.Errorf("Charge = level %d rank %d, want level 46 rank 3 (spell 11578)", war.Charge.RequiredLevel, war.Charge.Rank)
	}
	if got := war.Charge.CD.Duration; got != 15*time.Second {
		t.Errorf("Charge cooldown = %v, want 15s", got)
	}

	if !war.Charge.CanCast(sim, war.CurrentTarget) {
		t.Fatal("Charge cannot be cast at the pull")
	}
	before := war.CurrentRage()
	war.Charge.ApplyEffects(sim, war.CurrentTarget, war.Charge.Spell)
	if got := war.CurrentRage() - before; got != 15 {
		t.Errorf("Charge rank 3 granted %v rage, want 15 (150 tenths)", got)
	}
}

func TestRetaliationIsRegisteredFromLevelTwenty(t *testing.T) {
	war := buildWarriorForCostTest(t, emptyWarriorTalents)
	if war.Retaliation == nil {
		t.Fatal("a level 60 warrior has no Retaliation")
	}
	if war.Retaliation.RequiredLevel != 20 || war.Retaliation.CD.Duration != 15*time.Minute {
		t.Errorf("Retaliation = level %d, cooldown %v, want level 20, 15m0s", war.Retaliation.RequiredLevel, war.Retaliation.CD.Duration)
	}
	if got := war.Retaliation.RelatedSelfBuff.Duration; got != 15*time.Second {
		t.Errorf("Retaliation lasts %v, want 15s", got)
	}
	if low := buildWarriorAtLevel(t, 19, emptyWarriorTalents); low.Retaliation != nil {
		t.Error("a level 19 warrior has Retaliation")
	}
}
