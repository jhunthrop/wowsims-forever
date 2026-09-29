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

// newBareMeleeHunterAtLevel is newBareHunterAtLevel's shape (zero
// talents, zero gear, zero buffs, so a flat "amount" isolates cleanly)
// but in melee range (DistanceFromTarget: 5, matching
// core.MaxMeleeAttackDistance), which the trap spells' own ApplyEffects
// require (hunter.DistanceFromTarget > 5 is a no-op) - unlike
// newBareHunterAtLevel's ranged 25, which the traps' own gate rejects.
func newBareMeleeHunterAtLevel(t *testing.T, level int32) (*core.Simulation, *Hunter, *core.Unit) {
	t.Helper()

	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassHunter,
			Race:               proto.Race_RaceOrc,
			Level:              level,
			Equipment:          &proto.EquipmentSpec{},
			Buffs:              &proto.IndividualBuffs{},
			DistanceFromTarget: 5,
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

// TestImmolationTrapActionIDsMatchSpellconst guards immolation_trap.go
// against regressing to the old 409521/409524/409526/409528/409530
// ids, which do not exist anywhere in spellconst/hunter.json (and so
// are unresolved_id violations against the rotation ladder). The real
// ids, mana cost and required level below are spellconst/hunter.json's
// own spells table entries for Immolation Trap ranks 1-5.
func TestImmolationTrapActionIDsMatchSpellconst(t *testing.T) {
	_, built, _ := newBareHunterAtLevel(t, 60)
	timer := built.NewTimer()

	cases := []struct {
		rank          int
		spellID       int32
		manaCost      float64
		level         int
		oldSpellID    int32
		perTickDamage float64
	}{
		{1, 13795, 50, 16, 409521, 21},
		{2, 14302, 90, 26, 409524, 43},
		{3, 14303, 135, 36, 409526, 68},
		{4, 14304, 190, 46, 409528, 102},
		{5, 14305, 245, 56, 409530, 138},
	}
	for _, c := range cases {
		config := built.getImmolationTrapConfig(c.rank, timer)
		if got := config.ActionID.SpellID; got != c.spellID {
			t.Errorf("Immolation Trap rank %d spell ID = %d, want %d (not the old client-absent %d)", c.rank, got, c.spellID, c.oldSpellID)
		}
		if got := config.ManaCost.FlatCost; got != c.manaCost {
			t.Errorf("Immolation Trap rank %d mana cost = %.0f, want %.0f", c.rank, got, c.manaCost)
		}
		if got := config.RequiredLevel; got != c.level {
			t.Errorf("Immolation Trap rank %d required level = %d, want %d", c.rank, got, c.level)
		}
		if got, want := config.Cast.CD.Duration, 30*time.Second; got != want {
			t.Errorf("Immolation Trap rank %d cooldown = %v, want %v (spellconst category_cooldown_ms 30000)", c.rank, got, want)
		}
		if got, want := config.Dot.NumberOfTicks, int32(5); got != want {
			t.Errorf("Immolation Trap rank %d NumberOfTicks = %d, want %d", c.rank, got, want)
		}
		// spellconst's own child "Immolation Trap Effect" spell
		// (13797/14298/14299/14300/14301) carries period_ms 3000, not
		// the old 1500 - Wowhead's Forever page for it reads "22/139
		// every 3 seconds".
		if got, want := config.Dot.TickLength, 3*time.Second; got != want {
			t.Errorf("Immolation Trap rank %d TickLength = %v, want %v", c.rank, got, want)
		}

		sim, sameHunter, target := newBareHunterAtLevel(t, 60)
		spell := sameHunter.GetSpell(core.ActionID{SpellID: c.spellID})
		if spell == nil {
			t.Fatalf("level-60 hunter has no registered spell for Immolation Trap rank %d (id %d)", c.rank, c.spellID)
		}
		dot := spell.Dot(target)
		dot.Apply(sim)
		if got := dot.SnapshotBaseDamage; got != c.perTickDamage {
			t.Errorf("Immolation Trap rank %d per-tick snapshot damage = %.2f, want %.2f (spellconst per-tick amount)", c.rank, got, c.perTickDamage)
		}
	}
}

// TestExplosiveTrapActionIDsMatchSpellconst guards explosive_trap.go
// against regressing to the old 409532/409534/409535 ids, which do not
// exist anywhere in spellconst/hunter.json. The real ids, mana cost and
// required level below are spellconst/hunter.json's own spells table
// entries for Explosive Trap ranks 1-3.
func TestExplosiveTrapActionIDsMatchSpellconst(t *testing.T) {
	_, built, _ := newBareHunterAtLevel(t, 60)
	timer := built.NewTimer()

	cases := []struct {
		rank          int
		spellID       int32
		manaCost      float64
		level         int
		oldSpellID    int32
		perTickDamage float64
	}{
		{1, 13813, 275, 34, 409532, 15},
		{2, 14316, 395, 44, 409534, 24},
		{3, 14317, 520, 54, 409535, 33},
	}
	for _, c := range cases {
		config := built.getExplosiveTrapConfig(c.rank, timer)
		if got := config.ActionID.SpellID; got != c.spellID {
			t.Errorf("Explosive Trap rank %d spell ID = %d, want %d (not the old client-absent %d)", c.rank, got, c.spellID, c.oldSpellID)
		}
		if got := config.ManaCost.FlatCost; got != c.manaCost {
			t.Errorf("Explosive Trap rank %d mana cost = %.0f, want %.0f", c.rank, got, c.manaCost)
		}
		if got := config.RequiredLevel; got != c.level {
			t.Errorf("Explosive Trap rank %d required level = %d, want %d", c.rank, got, c.level)
		}
		if got, want := config.Cast.CD.Duration, 30*time.Second; got != want {
			t.Errorf("Explosive Trap rank %d cooldown = %v, want %v (spellconst category_cooldown_ms 30000)", c.rank, got, want)
		}
		if got, want := config.Dot.NumberOfTicks, int32(10); got != want {
			t.Errorf("Explosive Trap rank %d NumberOfTicks = %d, want %d", c.rank, got, want)
		}
		if got, want := config.Dot.TickLength, 2*time.Second; got != want {
			t.Errorf("Explosive Trap rank %d TickLength = %v, want %v", c.rank, got, want)
		}

		sim, sameHunter, _ := newBareHunterAtLevel(t, 60)
		spell := sameHunter.GetSpell(core.ActionID{SpellID: c.spellID})
		if spell == nil {
			t.Fatalf("level-60 hunter has no registered spell for Explosive Trap rank %d (id %d)", c.rank, c.spellID)
		}
		// Explosive Trap's Dot is IsAOE (its Aura lives on the caster,
		// not per-target), so it is reached via AOEDot(), not
		// Dot(target) - unlike Immolation Trap above.
		dot := spell.AOEDot()
		dot.Apply(sim)
		if got := dot.SnapshotBaseDamage; got != c.perTickDamage {
			t.Errorf("Explosive Trap rank %d per-tick snapshot damage = %.2f, want %.2f (spellconst per-tick amount)", c.rank, got, c.perTickDamage)
		}
	}
}

// TestExplosiveTrapInstantDamageMatchesSpellconst guards
// explosive_trap.go's instant hit against regressing to the old
// min/max roll (104-135/145-193/208-265), which never landed on
// spellconst/hunter.json's own "Explosive Trap Effect" flat amount
// (115/163/229, sp/ap coefficient both 0 - a single non-random value,
// corroborated on Wowhead's Forever pages: "School Damage ... Value:
// 116" / "230"). Averaged over many casts (in melee range, a bare
// single-target encounter so numHits is always 1) to smooth out the
// hit/crit table's own variance around that flat value, the same shape
// TestArcaneShotRank8DamageMatchesSpellconst uses.
func TestExplosiveTrapInstantDamageMatchesSpellconst(t *testing.T) {
	cases := []struct {
		rank     int
		spellID  int32
		wantFlat float64
	}{
		{1, 13813, 115},
		{2, 14316, 163},
		{3, 14317, 229},
	}
	for _, c := range cases {
		total := 0.0
		landed := 0
		const attempts = 40
		for i := 0; i < attempts; i++ {
			sim, built, target := newBareMeleeHunterAtLevel(t, 60)
			spell := built.GetSpell(core.ActionID{SpellID: c.spellID})
			if spell == nil {
				t.Fatalf("level-60 hunter has no registered spell for Explosive Trap rank %d (id %d)", c.rank, c.spellID)
			}
			spell.ApplyEffects(sim, target, spell)
			for step := 0; step < 50; step++ {
				if spell.SpellMetrics[target.UnitIndex].TotalDamage > 0 {
					break
				}
				if done := sim.Step(); done {
					break
				}
			}
			if got := spell.SpellMetrics[target.UnitIndex].TotalDamage; got > 0 {
				total += got
				landed++
			}
		}
		if landed == 0 {
			t.Fatalf("Explosive Trap rank %d never landed in %d attempts", c.rank, attempts)
		}
		avg := total / float64(landed)
		if avg < c.wantFlat*0.85 || avg > c.wantFlat*1.6 {
			// Upper bound is wide: OutcomeMagicHitAndCrit can crit at
			// up to 2x on a landed hit, and this loop stops at the
			// FIRST damage tick recorded, which on rare seeds can be
			// the AoE dot's own first (2s-later) tick rather than the
			// instant hit if the instant hit itself never landed - a
			// true failure (id resolves to nothing near wantFlat) is
			// still well outside this band on the low side.
			t.Errorf("Explosive Trap rank %d average landed damage = %.1f, want close to the client's flat %.1f", c.rank, avg, c.wantFlat)
		}
	}
}

// TestFreezingTrapActionIDsMatchSpellconst guards freezing_trap.go
// against regressing to the old fake id 409510 (absent from
// spellconst/hunter.json, so an unresolved_id violation against the
// rotation ladder) and its 15s cooldown. The real ids, mana cost and
// required level below are spellconst/hunter.json's own spells table
// entries for Freezing Trap ranks 1-3 (1499/14310/14311); the
// cooldown is the shared "Traps" category cooldown of 30s, matching
// Immolation/Explosive Trap's own fix, not the stale duplicate rank-3
// entry 27753 (different family_mask, 15000ms category_cooldown_ms).
func TestFreezingTrapActionIDsMatchSpellconst(t *testing.T) {
	_, built, _ := newBareHunterAtLevel(t, 60)
	timer := built.NewTimer()

	const oldSpellID = int32(409510)
	cases := []struct {
		rank     int
		spellID  int32
		manaCost float64
		level    int
	}{
		{1, 1499, 50, 20},
		{2, 14310, 75, 40},
		{3, 14311, 100, 60},
	}
	for _, c := range cases {
		config := built.getFreezingTrapConfig(c.rank, timer)
		if got := config.ActionID.SpellID; got != c.spellID {
			t.Errorf("Freezing Trap rank %d spell ID = %d, want %d (not the old client-absent %d)", c.rank, got, c.spellID, oldSpellID)
		}
		if got := config.ManaCost.FlatCost; got != c.manaCost {
			t.Errorf("Freezing Trap rank %d mana cost = %.0f, want %.0f", c.rank, got, c.manaCost)
		}
		if got := config.RequiredLevel; got != c.level {
			t.Errorf("Freezing Trap rank %d required level = %d, want %d", c.rank, got, c.level)
		}
		if got, want := config.Cast.CD.Duration, 30*time.Second; got != want {
			t.Errorf("Freezing Trap rank %d cooldown = %v, want %v (spellconst category_cooldown_ms 30000, shared Traps timer)", c.rank, got, want)
		}
		if config.Cast.CD.Timer != timer {
			t.Errorf("Freezing Trap rank %d does not share the Traps timer with Immolation/Explosive Trap", c.rank)
		}
	}
}
