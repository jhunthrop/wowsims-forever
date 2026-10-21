package dpsrogue

import (
	"math"
	"testing"
	"time"
)

// TestVenomNotRegisteredWithoutTheTalent rebuilds the state
// data/curated/apl/rogue-assassination.json's "inert" entry describes: a
// rogue who hasn't spent the point must not have Venom at all.
func TestVenomNotRegisteredWithoutTheTalent(t *testing.T) {
	_, built := buildRogueForTest(t, rogueTalentStringWith(t))

	if built.Venom != nil {
		t.Fatal("rogue without the Venom talent has Venom registered")
	}
}

// TestVenomDurationScalesWithComboPoints pins the talent tooltip's five
// stated durations (9/12/15/18/21 s for 1-5 combo points) and, per this
// lane's brief, that a 5-point Venom outlasts a 1-point Venom.
func TestVenomDurationScalesWithComboPoints(t *testing.T) {
	sim, built := buildRogueForTest(t, rogueTalentStringWith(t, "venom"))
	if built.Venom == nil {
		t.Fatal("talented level-60 rogue has no Venom registered")
	}
	target := sim.Encounter.TargetUnits[0]

	built.AddComboPoints(sim, 1, target, built.Venom.ComboPointMetrics())
	built.Venom.ApplyEffects(sim, target, built.Venom)
	oneComboPointDuration := built.VenomAura.Duration
	built.VenomAura.Deactivate(sim)

	if got, want := oneComboPointDuration, 9*time.Second; got != want {
		t.Errorf("Venom duration at 1 combo point = %v, want %v", got, want)
	}

	built.AddComboPoints(sim, 5, target, built.Venom.ComboPointMetrics())
	built.Venom.ApplyEffects(sim, target, built.Venom)
	fiveComboPointDuration := built.VenomAura.Duration

	if got, want := fiveComboPointDuration, 21*time.Second; got != want {
		t.Errorf("Venom duration at 5 combo points = %v, want %v", got, want)
	}

	if fiveComboPointDuration <= oneComboPointDuration {
		t.Errorf("a 5-point Venom (%v) must outlast a 1-point Venom (%v)", fiveComboPointDuration, oneComboPointDuration)
	}
}

// TestVenomIncreasesPoisonDamageAndProcChanceWhileActive pins the talent
// tooltip's "increases the damage of your Poisons by 30% and your
// chance to apply Poisons by 10%" against Instant Poison, and that both
// bonuses revert exactly when the aura expires (VenomousTotem's item
// effect toggles additivePoisonBonusChance the same way, in
// sim/rogue/items.go, for the same reason: this must net to zero, not
// leave a residual bonus behind after the buff ends).
func TestVenomIncreasesPoisonDamageAndProcChanceWhileActive(t *testing.T) {
	sim, built := buildRogueForTest(t, rogueTalentStringWith(t, "venom"))
	target := sim.Encounter.TargetUnits[0]

	if built.InstantPoison == nil {
		t.Fatal("level-60 rogue has no Instant Poison registered")
	}

	baseMultiplier := built.InstantPoison.DamageMultiplier
	baseProcChance := built.GetInstantPoisonProcChance()

	built.AddComboPoints(sim, 5, target, built.Venom.ComboPointMetrics())
	built.Venom.ApplyEffects(sim, target, built.Venom)

	if !built.VenomAura.IsActive() {
		t.Fatal("Venom's aura is not active after casting Venom")
	}
	if got, want := built.InstantPoison.DamageMultiplier, baseMultiplier*1.30; math.Abs(got-want) > 1e-9 {
		t.Errorf("Instant Poison DamageMultiplier while Venom is up = %v, want %v (30%% bonus)", got, want)
	}
	if got, want := built.GetInstantPoisonProcChance(), baseProcChance+0.10; math.Abs(got-want) > 1e-9 {
		t.Errorf("Instant Poison proc chance while Venom is up = %v, want %v (+10 points)", got, want)
	}

	built.VenomAura.Deactivate(sim)

	if got := built.InstantPoison.DamageMultiplier; got != baseMultiplier {
		t.Errorf("Instant Poison DamageMultiplier after Venom expires = %v, want the pre-Venom %v", got, baseMultiplier)
	}
	if got := built.GetInstantPoisonProcChance(); got != baseProcChance {
		t.Errorf("Instant Poison proc chance after Venom expires = %v, want the pre-Venom %v", got, baseProcChance)
	}
}
