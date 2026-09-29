package elemental

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/shaman"
)

// LavaBurstOnlyTalentsString sets only the Elemental capstone Lava Burst
// (the last field of the Elemental tree's 16 tracked nodes), the same
// "isolate the one talent this lane added" convention as warlock's
// WrackOnlyTalentsString / IncinerateOnlyTalentsString.
const LavaBurstOnlyTalentsString = "0000000000000001--"

func newLavaBurstShaman(t *testing.T, level int32) (*core.Simulation, *shaman.Shaman) {
	t.Helper()

	player := core.WithSpec(
		&proto.Player{
			Class:         proto.Class_ClassShaman,
			Race:          proto.Race_RaceTroll,
			Level:         level,
			Equipment:     &proto.EquipmentSpec{},
			Buffs:         core.FullBuffs.Player,
			TalentsString: LavaBurstOnlyTalentsString,
		},
		&proto.Player_ElementalShaman{
			ElementalShaman: &proto.ElementalShaman{
				Options: &proto.ElementalShaman_Options{},
			},
		},
	)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)

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

	agent, ok := sim.Raid.Parties[0].Players[0].(shaman.ShamanAgent)
	if !ok {
		t.Fatal("the raid's first player is not a shaman agent")
	}
	return sim, agent.GetShaman()
}

// TestLavaBurstLevel60HasMaxRankAndDealsDamage: registerLavaBurstSpell is
// gated on shaman.Talents.LavaBurst, and per spellranks.json a level-60
// shaman's castable rank is 1238300 (rank 3, spellconst amount 220,
// sp_coefficient 0.714).
func TestLavaBurstLevel60HasMaxRankAndDealsDamage(t *testing.T) {
	sim, built := newLavaBurstShaman(t, 60)

	if len(built.LavaBurst) == 0 {
		t.Fatal("level-60 shaman with the Lava Burst talent has no Lava Burst spell registered")
	}
	maxRank := built.LavaBurst[len(built.LavaBurst)-1]
	if got, want := maxRank.ActionID.SpellID, int32(1238300); got != want {
		t.Errorf("Lava Burst max rank spell ID = %d, want %d", got, want)
	}
	if got, want := maxRank.Rank, 3; got != want {
		t.Errorf("Lava Burst max rank = %d, want %d", got, want)
	}
	if got, want := maxRank.CD.Duration, shaman.LavaBurstCooldown; got != want {
		t.Errorf("Lava Burst cooldown = %v, want %v (client category_cooldown_ms 10000)", got, want)
	}

	target := sim.Encounter.TargetUnits[0]
	maxRank.ApplyEffects(sim, target, maxRank)

	metrics := maxRank.SpellMetrics[target.UnitIndex]
	if metrics.Hits+metrics.Crits == 0 {
		t.Fatalf("Lava Burst outcome was neither a hit nor a crit (misses=%d)", metrics.Misses)
	}
	if metrics.TotalDamage <= 0 {
		t.Errorf("Lava Burst dealt %v damage, want > 0", metrics.TotalDamage)
	}
}

// TestLavaBurstRanksMatchClientLevelsAndSpellIDs pins the three castable
// ranks (spellconst/shaman.json + spellranks.json) so a future data
// refresh notices if Forever's own ranks move.
func TestLavaBurstRanksMatchClientLevelsAndSpellIDs(t *testing.T) {
	cases := []struct {
		level     int32
		wantRank  int
		wantSpell int32
	}{
		{level: 40, wantRank: 1, wantSpell: 408490},
		{level: 50, wantRank: 2, wantSpell: 1238299},
		{level: 60, wantRank: 3, wantSpell: 1238300},
	}

	for _, tc := range cases {
		_, built := newLavaBurstShaman(t, tc.level)
		if len(built.LavaBurst) == 0 {
			t.Fatalf("level-%d shaman with the Lava Burst talent has no Lava Burst spell registered", tc.level)
		}
		maxRank := built.LavaBurst[len(built.LavaBurst)-1]
		if got := maxRank.Rank; got != tc.wantRank {
			t.Errorf("level %d: Lava Burst max rank = %d, want %d", tc.level, got, tc.wantRank)
		}
		if got := maxRank.ActionID.SpellID; got != tc.wantSpell {
			t.Errorf("level %d: Lava Burst max rank spell ID = %d, want %d", tc.level, got, tc.wantSpell)
		}
	}
}

// TestLavaBurstFlameShockInteraction: the talent's own text ("If your
// Flame Shock is on the target, Lava Burst deals 20% increased damage")
// is a flat multiplier, not Classic's guaranteed-crit convention. Two
// sims share the same RandomSeed; the only difference between them is
// that the "with Flame Shock" sim applies the Flame Shock DoT directly
// (Dot.Apply schedules ticks without rolling any RNG, so it doesn't
// perturb the Lava Burst hit/crit roll that follows), so both sims
// resolve the same hit/crit outcome for Lava Burst and the damage ratio
// isolates the multiplier exactly.
func TestLavaBurstFlameShockInteraction(t *testing.T) {
	simWithout, withoutShaman := newLavaBurstShaman(t, 60)
	targetWithout := simWithout.Encounter.TargetUnits[0]
	lavaBurstWithout := withoutShaman.LavaBurst[len(withoutShaman.LavaBurst)-1]

	if withoutShaman.GetActiveFlameShockSpell(targetWithout) != nil {
		t.Fatal("no Flame Shock was cast, but getActiveFlameShockSpell found one active")
	}
	lavaBurstWithout.ApplyEffects(simWithout, targetWithout, lavaBurstWithout)
	baseDamage := lavaBurstWithout.SpellMetrics[targetWithout.UnitIndex].TotalDamage
	if baseDamage <= 0 {
		t.Fatalf("Lava Burst without Flame Shock dealt %v damage, want > 0", baseDamage)
	}

	simWith, withShaman := newLavaBurstShaman(t, 60)
	targetWith := simWith.Encounter.TargetUnits[0]
	lavaBurstWith := withShaman.LavaBurst[len(withShaman.LavaBurst)-1]
	flameShockMaxRank := withShaman.FlameShock[len(withShaman.FlameShock)-1]
	flameShockMaxRank.Dot(targetWith).Apply(simWith)

	if withShaman.GetActiveFlameShockSpell(targetWith) == nil {
		t.Fatal("Flame Shock's DoT was applied directly, but getActiveFlameShockSpell found none active")
	}
	lavaBurstWith.ApplyEffects(simWith, targetWith, lavaBurstWith)
	bonusDamage := lavaBurstWith.SpellMetrics[targetWith.UnitIndex].TotalDamage

	wantBonusDamage := baseDamage * 1.20
	if diff := bonusDamage - wantBonusDamage; diff > 0.01 || diff < -0.01 {
		t.Errorf("Lava Burst with Flame Shock dealt %v damage, want %v (+20%% of %v)", bonusDamage, wantBonusDamage, baseDamage)
	}
}
