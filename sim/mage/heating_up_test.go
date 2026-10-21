package mage

import (
	"testing"
	"time"

	_ "github.com/wowsims/classic/sim/common" // imported to get item effects included.
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// newHeatingUpTestMage builds a level-60 mage with Pyroblast and Heating
// Up taken, prepulled so spells can be driven through ApplyEffects.
func newHeatingUpTestMage(t *testing.T) (*core.Simulation, *Mage) {
	t.Helper()
	talentsStr := talentStringWithRank(t, ForeverFrostTalents, "pyroblast", 1)
	talentsStr = talentStringWithRank(t, talentsStr, "heating_up", 1)

	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassMage,
			Race:               proto.Race_RaceTroll,
			Level:              60,
			Equipment:          &proto.EquipmentSpec{},
			Buffs:              core.FullBuffs.Player,
			TalentsString:      talentsStr,
			DistanceFromTarget: 5,
			Rotation:           &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
		},
		PlayerOptions,
	)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid: raid,
		Encounter: &proto.Encounter{
			Duration: 120,
			Targets:  []*proto.Target{core.DefaultTargetProtoLvl60},
		},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()

	agent, ok := sim.Raid.Parties[0].Players[0].(MageAgent)
	if !ok {
		t.Fatal("the raid's first player is not a mage agent")
	}
	return sim, agent.GetMage()
}

// forceOutcome makes the spell always land and either always or never
// crit, so a test does not depend on the seed.
func forceOutcome(spell *core.Spell, crit bool) {
	spell.BonusHitRating += 100 * core.HitRatingPerHitChance
	if crit {
		spell.BonusCritRating += 100 * core.CritRatingPerCritChance
	} else {
		spell.BonusCritRating -= 100 * core.CritRatingPerCritChance
	}
}

func castAndLand(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
	spell.ApplyEffects(sim, target, spell)
	waitForOutcomes(sim)
}

func heatingUpStacks(mage *Mage) int32 {
	if !mage.HeatingUpAura.IsActive() {
		return 0
	}
	return mage.HeatingUpAura.GetStacks()
}

// Pyroblast's cast time is the client's (6 s on every player rank), and
// without the buff nothing shortens it.
func TestPyroblastCastTimeReadsTheClientTable(t *testing.T) {
	_, mage := newHeatingUpTestMage(t)
	pyroblast := mage.Pyroblast[PyroblastRanks]
	if pyroblast == nil {
		t.Fatal("level-60 mage with the talent has no top-rank Pyroblast")
	}
	want := time.Duration(PyroblastCastTime[PyroblastRanks]) * time.Millisecond
	if got := pyroblast.CastTime(); got != want {
		t.Errorf("Pyroblast cast time = %v, want the client's %v", got, want)
	}
}

// "Non-periodic critical strikes with Fireball, Frostfire Bolt, Fire
// Blast, and Scorch reduce the cast time of your next Pyroblast cast
// within 20 sec by 25%, stacking up to 3 times."
func TestHeatingUpStacksOnFireCritsAndShortensPyroblast(t *testing.T) {
	sim, mage := newHeatingUpTestMage(t)
	target := sim.Encounter.TargetUnits[0]
	pyroblast := mage.Pyroblast[PyroblastRanks]
	base := time.Duration(PyroblastCastTime[PyroblastRanks]) * time.Millisecond

	feeders := []*core.Spell{mage.Fireball[core.MaxTrainerRank(FireballRanks)], mage.Scorch[ScorchRanks], mage.FireBlast[FireBlastRanks]}
	wantCast := []time.Duration{base * 75 / 100, base * 50 / 100, base * 25 / 100}
	for i, feeder := range feeders {
		if feeder == nil {
			t.Fatalf("feeder spell %d is not registered", i)
		}
		forceOutcome(feeder, true)
		castAndLand(sim, target, feeder)
		if got, want := heatingUpStacks(mage), int32(i+1); got != want {
			t.Fatalf("stacks after crit %d = %d, want %d", i+1, got, want)
		}
		if got := pyroblast.CastTime(); got != wantCast[i] {
			t.Errorf("Pyroblast cast time at %d stack(s) = %v, want %v", i+1, got, wantCast[i])
		}
	}

	// A fourth crit stays at the cap.
	castAndLand(sim, target, feeders[0])
	if got := heatingUpStacks(mage); got != 3 {
		t.Errorf("stacks after a fourth crit = %d, want the cap of 3", got)
	}
}

