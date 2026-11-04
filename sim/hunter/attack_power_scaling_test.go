package hunter

import (
	"math"
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
)

// Round 20 (hunter lane). Three hunter spells scale with attack power on
// the Forever server while every one of their client SpellEffect rows
// carries BonusCoefficientFromAP 0:
//
//   - Serpent Sting, 5% of ranged attack power a tick: "Serpent Sting:
//     Attack Power coefficient per tick increased to 5% from 3%" (WoW
//     Forever Beta Development Notes, updated 3 November 2026). The beta
//     logs of 2-10 October (eight hunters, reports 2695-2764) tick 3% of
//     the hunter's ranged attack power over the pre-push base.
//   - Arcane Shot, 15% of ranged attack power: "Arcane Shot: ... Attack
//     Power coefficient raised from 0.1 to 0.15" (same notes). The same
//     logs hit 10% over the pre-push base.
//   - Immolation Trap, 2% of ranged attack power a tick: "The Hunter's
//     Fire traps now correctly scale with Attack Power" (notes of 16
//     October 2026); the coefficient is the forever-hunter wiki's tested
//     value (Attack Formulas, "2% AP per tick", single source).

func newRangedHunterAt60(t *testing.T) (*core.Simulation, *Hunter, *core.Unit) {
	t.Helper()
	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassHunter,
			Race:               proto.Race_RaceOrc,
			Level:              60,
			Equipment:          &proto.EquipmentSpec{},
			TalentsString:      "5320000501000000-0051550001503050-500000000000000000",
			DistanceFromTarget: 25,
		},
		P1PlayerOptions,
	)
	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid: raid,
		Encounter: &proto.Encounter{
			Duration: 30,
			Targets:  []*proto.Target{core.DefaultTargetProtoLvl60},
		},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	built := sim.Raid.Parties[0].Players[0].(HunterAgent).GetHunter()
	return sim, built, &sim.Encounter.Targets[0].Unit
}

func TestSerpentStingTicksScaleWithRangedAttackPower(t *testing.T) {
	sim, built, target := newRangedHunterAt60(t)
	rap := built.GetStat(stats.RangedAttackPower)
	if rap < 100 {
		t.Fatalf("a naked level-60 hunter has %.0f ranged attack power, want enough to see 5%% of it", rap)
	}
	dot := built.SerpentSting.Dot(target)
	dot.Apply(sim)
	want := SerpentStingDamage[8].Amount + 0.05*rap
	if got := dot.SnapshotBaseDamage; math.Abs(got-want) > 0.5 {
		t.Errorf("Serpent Sting rank 8 tick = %.1f, want %.1f (100 + 5%% of %.0f ranged attack power)", got, want, rap)
	}
}

func TestArcaneShotScalesWithRangedAttackPower(t *testing.T) {
	// The Beast Mastery raid build, with no pet out: no Ranged Weapon
	// Specialization, Improved Tracking, Focused Fire or Lone Wolf
	// multiplies the hit, so the hit is the formula itself.
	sim, built, target := newRunningHunterWithTalents(t, 60, "5420001505001251-00523500014-0", proto.Hunter_Options_PetNone, 25, nil)
	rap := built.GetStat(stats.RangedAttackPower)
	// Every shot lands and none crits, so the hit is the formula itself.
	built.AddStatDynamic(sim, stats.Hit, 100*core.HitRatingPerHitChance)
	built.AddStatDynamic(sim, stats.Crit, -200*core.CritRatingPerCritChance)

	// The hit lands through a delayed action (WaitTravelTime): no flight,
	// then step the sim until it has run.
	built.ArcaneShot.MissileSpeed = 0
	built.ArcaneShot.ApplyEffects(sim, target, built.ArcaneShot)
	stepUntil(sim, sim.CurrentTime+time.Second)
	metrics := built.ArcaneShot.SpellMetrics[target.UnitIndex]
	if metrics.Hits != 1 || metrics.Crits != 0 {
		t.Fatalf("Arcane Shot: hits %d, crits %d, misses %d; want exactly one hit", metrics.Hits, metrics.Crits, metrics.Misses)
	}
	want := ArcaneShotDamage[8].Amount + 0.15*rap
	if got := metrics.TotalDamage; math.Abs(got-want) > 1 {
		t.Errorf("Arcane Shot rank 8 hit = %.1f, want %.1f (280 + 15%% of %.0f ranged attack power)", got, want, rap)
	}
}

func TestImmolationTrapTicksScaleWithRangedAttackPower(t *testing.T) {
	sim, built, target := buildSurvivalMeleeHunter(t)
	rap := built.GetStat(stats.RangedAttackPower)
	if rap < 100 {
		t.Fatalf("a level-60 Survival hunter has %.0f ranged attack power, want enough to see 2%% of it", rap)
	}
	dot := built.ImmolationTrap.Dot(target)
	dot.Apply(sim)
	want := ImmolationTrapTickDamage[5].Amount + 0.02*rap
	if got := dot.SnapshotBaseDamage; math.Abs(got-want) > 0.5 {
		t.Errorf("Immolation Trap rank 5 tick = %.1f, want %.1f (138 + 2%% of %.0f ranged attack power)", got, want, rap)
	}
}
