package mage

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// Blizzard's 9 November hotfix (us.forums.blizzard.com t/2403040, post 5):
// "Arcane Concentration's activation chance now correctly splits across the
// cast and every bolt (was granting 10% on cast and per bolt)." At 5/5 an
// Arcane Missiles channel rolls its 10% once, shared by the cast and its
// five missiles, so about one channel in ten leaves Clearcasting (1 -
// (1 - 0.1/6)^6 = 9.6%); one 10% roll per missile left it after 41% of them.
// The client row (11213) is unchanged: the split is server side.
func TestArcaneConcentrationSplitsAcrossTheCastAndEveryMissile(t *testing.T) {
	const channels = 600
	built, sim, target := longFrostMageSim(t, channels*8*time.Second)
	if built.ClearcastingAura == nil {
		t.Fatal("Clearcasting is not registered with Arcane Concentration taken")
	}
	am := topRank(t, built.ArcaneMissiles)
	refill := built.NewManaMetrics(core.ActionID{SpellID: 11213})
	procs := 0
	for i := 0; i < channels; i++ {
		built.AddMana(sim, built.MaxMana(), refill)
		if !am.Cast(sim, target) {
			t.Fatalf("Arcane Missiles did not cast on channel %d at %v (casting %v, channeling %v, GCD ready %v, mana %.0f)",
				i, sim.CurrentTime, built.IsCasting(sim), built.IsChanneling(sim), built.GCD.IsReady(sim), built.CurrentMana())
		}
		// The channel and the last missile's flight; the empty rotation
		// leaves the GCD parked, so it is released by hand.
		stepUntil(sim, sim.CurrentTime+7*time.Second)
		built.GCD.Reset()
		if built.ClearcastingAura.IsActive() {
			procs++
			built.ClearcastingAura.Deactivate(sim)
		}
	}
	if rate := float64(procs) / channels; rate < 0.05 || rate > 0.16 {
		t.Errorf("Arcane Missiles left Clearcasting after %.1f%% of channels, want about 9.6%% (one 10%% roll split over the cast and five missiles)", 100*rate)
	}
}

// longFrostMageSim is the reference Frost build (Arcane Concentration 5/5)
// against a training dummy for a fight long enough to count procs over.
func longFrostMageSim(t *testing.T, fight time.Duration) (*Mage, *core.Simulation, *core.Unit) {
	t.Helper()
	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassMage,
			Race:               proto.Race_RaceTroll,
			Level:              60,
			Equipment:          &proto.EquipmentSpec{},
			TalentsString:      ForeverFrostTalents,
			DistanceFromTarget: 5,
			Rotation:           &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
		},
		PlayerOptions,
	)
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       core.SinglePlayerRaidProto(player, nil, nil, nil),
		Encounter:  &proto.Encounter{Duration: fight.Seconds(), Targets: []*proto.Target{core.DefaultTargetProtoLvl60}},
		SimOptions: &proto.SimOptions{RandomSeed: 7, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()
	built := sim.Raid.Parties[0].Players[0].(MageAgent).GetMage()
	return built, sim, sim.Encounter.TargetUnits[0]
}
