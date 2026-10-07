package mage

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// A level-63 raid boss: Frost Nova cannot freeze it (canFreeze), so any
// Frozen state a test sees on it came from Fingers of Frost.
func bossTarget() *proto.Target { return core.DefaultTargetProtoLvl60 }

// A level-60 target Frost Nova can freeze.
func freezableTarget() *proto.Target {
	return &proto.Target{Level: 60, Stats: core.DefaultTargetProtoLvl60.Stats}
}

func fingersTalents(t *testing.T, rank int) string {
	t.Helper()
	return talentStringWithRank(t, ForeverFrostTalents, "fingers_of_frost", rank)
}

func TestFingersOfFrostChargesFollowTheTalentRank(t *testing.T) {
	for rank, wantCharges := range map[int]int32{1: 1, 2: 2} {
		built, sim, _ := newFrostMageSimWithTalents(t, bossTarget(), fingersTalents(t, rank))
		if built.FingersOfFrostAura == nil {
			t.Fatalf("rank %d: no Fingers of Frost aura registered", rank)
		}
		built.grantFingersOfFrost(sim)
		if got := built.FingersOfFrostAura.GetStacks(); got != wantCharges {
			t.Errorf("rank %d: %d charges, want %d", rank, got, wantCharges)
		}
		if got := built.FingersOfFrostAura.RemainingDuration(sim); got != 15*time.Second {
			t.Errorf("rank %d: lasts %v, want 15s", rank, got)
		}
	}
}

func TestFingersOfFrostIsAbsentWithoutTheTalent(t *testing.T) {
	built, _, _ := newFrostMageSimWithTalents(t, bossTarget(), fingersTalents(t, 0))
	if built.FingersOfFrostAura != nil {
		t.Fatal("a mage without the talent has a Fingers of Frost aura")
	}
}

func TestFingersOfFrostChargeIsSpentByTheNextDamagingCastAndNoOther(t *testing.T) {
	built, sim, boss := newFrostMageSimWithTalents(t, bossTarget(), fingersTalents(t, 2))
	built.grantFingersOfFrost(sim)

	lance := built.IceLance[IceLanceRanks]
	lance.ApplyEffects(sim, boss, lance)
	if got := built.FingersOfFrostAura.GetStacks(); got != 1 {
		t.Fatalf("after one Ice Lance %d charges remain, want 1", got)
	}
	lance.ApplyEffects(sim, boss, lance)
	if built.FingersOfFrostAura.IsActive() {
		t.Fatal("the aura outlives its last charge")
	}
}

func TestFingersOfFrostConsumerSet(t *testing.T) {
	built, _, _ := newFrostMageSimWithTalents(t, bossTarget(), fingersTalents(t, 2))
	cases := map[string]struct {
		spell *core.Spell
		want  bool
	}{
		"Frostbolt":   {built.Frostbolt[FrostboltRanks-1], true},
		"Ice Lance":   {built.IceLance[IceLanceRanks], true},
		"Frost Nova":  {built.FrostNova[FrostNovaRanks], true},
		"Blizzard":    {built.Blizzard[len(built.Blizzard)-1], false},
		"Ice Barrier": {built.IceBarrier[len(built.IceBarrier)-1], false},
	}
	for name, c := range cases {
		if c.spell == nil {
			t.Fatalf("%s is not registered", name)
		}
		if got := built.consumesFingersOfFrost(c.spell); got != c.want {
			t.Errorf("%s consumes a charge = %v, want %v", name, got, c.want)
		}
	}
}

// ratioOfTotals sums Ice Lance damage over iterations casts, granting
// Fingers of Frost before each when grant is set.
func iceLanceDamage(t *testing.T, talents string, target *proto.Target, iterations int, grant bool) float64 {
	t.Helper()
	built, sim, unit := newFrostMageSimWithTalents(t, target, talents)
	lance := built.IceLance[IceLanceRanks]
	for i := 0; i < iterations; i++ {
		if grant {
			built.grantFingersOfFrost(sim)
		}
		lance.ApplyEffects(sim, unit, lance)
	}
	waitForOutcomes(sim)
	return lance.SpellMetrics[unit.UnitIndex].TotalDamage
}

func TestFingersOfFrostTriplesIceLanceOnABoss(t *testing.T) {
	const iterations = 200
	talents := fingersTalents(t, 2)
	with := iceLanceDamage(t, talents, bossTarget(), iterations, true)
	without := iceLanceDamage(t, talents, bossTarget(), iterations, false)
	if with < without*1.5 {
		t.Errorf("Ice Lance on a Fingers of Frost charge dealt %v, want at least 1.5x the %v without", with, without)
	}
}

func critRate(t *testing.T, talents string, target *proto.Target, freeze, grant bool) float64 {
	t.Helper()
	const iterations = 600
	built, sim, unit := newFrostMageSimWithTalents(t, target, talents)
	if freeze {
		nova := built.FrostNova[FrostNovaRanks]
		nova.ApplyEffects(sim, unit, nova)
	}
	bolt := built.Frostbolt[FrostboltRanks-1]
	for i := 0; i < iterations; i++ {
		if grant {
			built.grantFingersOfFrost(sim)
		}
		bolt.ApplyEffects(sim, unit, bolt)
	}
	waitForOutcomes(sim)
	m := bolt.SpellMetrics[unit.UnitIndex]
	return float64(m.Crits) / float64(m.Hits)
}

