package rogue_test

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

// subtletyPreparationTalents is a zero build with Preparation and the
// Subtlety cooldowns it is meant to reset.
func subtletyPreparationTalents(t *testing.T) string {
	t.Helper()
	talents := zeroRogueTalents
	for _, name := range []string{"preparation", "premeditation", "ghostly_strike", "death_mark", "cold_blood"} {
		talents = talentStringWithRank(t, talents, name, 1)
	}
	return talents
}

// Preparation (14185) "immediately finishes the cooldown on your other Rogue
// abilities" (70755 rank text; its one effect is a server dummy, effect 3).
// The engine reset only Cold Blood, Shadowstep and Vanish, so Premeditation,
// Shank, Ghostly Strike and Death Mark kept their cooldowns.
func TestPreparationFinishesEveryOtherRogueCooldown(t *testing.T) {
	sim, built := buildRogueSimForTalentTest(t, subtletyPreparationTalents(t), "combat_backstab_prebis")
	r := built.GetRogue()
	spells := map[string]*core.Spell{
		"Premeditation": r.Premeditation, "Shank": r.Shank, "Ghostly Strike": r.GhostlyStrike,
		"Death Mark": r.DeathMark, "Cold Blood": r.ColdBlood, "Vanish": r.Vanish,
	}
	for name, spell := range spells {
		if spell == nil {
			t.Fatalf("%s is not registered", name)
		}
		spell.CD.Use(sim)
		if spell.CD.IsReady(sim) {
			t.Fatalf("%s is ready right after its cooldown was started", name)
		}
	}
	r.Preparation.CD.Use(sim)
	r.Preparation.ApplyEffects(sim, sim.Encounter.TargetUnits[0], r.Preparation)
	for name, spell := range spells {
		if !spell.CD.IsReady(sim) {
			t.Errorf("Preparation did not finish %s's cooldown", name)
		}
	}
	if r.Preparation.CD.IsReady(sim) {
		t.Error("Preparation reset its own cooldown")
	}
}

// The automatic use waited for Vanish to be on cooldown, but nothing casts
// Vanish in a fight, so Preparation was never used. It goes once a long DPS
// cooldown it resets (here Premeditation) is cooling down, after Shank and
// Ghostly Strike have been spent, so the reset gives them back too.
func TestPreparationIsUsedOnceAResetIsWorthIt(t *testing.T) {
	sim, built := buildRogueSimForTalentTest(t, subtletyPreparationTalents(t), "combat_backstab_prebis")
	r := built.GetRogue()
	r.StealthAura.Deactivate(sim)
	mcd := r.GetMajorCooldown(r.Preparation.ActionID)
	if mcd == nil {
		t.Fatal("Preparation is not a major cooldown")
	}
	if mcd.ShouldActivate(sim, &r.Character) {
		t.Error("Preparation wants to go with every cooldown it resets ready")
	}
	r.Premeditation.CD.Use(sim)
	if mcd.ShouldActivate(sim, &r.Character) {
		t.Error("Preparation goes while Shank and Ghostly Strike are still ready")
	}
	r.Shank.CD.Use(sim)
	r.GhostlyStrike.CD.Use(sim)
	if !mcd.ShouldActivate(sim, &r.Character) {
		t.Error("Preparation does not go with Premeditation, Shank and Ghostly Strike cooling down")
	}
}
