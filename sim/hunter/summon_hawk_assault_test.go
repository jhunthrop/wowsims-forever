package hunter

import (
	"math"
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// Round 18: Summon Hawk's "continuing its assault for 18 sec". The beta
// combat logs (foreverlogs.gg 2695-2764, 48 encounters, 8 Beast Mastery
// hunters at 25-35) show a real "Hawk" pet unit that appears about 1.07 s
// after the cast, swings in melee every 2.26 s (1.62 s inside Rapid
// Fire), lands about eight swings a summon, crits only with Ferocity, and
// never more than two at once.

func hawkTestTalents(t *testing.T) string {
	t.Helper()
	return buildTalentsStringFromRanks(t, map[string]int{
		"summon_hawk": 1, "ferocity": 5, "unleashed_fury": 5,
	})
}

func stepUntil(sim *core.Simulation, at time.Duration) {
	for sim.CurrentTime < at {
		if sim.Step() {
			return
		}
	}
}

func activeHawks(hunter *Hunter) int {
	n := 0
	for _, hawk := range hunter.hawks {
		if hawk.IsEnabled() {
			n++
		}
	}
	return n
}

func hawkSwings(hunter *Hunter) int32 {
	var swings int32
	for _, hawk := range hunter.hawks {
		for _, m := range hawk.AutoAttacks.MHAuto().SpellMetrics {
			swings += m.Casts
		}
	}
	return swings
}

func TestSummonHawkSendsAHawkThatSwingsForEighteenSeconds(t *testing.T) {
	sim, hunter, target := newRunningHunterWithTalents(t, 60, hawkTestTalents(t), proto.Hunter_Options_PetNone, 25, nil)
	if len(hunter.hawks) == 0 {
		t.Fatal("a Summon Hawk hunter has no hawks to send")
	}
	stepUntil(sim, 0)
	start := sim.CurrentTime
	hunter.SummonHawk.Cast(sim, target)

	stepUntil(sim, start+hawkArrivalDelay+time.Millisecond)
	if n := activeHawks(hunter); n != 1 {
		t.Fatalf("%d hawks out after the dive, want 1", n)
	}
	if got := hunter.activeHawk().summonedAt; got != start+hawkArrivalDelay {
		t.Errorf("hawk arrived at %v, want %v after the cast", got-start, hawkArrivalDelay)
	}
	if got := hunter.HawkAssaultAura.GetStacks(); got != 1 {
		t.Errorf("hawk count aura = %d stacks, want 1", got)
	}
	stepUntil(sim, start+hawkArrivalDelay+hawkAssaultDuration+time.Millisecond)
	if n := activeHawks(hunter); n != 0 {
		t.Fatalf("%d hawks still out 18 s after the dive, want 0", n)
	}
	if hunter.HawkAssaultAura.IsActive() {
		t.Error("hawk count aura still up with no hawk out")
	}
	// Swings at 0, 2.26 ... 15.82 s after arrival: eight, the logs' mode.
	if got := hawkSwings(hunter); got != 8 {
		t.Errorf("hawk swung %d times in one summon, want 8", got)
	}
}

func TestOnlyTwoHawksAtOnce(t *testing.T) {
	sim, hunter, target := newRunningHunterWithTalents(t, 60, hawkTestTalents(t), proto.Hunter_Options_PetNone, 25, nil)
	stepUntil(sim, 0)
	for i := 0; i < 3; i++ {
		at := sim.CurrentTime
		hunter.SummonHawk.CD.Reset()
		hunter.SummonHawk.Cast(sim, target)
		stepUntil(sim, at+hawkArrivalDelay+time.Millisecond)
		if want := min(i+1, hawkMaxActive); activeHawks(hunter) != want {
			t.Fatalf("after summon %d: %d hawks out, want %d", i+1, activeHawks(hunter), want)
		}
		stepUntil(sim, at+3*time.Second)
	}
	if got := hunter.HawkAssaultAura.GetStacks(); got != hawkMaxActive {
		t.Errorf("hawk count aura = %d stacks, want %d", got, hawkMaxActive)
	}
}

func TestRapidFireHastesTheHawks(t *testing.T) {
	sim, hunter, target := newRunningHunterWithTalents(t, 60, hawkTestTalents(t), proto.Hunter_Options_PetNone, 25, nil)
	stepUntil(sim, 0)
	hunter.SummonHawk.Cast(sim, target)
	stepUntil(sim, sim.CurrentTime+hawkArrivalDelay+time.Millisecond)
	hawk := hunter.activeHawk()
	if hawk == nil {
		t.Fatal("no hawk out")
	}
	base := hawk.AutoAttacks.MainhandSwingSpeed()
	if want := time.Duration(hawkSwingSpeed * float64(time.Second)); absDuration(base-want) > time.Millisecond {
		t.Fatalf("hawk swing = %v, want %v", base, want)
	}
	hunter.RapidFireAura.Activate(sim)
	hasted := hawk.AutoAttacks.MainhandSwingSpeed()
	if want := time.Duration(float64(base) / 1.4); absDuration(hasted-want) > time.Millisecond {
		t.Errorf("hawk swing inside Rapid Fire = %v, want %v (the logs: 1.62 s)", hasted, want)
	}
	hunter.RapidFireAura.Deactivate(sim)
	if back := hawk.AutoAttacks.MainhandSwingSpeed(); absDuration(back-base) > time.Millisecond {
		t.Errorf("hawk swing after Rapid Fire = %v, want %v", back, base)
	}
}

func TestHawkTakesFerocityAndUnleashedFuryNotTheHuntersCrit(t *testing.T) {
	sim, hunter, target := newRunningHunterWithTalents(t, 60, hawkTestTalents(t), proto.Hunter_Options_PetNone, 25, nil)
	stepUntil(sim, 0)
	hunter.AddStatDynamic(sim, stats.Crit, 20*core.CritRatingPerCritChance)
	hunter.SummonHawk.Cast(sim, target)
	stepUntil(sim, sim.CurrentTime+hawkArrivalDelay+time.Millisecond)
	hawk := hunter.activeHawk()
	if hawk == nil {
		t.Fatal("no hawk out")
	}
	if got, want := hawk.GetStat(stats.Crit), ferocityCritPerRank*5*core.CritRatingPerCritChance; math.Abs(got-want) > 1e-6 {
		t.Errorf("hawk crit rating = %v, want Ferocity's %v and none of the hunter's", got, want)
	}
	if got, want := hawk.PseudoStats.DamageDealtMultiplier, 1+unleashedFuryPerRank*5; math.Abs(got-want) > 1e-9 {
		t.Errorf("hawk damage multiplier = %v, want Unleashed Fury's %v", got, want)
	}
	mh := hawk.AutoAttacks.MH()
	// Blizzard's 10 November hotfix: "Melee damage dealt by the Hawk pet
	// increased by about 80%." on the beta logs' 0.32 a level.
	mean := 0.32 * 1.8 * 60
	if math.Abs(mh.BaseDamageMin-mean*(1-hawkSwingSpread)) > 1e-9 || math.Abs(mh.BaseDamageMax-mean*(1+hawkSwingSpread)) > 1e-9 {
		t.Errorf("hawk swing %v-%v at 60, want %v +/- %v%%", mh.BaseDamageMin, mh.BaseDamageMax, mean, 100*hawkSwingSpread)
	}
	if ap := hawk.GetStat(stats.AttackPower); ap != 0 {
		t.Errorf("hawk attack power = %v, want 0 (it takes nothing from the hunter)", ap)
	}
}

func TestNoHawksWithoutTheTalent(t *testing.T) {
	_, hunter, _ := newRunningHunterWithTalents(t, 60, buildTalentsStringFromRanks(t, map[string]int{"ferocity": 5}), proto.Hunter_Options_PetNone, 25, nil)
	if len(hunter.hawks) != 0 || hunter.HawkAssaultAura != nil {
		t.Error("a hunter without Summon Hawk has hawks")
	}
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}
