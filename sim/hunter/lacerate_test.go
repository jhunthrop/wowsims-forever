package hunter

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

func TestLacerateRegistersRankByLevel(t *testing.T) {
	cases := []struct {
		level  int32
		wantID int32
	}{
		{29, 0},
		{30, 24118},
		{40, 24119},
		{50, 24120},
		{60, 1299332},
	}
	for _, c := range cases {
		_, built, _ := newBareMeleeHunterAtLevel(t, c.level)
		if c.wantID == 0 {
			if built.Lacerate != nil {
				t.Errorf("level %d hunter has Lacerate; the client teaches it at 30", c.level)
			}
			continue
		}
		if built.Lacerate == nil {
			t.Fatalf("level %d hunter has no Lacerate", c.level)
		}
		if got := built.Lacerate.ActionID.SpellID; got != c.wantID {
			t.Errorf("level %d Lacerate id = %d, want %d", c.level, got, c.wantID)
		}
	}
}

func TestLacerateCostsClientMana(t *testing.T) {
	_, built, _ := newBareMeleeHunterAtLevel(t, 60)
	if got, want := built.Lacerate.Cost.BaseCost, 95.0; got != want {
		t.Errorf("Lacerate mana cost = %v, want %v", got, want)
	}
}

func TestLacerateDeclaresClientBaseDamage(t *testing.T) {
	_, built, _ := newBareMeleeHunterAtLevel(t, 60)
	if got, want := built.Lacerate.ClientBaseDamage, [2]float64{58, 58}; got != want {
		t.Errorf("Lacerate ClientBaseDamage = %v, want %v (per-tick amount, effect 6 aura 3)", got, want)
	}
}

func TestLacerateBleedsSevenTicksOverTwentyOneSeconds(t *testing.T) {
	sim, built, target := newBareMeleeHunterAtLevel(t, 60)

	for i := 0; i < 20 && !built.Lacerate.Dot(target).IsActive(); i++ {
		built.Lacerate.ApplyEffects(sim, target, built.Lacerate)
	}
	dot := built.Lacerate.Dot(target)
	if !dot.IsActive() {
		t.Fatal("Lacerate never landed in 20 attempts")
	}
	if got, want := dot.NumberOfTicks, int32(7); got != want {
		t.Errorf("Lacerate ticks = %d, want %d", got, want)
	}
	if got, want := dot.TickLength, 3*time.Second; got != want {
		t.Errorf("Lacerate tick length = %v, want %v", got, want)
	}
	if got, want := dot.SnapshotBaseDamage, 58.0; got != want {
		t.Errorf("Lacerate per-tick snapshot = %v, want %v", got, want)
	}
}

func TestLacerateIsAPhysicalMeleeBleed(t *testing.T) {
	_, built, _ := newBareMeleeHunterAtLevel(t, 60)
	if !built.Lacerate.SpellSchool.Matches(core.SpellSchoolPhysical) {
		t.Error("Lacerate is not physical; the engine reads physical periodic damage as a bleed")
	}
	if !built.Lacerate.ProcMask.Matches(core.ProcMaskMeleeMHSpecial) {
		t.Error("Lacerate does not carry the melee special proc mask, so melee-ability talents skip it")
	}
}

func TestLacerateNeedsMeleeRange(t *testing.T) {
	sim, built, target := newBareHunterAtLevel(t, 60)
	if built.Lacerate.CanCast(sim, target) {
		t.Error("Lacerate is castable at 25 yd; it needs melee range")
	}
}

// Resourcefulness's cost cut (440529 effect 0, aura 108 misc 14) names its
// spells by class mask: 194 in word 0 (Raptor Strike, Mongoose Bite, Wing
// Clip, the traps), 524288 in word 1 (Counterattack) and 2097152 in word 2
// (Strider Kick). Lacerate (24118, 24119, 24120, 1299332) carries 4096 in
// word 1, which Savage Strikes and Predator's Edge name (528384) and
// Resourcefulness does not, so Lacerate keeps its full mana cost
// (1.60.1.70790 and 1.60.1.70888; sim-accuracy 2026-11-12, melee-hunters H1).
func TestResourcefulnessLeavesLacerateAtItsFullCost(t *testing.T) {
	_, plain, _ := newBareMeleeHunterAtLevel(t, 60)
	_, talented, _ := newRunningHunterForTalentTest(t, 60, "resourcefulness", 2, proto.Hunter_Options_PetNone, 5, nil)
	if got, want := talented.Lacerate.Cost.Multiplier, plain.Lacerate.Cost.Multiplier; got != want {
		t.Errorf("Resourcefulness 2/2 moved Lacerate's cost multiplier to %d, want %d (Lacerate is not in 440529's mask)", got, want)
	}
	if talented.RaptorStrike.Cost.Multiplier >= plain.RaptorStrike.Cost.Multiplier {
		t.Error("Resourcefulness 2/2 did not reduce Raptor Strike's mana cost")
	}
}
