package rogue_test

import (
	"math"
	"testing"
	"time"

	"github.com/wowsims/classic/sim/common/clientsetbonus"
	"github.com/wowsims/classic/sim/common/clientsetbonus/clientsetbonustest"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	rogueclass "github.com/wowsims/classic/sim/rogue"
	dpsrogue "github.com/wowsims/classic/sim/rogue/dps_rogue"
)

const (
	bonescytheSetID          int32 = 524
	bonescytheHeadRushRow    int32 = 28812
	bonescytheThreatRow      int32 = 28811
	bonescytheRevealedFlaw   int32 = 28815
	bonescytheHeadRushPieces       = 4
	bonescytheHeadRushEnergy int32 = 28813
	bonescytheThreatPieces         = 6
	bonescytheFlawPieces           = 8
	bonescytheFlawLabel            = "Revealed Flaw"
	maxComboPoints                 = 5
	revealedFlawTrials             = 4_000
)

func bonescytheRogue(t *testing.T, pieces int) (*core.Simulation, *dpsrogue.DpsRogue) {
	t.Helper()
	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassRogue,
			Race:               proto.Race_RaceHuman,
			Level:              60,
			Buffs:              core.FullBuffs.Player,
			DistanceFromTarget: 5,
		},
		&proto.Player_Rogue{Rogue: &proto.Rogue{Options: &proto.RogueOptions{}}},
	)
	sim := clientsetbonustest.PrePulledSim(t, player, bonescytheSetID, pieces)
	built, ok := sim.Raid.Parties[0].Players[0].(*dpsrogue.DpsRogue)
	if !ok {
		t.Fatal("player 0 did not build as a *dpsrogue.DpsRogue")
	}
	return sim, built
}

// 6P: "Reduces the threat from your Backstab, Sinister Strike,
// Hemorrhage, and Eviscerate abilities" by the row's percent.
func TestBonescytheSixPieceCutsThreatByTheRowPercent(t *testing.T) {
	_, five := bonescytheRogue(t, bonescytheThreatPieces-1)
	_, six := bonescytheRogue(t, bonescytheThreatPieces)
	want := 1 + clientsetbonus.PercentModifier(bonescytheThreatRow, clientsetbonus.ModOpThreat)
	for name, spells := range map[string][2]*core.Spell{
		"Backstab":        {five.GetRogue().Backstab, six.GetRogue().Backstab},
		"Sinister Strike": {five.GetRogue().SinisterStrike, six.GetRogue().SinisterStrike},
		"Eviscerate":      {five.GetRogue().Eviscerate, six.GetRogue().Eviscerate},
	} {
		if got := spells[1].ThreatMultiplier / spells[0].ThreatMultiplier; math.Abs(got-want) > 1e-9 {
			t.Errorf("%s threat x%v, want the client's x%v", name, got, want)
		}
	}
}

// spendFinisher spends a full set of combo points on Eviscerate.
func spendFinisher(sim *core.Simulation, rogue *dpsrogue.DpsRogue) {
	character := rogue.GetCharacter()
	metrics := character.NewComboPointMetrics(core.ActionID{SpellID: 1})
	character.AddComboPointsIgnoreTarget(sim, maxComboPoints, metrics)
	character.SpendComboPoints(sim, rogue.GetRogue().Eviscerate)
}

// 8P: Revealed Flaw is a chance on Eviscerate and waits the buff spell's
// own cooldown before it can proc again.
func TestBonescytheEightPieceRevealedFlawWaitsItsCooldown(t *testing.T) {
	sim, rogue := bonescytheRogue(t, bonescytheFlawPieces)
	flaw := rogue.GetCharacter().GetAura(bonescytheFlawLabel)
	proc := func() bool {
		flaw.Deactivate(sim)
		for i := 0; i < revealedFlawTrials; i++ {
			spendFinisher(sim, rogue)
			if flaw.IsActive() {
				return true
			}
		}
		return false
	}
	if !proc() {
		t.Fatalf("no Revealed Flaw in %d finishers", revealedFlawTrials)
	}
	if proc() {
		t.Fatal("Revealed Flaw procs again inside its cooldown")
	}
	sim.CurrentTime += time.Duration(core.MustClientSpellRow(bonescytheRevealedFlaw).CooldownMS) * time.Millisecond
	if !proc() {
		t.Fatalf("no Revealed Flaw in %d finishers after its cooldown", revealedFlawTrials)
	}
}

// 4P: a Backstab, Sinister Strike or Hemorrhage crit returns the row's
// energy, once per internal cooldown.
func TestBonescytheFourPieceHeadRushReturnsEnergyOncePerCooldown(t *testing.T) {
	sim, rogue := bonescytheRogue(t, bonescytheHeadRushPieces)
	character := rogue.GetCharacter()
	head := character.GetAura("Head Rush")
	if head == nil {
		t.Fatal("four pieces do not carry Head Rush")
	}
	spent := character.NewEnergyMetrics(core.ActionID{SpellID: 1})
	crit := func() float64 {
		character.SpendEnergy(sim, character.CurrentEnergy(), spent)
		head.OnSpellHitDealt(head, sim, &core.Spell{SpellCode: rogueclass.SpellCode_RogueBackstab},
			&core.SpellResult{Outcome: core.OutcomeCrit, Target: character.CurrentTarget, Damage: 1})
		return character.CurrentEnergy()
	}
	want := core.MustClientSpellRow(bonescytheHeadRushEnergy).Effects[0].Points
	if got := crit(); got != want {
		t.Fatalf("a crit returns %v energy, want the client's %v", got, want)
	}
	if got := crit(); got != 0 {
		t.Errorf("a second crit inside the internal cooldown returns %v energy", got)
	}
	sim.CurrentTime += time.Duration(core.MustClientSpellRow(bonescytheHeadRushRow).InternalCooldownMS) * time.Millisecond
	if got := crit(); got != want {
		t.Errorf("a crit after the internal cooldown returns %v energy, want %v", got, want)
	}
}
