package dpswarrior

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

// Booming Voice's cost clause reaches Howls since 1.60.1.70755 (12321
// e1's class mask adds Piercing Howl's family bit): Piercing Howl's 10
// rage is 7.5 at 5/5, 25% off like the shouts.
func TestBoomingVoiceDiscountsPiercingHowl(t *testing.T) {
	plain := buildWarriorForCostTest(t, talentStringWithRank(t, emptyWarriorTalents, "piercing_howl", 1))
	boomed := buildWarriorForCostTest(t, talentStringWithRank(t, talentStringWithRank(t, emptyWarriorTalents, "piercing_howl", 1), "booming_voice", 5))
	if plain.PiercingHowl == nil || boomed.PiercingHowl == nil {
		t.Fatal("Piercing Howl is not registered with its talent")
	}
	if got := plain.PiercingHowl.Cost.GetCurrentCost(); got != 10 {
		t.Fatalf("Piercing Howl costs %v rage with no Booming Voice, want the client's 10", got)
	}
	if got := boomed.PiercingHowl.Cost.GetCurrentCost(); got != 7.5 {
		t.Errorf("Piercing Howl costs %v rage at Booming Voice 5, want 7.5 (25%% off)", got)
	}
}

// Reckless Swings is the talent's learned ability: registered only with
// the point, castable from a rotation, on its 15 s cooldown and a 1 s
// global cooldown.
func TestRecklessSwingsRegistersOnlyWhenTalented(t *testing.T) {
	if war := buildWarriorForCostTest(t, emptyWarriorTalents); war.RecklessSwings != nil || war.RecklessSwingsAura != nil {
		t.Error("a warrior with no point in Reckless Swings has the spell or its aura")
	}
	war := buildWarriorForCostTest(t, talentStringWithRank(t, emptyWarriorTalents, "reckless_swings", 1))
	if war.RecklessSwings == nil {
		t.Fatal("a warrior with Reckless Swings talented has no spell registered")
	}
	if war.RecklessSwings.ActionID.SpellID != 1329555 {
		t.Errorf("Reckless Swings is spell %d, want 1329555", war.RecklessSwings.ActionID.SpellID)
	}
	if !war.RecklessSwings.Flags.Matches(core.SpellFlagAPL) {
		t.Error("Reckless Swings is not castable from a rotation (no SpellFlagAPL)")
	}
	if got := war.RecklessSwings.CD.Duration; got != 15*time.Second {
		t.Errorf("Reckless Swings cooldown = %s, want 15s", got)
	}
	if got := war.RecklessSwings.DefaultCast.GCD; got != time.Second {
		t.Errorf("Reckless Swings global cooldown = %s, want 1s", got)
	}
	if war.RecklessSwings.Cost != nil {
		t.Error("Reckless Swings has a resource cost; it costs health, not rage")
	}
}

// "Your next 4 melee attacks generate 5 additional Rage. Lasts 8 sec."
// Each landed white swing grants 5 rage and spends a charge; a missed
// one does neither, and a special attack is not a melee auto attack.
func TestRecklessSwingsGrantsFiveRageOnFourLandedSwings(t *testing.T) {
	war, sim := buildWarriorAndSimForCostTest(t, talentStringWithRank(t, emptyWarriorTalents, "reckless_swings", 1))
	target := sim.Encounter.TargetUnits[0]
	healthBefore := war.CurrentHealth()

	war.RecklessSwings.Cast(sim, target)
	aura := war.RecklessSwingsAura
	if !aura.IsActive() || aura.GetStacks() != 4 {
		t.Fatalf("after the cast the aura is active=%v with %d charges, want 4", aura.IsActive(), aura.GetStacks())
	}
	if got := aura.Duration; got != 8*time.Second {
		t.Errorf("Reckless Swings lasts %s, want 8s", got)
	}
	if got := war.CurrentHealth(); got != healthBefore {
		t.Errorf("a DPS warrior paid %v health; the cost is taken as healed outside a tank sim", healthBefore-got)
	}

	white := war.AutoAttacks.MHAuto()
	miss := &core.SpellResult{Target: target, Outcome: core.OutcomeMiss}
	aura.OnSpellHitDealt(aura, sim, white, miss)
	if aura.GetStacks() != 4 {
		t.Errorf("a missed swing spent a charge (%d left)", aura.GetStacks())
	}
	special := &core.SpellResult{Target: target, Outcome: core.OutcomeHit}
	aura.OnSpellHitDealt(aura, sim, war.Whirlwind.Spell, special)
	if aura.GetStacks() != 4 {
		t.Errorf("a special attack spent a charge (%d left)", aura.GetStacks())
	}

	rageBefore := war.CurrentRage()
	for i := 0; i < 4; i++ {
		aura.OnSpellHitDealt(aura, sim, white, &core.SpellResult{Target: target, Outcome: core.OutcomeHit})
	}
	if got := war.CurrentRage() - rageBefore; got != 20 {
		t.Errorf("four landed swings granted %v rage, want 20 (5 each)", got)
	}
	if aura.IsActive() {
		t.Error("Reckless Swings is still active after its four charges")
	}
}

// Reckless Swings costs 25% of base health (SpellPower PowerType -2,
// health, PowerCostPct 25). A tanking warrior, the one sim with a health
// model, pays it.
func TestRecklessSwingsCostsATankAQuarterOfBaseHealth(t *testing.T) {
	war, sim := buildWarriorAndSimForCostTest(t, talentStringWithRank(t, emptyWarriorTalents, "reckless_swings", 1))
	target := sim.Encounter.TargetUnits[0]
	target.CurrentTarget = &war.Unit
	before := war.CurrentHealth()
	war.RecklessSwings.Cast(sim, target)
	want := 0.25 * war.GetBaseStats()[stats.Health]
	if got := before - war.CurrentHealth(); got != want {
		t.Errorf("a tanking warrior paid %v health, want %v (25%% of base health)", got, want)
	}
}
