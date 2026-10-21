package hunter

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// The client rows (build 1.60.1.70009): every Mongoose Bite rank says
// "Can only be performed after you dodge"; Defensive State (5302) lasts
// 5 sec with one charge a melee spell consumes; Expose Prey (1310532)
// has a 5/10% chance on a 340 proc mask (melee and ranged autos and
// specials) and grants the same window (1310726, "Mongoose Bite
// activated").

// landedHit replays one landed attack with the given proc mask through
// the hunter's own damage-dealt handlers.
func landedHit(sim *core.Simulation, hunter *Hunter, target *core.Unit, mask core.ProcMask) {
	hunter.OnSpellHitDealt(sim, &core.Spell{ProcMask: mask}, &core.SpellResult{Outcome: core.OutcomeHit, Target: target})
}

func markedExposePreyHunter(t *testing.T, rank int32, marked bool) (*core.Simulation, *Hunter, *core.Unit) {
	t.Helper()
	debuffs := &proto.Debuffs{}
	if marked {
		debuffs.HuntersMark = proto.TristateEffect_TristateEffectRegular
	}
	return newRunningHunterForTalentTest(t, 60, "expose_prey", rank, proto.Hunter_Options_PetNone, 5, debuffs)
}

// procsWithin reports whether 300 replayed hits ever opened the window.
func procsWithin(sim *core.Simulation, hunter *Hunter, target *core.Unit, mask core.ProcMask) bool {
	for i := 0; i < 300; i++ {
		landedHit(sim, hunter, target, mask)
		if hunter.MongooseBiteWindowAura.IsActive() {
			return true
		}
	}
	return false
}

func TestMongooseBiteNeedsTheWindow(t *testing.T) {
	sim, hunter, target := buildSurvivalMeleeHunter(t)

	if hunter.MongooseBite.CanCast(sim, target) {
		t.Fatal("Mongoose Bite is castable with no dodge and no Expose Prey window")
	}
	hunter.MongooseBiteWindowAura.Activate(sim)
	if !hunter.MongooseBite.CanCast(sim, target) {
		t.Fatal("Mongoose Bite is not castable inside its window")
	}
}

func TestMongooseBiteWindowIsFiveSecondsAndSpendsOnCast(t *testing.T) {
	sim, hunter, target := buildSurvivalMeleeHunter(t)

	if got, want := hunter.MongooseBiteWindowAura.Duration, 5*time.Second; got != want {
		t.Errorf("window duration = %v, want %v (spell 5302)", got, want)
	}
	if got, want := hunter.MongooseBiteWindowAura.ActionID.SpellID, int32(5302); got != want {
		t.Errorf("window aura id = %d, want %d", got, want)
	}
	hunter.MongooseBiteWindowAura.Activate(sim)
	hunter.MongooseBite.ApplyEffects(sim, target, hunter.MongooseBite)
	if hunter.MongooseBiteWindowAura.IsActive() {
		t.Error("the window survived a Mongoose Bite; its single charge must be spent")
	}
}

func TestDodgeOpensTheMongooseBiteWindow(t *testing.T) {
	sim, hunter, target := buildSurvivalMeleeHunter(t)

	hunter.OnSpellHitTaken(sim, &core.Spell{}, &core.SpellResult{Outcome: core.OutcomeParry, Target: &hunter.Unit})
	if hunter.MongooseBiteWindowAura.IsActive() {
		t.Fatal("a parry opened the Mongoose Bite window; only a dodge does")
	}
	hunter.OnSpellHitTaken(sim, &core.Spell{}, &core.SpellResult{Outcome: core.OutcomeDodge, Target: &hunter.Unit})
	if !hunter.MongooseBiteWindowAura.IsActive() {
		t.Fatal("a dodge did not open the Mongoose Bite window")
	}
	if !hunter.MongooseBite.CanCast(sim, target) {
		t.Error("Mongoose Bite is not castable after a dodge")
	}
}

func TestExposePreyOpensTheWindowOnMarkedTargetsForMeleeAndRanged(t *testing.T) {
	masks := map[string]core.ProcMask{
		"melee auto":     core.ProcMaskMeleeMHAuto,
		"melee special":  core.ProcMaskMeleeMHSpecial,
		"ranged auto":    core.ProcMaskRangedAuto,
		"ranged special": core.ProcMaskRangedSpecial,
	}
	for name, mask := range masks {
		sim, hunter, target := markedExposePreyHunter(t, 2, true)
		if !procsWithin(sim, hunter, target, mask) {
			t.Errorf("Expose Prey never opened the window on a %s hit against a marked target", name)
		}
	}
}

func TestExposePreyIgnoresSpellDamageAndUnmarkedTargets(t *testing.T) {
	sim, hunter, target := markedExposePreyHunter(t, 2, true)
	if procsWithin(sim, hunter, target, core.ProcMaskSpellDamage) {
		t.Error("Expose Prey proc'd off a spell-damage hit; the client mask is melee and ranged attacks")
	}

	sim, hunter, target = markedExposePreyHunter(t, 2, false)
	if procsWithin(sim, hunter, target, core.ProcMaskMeleeMHAuto) {
		t.Error("Expose Prey proc'd against a target with no Hunter's Mark")
	}
}

func TestExposePreyDoesNotResetTheCooldown(t *testing.T) {
	sim, hunter, target := markedExposePreyHunter(t, 2, true)
	hunter.MongooseBite.CD.Use(sim)
	for i := 0; i < 300; i++ {
		landedHit(sim, hunter, target, core.ProcMaskMeleeMHAuto)
	}
	if hunter.MongooseBite.CD.IsReady(sim) {
		t.Error("Expose Prey reset Mongoose Bite's cooldown; it only opens the window")
	}
}

// Expose Prey holds the window for 10 sec (5 sec before build 1.60.1.70291);
// a dodge's 5 sec window never cuts it short.
func TestExposePreyHoldsTheWindowForTenSeconds(t *testing.T) {
	sim, hunter, target := markedExposePreyHunter(t, 2, true)
	if !procsWithin(sim, hunter, target, core.ProcMaskMeleeMHAuto) {
		t.Fatal("Expose Prey never opened the window")
	}
	if got := hunter.MongooseBiteWindowAura.RemainingDuration(sim); got < 9*time.Second || got > exposePreyWindowLength {
		t.Errorf("window has %v left right after the proc, want about %v", got, exposePreyWindowLength)
	}
	hunter.openMongooseBiteWindowFor(sim, mongooseBiteWindowLength)
	if got := hunter.MongooseBiteWindowAura.RemainingDuration(sim); got < 9*time.Second {
		t.Errorf("a dodge's window cut the Expose Prey window to %v", got)
	}
}