func TestHeatingUpIgnoresNonCritsAndPyroblastItself(t *testing.T) {
	sim, mage := newHeatingUpTestMage(t)
	target := sim.Encounter.TargetUnits[0]

	fireball := mage.Fireball[core.MaxTrainerRank(FireballRanks)]
	forceOutcome(fireball, false)
	castAndLand(sim, target, fireball)
	if got := heatingUpStacks(mage); got != 0 {
		t.Errorf("a non-crit Fireball gave %d stack(s), want 0", got)
	}

	pyroblast := mage.Pyroblast[PyroblastRanks]
	forceOutcome(pyroblast, true)
	castAndLand(sim, target, pyroblast)
	if got := heatingUpStacks(mage); got != 0 {
		t.Errorf("a Pyroblast crit gave %d stack(s), want 0: it is not one of the four feeders", got)
	}
}

func TestHeatingUpIsSpentByTheNextPyroblast(t *testing.T) {
	sim, mage := newHeatingUpTestMage(t)
	target := sim.Encounter.TargetUnits[0]
	fireball := mage.Fireball[core.MaxTrainerRank(FireballRanks)]
	forceOutcome(fireball, true)
	castAndLand(sim, target, fireball)
	castAndLand(sim, target, fireball)

	pyroblast := mage.Pyroblast[PyroblastRanks]
	if !pyroblast.Cast(sim, target) {
		t.Fatal("Pyroblast would not cast")
	}
	advanceBy(sim, 4*time.Second)
	if got := heatingUpStacks(mage); got != 0 {
		t.Errorf("stacks after casting Pyroblast = %d, want 0 (the buff is spent)", got)
	}
	if got, want := pyroblast.CastTime(), time.Duration(PyroblastCastTime[PyroblastRanks])*time.Millisecond; got != want {
		t.Errorf("Pyroblast cast time after the buff was spent = %v, want %v", got, want)
	}
}

func TestHeatingUpExpiresAfterTwentySecondsAndRefreshesOnCrit(t *testing.T) {
	sim, mage := newHeatingUpTestMage(t)
	target := sim.Encounter.TargetUnits[0]
	fireball := mage.Fireball[core.MaxTrainerRank(FireballRanks)]
	forceOutcome(fireball, true)
	castAndLand(sim, target, fireball)

	if got, want := mage.HeatingUpAura.Duration, 20*time.Second; got != want {
		t.Fatalf("Heating Up duration = %v, want %v", got, want)
	}
	start := mage.HeatingUpAura.RemainingDuration(sim)
	advanceBy(sim, 10*time.Second)
	castAndLand(sim, target, fireball)
	if got := mage.HeatingUpAura.RemainingDuration(sim); got <= start-time.Second {
		t.Errorf("a second crit left %v, want the 20 s clock restarted", got)
	}
	if got := heatingUpStacks(mage); got != 2 {
		t.Errorf("stacks = %d, want 2", got)
	}

	advanceBy(sim, 21*time.Second)
	if got := heatingUpStacks(mage); got != 0 {
		t.Errorf("stacks after 21 s without a crit = %d, want 0", got)
	}
	if got, want := mage.Pyroblast[PyroblastRanks].CastTime(), time.Duration(PyroblastCastTime[PyroblastRanks])*time.Millisecond; got != want {
		t.Errorf("Pyroblast cast time after expiry = %v, want %v", got, want)
	}
}

// advanceBy runs the sim's event queue up to d from now.
func advanceBy(sim *core.Simulation, d time.Duration) {
	until := sim.CurrentTime + d
	sim.AddPendingAction(&core.PendingAction{NextActionAt: until, OnAction: func(*core.Simulation) {}})
	for sim.CurrentTime < until {
		if sim.Step() {
			return
		}
	}
}
