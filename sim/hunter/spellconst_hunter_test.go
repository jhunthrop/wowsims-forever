package hunter

import (
	"testing"
	"time"

	_ "github.com/wowsims/classic/sim/common" // imported to get item effects included.
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// newBareHunterAtLevel builds a zero-talent, zero-gear, zero-buff hunter
// at the given level against the standard level-63 dummy, the same
// shape the rotation ladder uses for its bare characters. Buffs are
// empty (not core.FullBuffs, which spellconst_damage_test.go's warlock
// equivalent uses) so a spell's own BonusCoefficient*SpellPower term
// stays at zero and a damage assertion isolates the registered flat
// "amount" cleanly, with no buff-derived spell power to widen the
// tolerance band for.
func newBareHunterAtLevel(t *testing.T, level int32) (*core.Simulation, *Hunter, *core.Unit) {
	t.Helper()

	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassHunter,
			Race:               proto.Race_RaceOrc,
			Level:              level,
			Equipment:          &proto.EquipmentSpec{},
			Buffs:              &proto.IndividualBuffs{},
			DistanceFromTarget: 25, // outside core.MinRangedAttackDistance.
		},
		P1PlayerOptions,
	)
	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})

	sim := core.NewSim(&proto.RaidSimRequest{
		Raid: raid,
		Encounter: &proto.Encounter{
			Duration: 60,
			Targets:  []*proto.Target{core.DefaultTargetProtoLvl60},
		},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()

	agent, ok := sim.Raid.Parties[0].Players[0].(HunterAgent)
	if !ok {
		t.Fatal("the raid's first player is not a hunter agent")
	}
	built := agent.GetHunter()
	target := sim.Encounter.TargetUnits[0]
	return sim, built, target
}

// averageArcaneShotRank8Damage casts Arcane Shot's max rank n times (a
// fresh bare level-60 hunter each time, so cooldown never gates a later
// cast) and returns the mean landed TotalDamage, the same shape as
// spellconst_damage_test.go's averageDamagePerCast.
func averageArcaneShotRank8Damage(t *testing.T, n int) float64 {
	t.Helper()
	total := 0.0
	for i := 0; i < n; i++ {
		sim, built, target := newBareHunterAtLevel(t, 60)
		if built.ArcaneShot == nil {
			t.Fatal("level-60 hunter has no Arcane Shot registered")
		}
		spell := built.ArcaneShot
		spell.ApplyEffects(sim, target, spell)
		for step := 0; step < 50; step++ {
			if spell.SpellMetrics[target.UnitIndex].TotalDamage > 0 {
				break
			}
			if done := sim.Step(); done {
				break
			}
		}
		total += spell.SpellMetrics[target.UnitIndex].TotalDamage
	}
	return total / float64(n)
}

// TestArcaneShotRank8DamageMatchesSpellconst guards arcane_shot.go's
// baseDamage array against regressing back to the pre-Forever classic
// numbers (13/21/33/59/83/115/145/183) instead of spellconst/
// hunter.json's own flat rank-8 amount (14287: 217, at zero spell power
// so BonusCoefficient contributes nothing here).
func TestArcaneShotRank8DamageMatchesSpellconst(t *testing.T) {
	const wantBase = 217.0
	const oldEngineValue = 183.0

	got := averageArcaneShotRank8Damage(t, 30)

	if got < wantBase*0.85 || got > wantBase*1.25 {
		t.Errorf("Arcane Shot rank 8 average damage = %.1f, want close to the client's flat %.1f", got, wantBase)
	}
	if got < oldEngineValue*1.05 {
		t.Errorf("Arcane Shot rank 8 average damage = %.1f, still in range of the old engine value %.1f - the fix did not take", got, oldEngineValue)
	}
}

// TestArcaneShotCooldownUsesForeversImprovedArcaneShotRate guards the
// cooldown formula in getArcaneShotConfig against regressing to
// Classic's 200ms/rank (max 2 ranks): Forever's Improved Arcane Shot
// (talents/hunter.json node 105006) is max_rank 5, and each rank's own
// tooltip reduces the cooldown by 0.3s ("Reduces the cooldown of your
// Arcane Shot by 0.3/0.6/.../1.5 sec"). getArcaneShotConfig only builds
// a SpellConfig struct (no aura registration), so calling it again on an
// already-finalized environment is safe, unlike the aura-registering
// config builders exercised via already-registered spells below.
func TestArcaneShotCooldownUsesForeversImprovedArcaneShotRate(t *testing.T) {
	_, built, _ := newBareHunterAtLevel(t, 60)
	timer := built.NewTimer()

	cases := []struct {
		talentPoints int32
		want         time.Duration
	}{
		{0, 6 * time.Second},
		{2, 6*time.Second - 600*time.Millisecond}, // old formula's own cap would have stopped here at 5.6s.
		{5, 6*time.Second - 1500*time.Millisecond},
	}
	for _, c := range cases {
		built.Talents.ImprovedArcaneShot = c.talentPoints
		config := built.getArcaneShotConfig(8, timer)
		if got := config.Cast.CD.Duration; got != c.want {
			t.Errorf("Improved Arcane Shot %d/5: Arcane Shot cooldown = %v, want %v", c.talentPoints, got, c.want)
		}
	}
}

// TestMultiShotIsForeversSingleFlatSpell guards multi_shot.go against
// regressing to Classic's five-rank progression (2643/14288/14289/
// 14290/25294): spellconst/hunter.json carries only spell 2643 for
// Multi-Shot, at every level, with category_cooldown_ms 6000 (6s, not
// Classic's 10s) and a mana cost of 13.9% of base mana (Wowhead's
// Forever tooltip for 2643; spellconst's own flat "cost" column is 0 for
// this spell, which the package's own docs call "the table does not
// know"). A level-30 hunter (between Classic's old rank-2 and rank-3
// breakpoints) must still resolve to 2643, not a legacy id the pinned
// client's rotation ladder cannot resolve at all.
func TestMultiShotIsForeversSingleFlatSpell(t *testing.T) {
	for _, level := range []int32{18, 30, 60} {
		_, built, _ := newBareHunterAtLevel(t, level)

		if built.MultiShot == nil {
			t.Fatalf("level-%d hunter has no Multi-Shot registered", level)
		}
		if got, want := built.MultiShot.ActionID.SpellID, int32(2643); got != want {
			t.Errorf("level-%d Multi-Shot spell ID = %d, want %d", level, got, want)
		}
		if got, want := built.MultiShot.CD.Duration, 6*time.Second; got != want {
			t.Errorf("level-%d Multi-Shot cooldown = %v, want %v", level, got, want)
		}
		wantCost := multiShotBaseManaCostPercent * built.BaseMana
		if got := built.MultiShot.Cost.BaseCost; got != wantCost {
			t.Errorf("level-%d Multi-Shot mana cost = %.2f, want %.2f (%.1f%% of %.2f base mana)", level, got, wantCost, multiShotBaseManaCostPercent*100, built.BaseMana)
		}
	}
}

// TestAspectOfTheHawkChargesManaMatchingSpellconst guards the mana cost
// this fix added to getAspectOfTheHawkSpellConfig -- the aura used to be
// entirely free to activate, at every rank. Costs are spellconst/
// hunter.json's own flat per-rank amount. Each case builds a fresh
// hunter at that rank's own learn level so the already-registered
// hunter.AspectOfTheHawk-backing spell (fetched via GetSpell) is
// inspected rather than calling the aura-registering config builder a
// second time on a finalized environment.
func TestAspectOfTheHawkChargesManaMatchingSpellconst(t *testing.T) {
	cases := []struct {
		level    int32
		spellID  int32
		wantCost float64
	}{
		{10, 13165, 20},
		{58, 14322, 110},
		{60, 25296, 120},
	}
	for _, c := range cases {
		_, built, _ := newBareHunterAtLevel(t, c.level)
		spell := built.GetSpell(core.ActionID{SpellID: c.spellID})
		if spell == nil {
			t.Fatalf("level-%d hunter has no spell %d registered", c.level, c.spellID)
		}
		if got, want := spell.Cost.BaseCost, c.wantCost; got != want {
			t.Errorf("Aspect of the Hawk (spell %d, level %d) mana cost = %.0f, want %.0f", c.spellID, c.level, got, want)
		}
	}
}

// TestAspectOfTheHawkRank6AttackPowerMatchesSpellconst guards rank 6
// (14322, learned at 58) against reverting to Classic's 110 ranged
// attack power: spellconst's own effect amount for 14322 is 55,
// corroborated by Wowhead's Forever tooltip for that spell ("Mod Ranged
// Attack Power Value: 56"). getMaxAspectOfTheHawkAttackPower is a pure
// lookup (no aura registration), so it's safe to call directly.
func TestAspectOfTheHawkRank6AttackPowerMatchesSpellconst(t *testing.T) {
	_, built, _ := newBareHunterAtLevel(t, 60)
	if got, want := built.getMaxAspectOfTheHawkAttackPower(6), 55.0; got != want {
		t.Errorf("Aspect of the Hawk rank 6 attack power = %.0f, want %.0f", got, want)
	}
}

// TestVolleyHasNoCooldownMatchingSpellconst guards volley.go against
// reintroducing a cooldown: spellconst/hunter.json's cooldown_ms and
// category_cooldown_ms are both 0 for all three ranks (1510/14294/
// 14295), and Wowhead's Forever tooltip for 1510 states "n/a" for
// cooldown. Volley is gated only by its mana cost and its own 6s
// channel, unchanged by this fix. Built at level 58 so hunter.Volley
// resolves to rank 3 (14295) via the already-registered spell, rather
// than calling the Dot-aura-registering config builder again on a
// finalized environment.
func TestVolleyHasNoCooldownMatchingSpellconst(t *testing.T) {
	_, built, _ := newBareHunterAtLevel(t, 58)
	if built.Volley == nil {
		t.Fatal("level-58 hunter has no Volley registered")
	}
	if got, want := built.Volley.ActionID.SpellID, int32(14295); got != want {
		t.Fatalf("Volley spell ID = %d, want rank 3 (%d)", got, want)
	}
	if built.Volley.CD.Timer != nil {
		t.Errorf("Volley rank 3 has a cooldown timer, want none")
	}
}

// TestSerpentStingPerTickDamageMatchesSpellconst checks the exact
// per-tick snapshot value (no crit/hit-table noise to average out for a
// DoT) at three ranks against spellconst/hunter.json's own per-tick
// effect amount, guarding against regressing to the old array's
// total-then-divide-by-5 numbers, which undershot every rank but the
// last (rank 2's old per-tick value was 40/5=8 against the client's 6;
// rank 9's 555/5=111 happened to match the client's 111 by coincidence).
// Each case builds a hunter at that rank's own learn level so
// hunter.SerpentSting is already registered at exactly that rank.
func TestSerpentStingPerTickDamageMatchesSpellconst(t *testing.T) {
	cases := []struct {
		level   int32
		spellID int32
		want    float64
	}{
		{10, 13549, 6},   // rank 2
		{34, 13552, 34},  // rank 5
		{60, 25295, 111}, // rank 9
	}
	for _, c := range cases {
		found := false
		for attempt := 0; attempt < 20 && !found; attempt++ {
			sim, built, target := newBareHunterAtLevel(t, c.level)
			spell := built.SerpentSting
			if spell == nil {
				t.Fatalf("level-%d hunter has no Serpent Sting registered", c.level)
			}
			if got := spell.ActionID.SpellID; got != c.spellID {
				t.Fatalf("level-%d Serpent Sting spell ID = %d, want %d", c.level, got, c.spellID)
			}

			result := spell.CalcOutcome(sim, target, spell.OutcomeRangedHitNoHitCounter)
			if !result.Landed() {
				continue
			}
			spell.Dot(target).Apply(sim)

			dot := spell.Dot(target)
			if !dot.IsActive() {
				continue
			}
			found = true
			if got := dot.SnapshotBaseDamage; got != c.want {
				t.Errorf("Serpent Sting (spell %d) per-tick snapshot damage = %.2f, want %.2f (spellconst amount)", c.spellID, got, c.want)
			}
		}
		if !found {
			t.Fatalf("Serpent Sting (spell %d) never landed in 20 attempts", c.spellID)
		}
	}
}