func TestShatterAddsItsCritToFingersOfFrostCasts(t *testing.T) {
	talents := fingersTalents(t, 2)
	frozen := critRate(t, talents, bossTarget(), false, true)
	plain := critRate(t, talents, bossTarget(), false, false)
	// Shatter 3 is +50 crit points; 0.2 leaves room for sampling noise.
	if frozen < plain+0.2 {
		t.Errorf("crit rate on a Fingers of Frost cast %.2f, plain %.2f; want Shatter's bonus", frozen, plain)
	}
}

func TestShatterAddsItsCritAgainstAFrostNovaFrozenTarget(t *testing.T) {
	talents := fingersTalents(t, 2)
	frozen := critRate(t, talents, freezableTarget(), true, false)
	plain := critRate(t, talents, freezableTarget(), false, false)
	if frozen < plain+0.2 {
		t.Errorf("crit rate on a Frost Nova frozen target %.2f, plain %.2f; want Shatter's bonus", frozen, plain)
	}
}

func TestShatterBonusDoesNotLeakPastTheCast(t *testing.T) {
	built, sim, boss := newFrostMageSimWithTalents(t, bossTarget(), fingersTalents(t, 2))
	bolt := built.Frostbolt[FrostboltRanks-1]
	before := bolt.BonusCritRating
	built.grantFingersOfFrost(sim)
	bolt.ApplyEffects(sim, boss, bolt)
	if bolt.BonusCritRating != before {
		t.Errorf("Frostbolt BonusCritRating %v after the cast, was %v before", bolt.BonusCritRating, before)
	}
	if built.isTargetFrozen(boss) {
		t.Error("the boss still reads as Frozen after the consuming cast")
	}
}

// grantedWithin casts spell repeatedly, a second apart, and reports
// whether Fingers of Frost was ever active afterwards.
func grantedWithin(t *testing.T, pick func(*Mage) *core.Spell, casts int) bool {
	t.Helper()
	built, sim, unit := newFrostMageSimWithTalents(t, bossTarget(), fingersTalents(t, 1))
	spell := pick(built)
	for i := 0; i < casts; i++ {
		spell.ApplyEffects(sim, unit, spell)
		waitForOutcomes(sim)
		if built.FingersOfFrostAura.IsActive() {
			return true
		}
	}
	return false
}

func TestFingersOfFrostProcsFromChillSpells(t *testing.T) {
	// 40 landed casts at 15% leave a 0.15% chance of no proc; the seed is
	// fixed so the outcome is not flaky either way.
	for name, pick := range map[string]func(*Mage) *core.Spell{
		"Frostbolt":      func(m *Mage) *core.Spell { return m.Frostbolt[FrostboltRanks-1] },
		"Frostfire Bolt": func(m *Mage) *core.Spell { return m.FrostfireBolt[FrostfireBoltRanks] },
	} {
		if !grantedWithin(t, pick, 40) {
			t.Errorf("40 %s casts never granted Fingers of Frost", name)
		}
	}
}

func TestFrostNovaIsNotAChillEffect(t *testing.T) {
	// The client's Chill class bit is on Frostbolt, Cone of Cold and
	// Frostfire Bolt but not on Frost Nova.
	if grantedWithin(t, func(m *Mage) *core.Spell { return m.FrostNova[FrostNovaRanks] }, 40) {
		t.Error("Frost Nova granted Fingers of Frost")
	}
}

func TestFingersOfFrostProcChanceIsFifteenPercent(t *testing.T) {
	built, sim, _ := newFrostMageSimWithTalents(t, bossTarget(), fingersTalents(t, 1))
	const rolls = 4000
	granted := 0
	for i := 0; i < rolls; i++ {
		built.FingersOfFrostAura.Deactivate(sim)
		built.rollFingersOfFrost(sim)
		if built.FingersOfFrostAura.IsActive() {
			granted++
		}
	}
	rate := float64(granted) / rolls
	if rate < 0.12 || rate > 0.18 {
		t.Errorf("proc rate %.3f, want about 0.15", rate)
	}
}

func TestTheConsumingBoltCanReproc(t *testing.T) {
	// The charge is spent when the cast resolves; the chill lands at
	// impact, afterwards, so a Frostbolt cast on the last charge can
	// grant a fresh one. Over many casts on a single held charge the
	// aura must come back at least once.
	built, sim, boss := newFrostMageSimWithTalents(t, bossTarget(), fingersTalents(t, 1))
	bolt := built.Frostbolt[FrostboltRanks-1]
	reproc := false
	for i := 0; i < 60 && !reproc; i++ {
		built.FingersOfFrostAura.Deactivate(sim)
		built.grantFingersOfFrost(sim)
		bolt.ApplyEffects(sim, boss, bolt)
		if built.FingersOfFrostAura.IsActive() {
			t.Fatal("the charge was not spent by the resolving cast")
		}
		waitForOutcomes(sim)
		reproc = built.FingersOfFrostAura.IsActive()
	}
	if !reproc {
		t.Error("60 consuming Frostbolts never re-procced")
	}
}
