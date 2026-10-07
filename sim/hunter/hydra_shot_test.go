package hunter

import (
	"testing"
	"time"

	_ "github.com/wowsims/classic/sim/common" // imported to get item effects included.
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// newMultiTargetHunter builds a bare level-60 ranged hunter against
// targetCount standard level-60 targets, the shape Hydra Shot's chain
// needs to be driven.
func newMultiTargetHunter(t *testing.T, seed int64, targetCount int) (*core.Simulation, *Hunter) {
	t.Helper()

	targets := make([]*proto.Target, targetCount)
	for i := range targets {
		targets[i] = core.DefaultTargetProtoLvl60
	}
	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassHunter,
			Race:               proto.Race_RaceOrc,
			Level:              60,
			Equipment:          &proto.EquipmentSpec{},
			Buffs:              &proto.IndividualBuffs{},
			DistanceFromTarget: 25,
		},
		P1PlayerOptions,
	)
	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 60, Targets: targets},
		SimOptions: &proto.SimOptions{RandomSeed: seed, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()

	agent, ok := sim.Raid.Parties[0].Players[0].(HunterAgent)
	if !ok {
		t.Fatal("the raid's first player is not a hunter agent")
	}
	return sim, agent.GetHunter()
}

// stepUntilDamageLands drives the sim until the missile has landed on
// every target the cast hit (Hydra Shot's damage waits on travel time).
func stepUntilDamageLands(sim *core.Simulation, spell *core.Spell, target *core.Unit) {
	for i := 0; i < 200 && spell.SpellMetrics[target.UnitIndex].TotalDamage == 0; i++ {
		if sim.Step() {
			return
		}
	}
}

func TestHydraShotRegistersAtLevel60Only(t *testing.T) {
	_, built, _ := newBareHunterAtLevel(t, 60)
	if built.HydraShot == nil {
		t.Fatal("level-60 hunter has no Hydra Shot registered")
	}
	if got, want := built.HydraShot.ActionID.SpellID, int32(1293020); got != want {
		t.Errorf("Hydra Shot spell id = %d, want %d", got, want)
	}

	_, below, _ := newBareHunterAtLevel(t, 59)
	if below.HydraShot != nil {
		t.Error("level-59 hunter has Hydra Shot; the client teaches it at 60")
	}
}

func TestHydraShotSharesCooldownWithArcaneShot(t *testing.T) {
	sim, built, target := newBareHunterAtLevel(t, 60)

	if built.HydraShot.CD.Timer != built.ArcaneShot.CD.Timer {
		t.Fatal("Hydra Shot does not share Arcane Shot's cooldown timer (client tooltip: shares its cooldown with Arcane Shot)")
	}
	if got, want := built.HydraShot.CD.Duration, 6*time.Second; got != want {
		t.Errorf("Hydra Shot cooldown = %v, want %v (category_cooldown_ms 6000)", got, want)
	}
	if !built.HydraShot.Cast(sim, target) {
		t.Fatal("Hydra Shot could not be cast on a fresh level-60 hunter")
	}
	if built.ArcaneShot.IsReady(sim) {
		t.Error("Arcane Shot is ready right after Hydra Shot; the cooldown is shared")
	}
}

func TestHydraShotCostsClientMana(t *testing.T) {
	_, built, _ := newBareHunterAtLevel(t, 60)
	if got, want := built.HydraShot.Cost.BaseCost, 250.0; got != want {
		t.Errorf("Hydra Shot mana cost = %v, want %v", got, want)
	}
}

func TestHydraShotDeclaresClientBaseDamage(t *testing.T) {
	_, built, _ := newBareHunterAtLevel(t, 60)
	if got, want := built.HydraShot.ClientBaseDamage, [2]float64{260, 260}; got != want {
		t.Errorf("Hydra Shot ClientBaseDamage = %v, want %v (effect 121 amount 260)", got, want)
	}
}

func TestHydraShotTalentsAndMasks(t *testing.T) {
	_, plain, _ := newBareHunterAtLevel(t, 60)
	if !plain.HydraShot.Flags.Matches(SpellFlagShot) {
		t.Error("Hydra Shot is not flagged as a Shot, so Efficiency and the Shot talents skip it")
	}
	if !plain.HydraShot.ProcMask.Matches(core.ProcMaskRangedSpecial) {
		t.Error("Hydra Shot does not carry the ranged special proc mask")
	}

	_, mortal, _ := newRunningHunterForTalentTest(t, 60, "mortal_shots", 5, proto.Hunter_Options_PetNone, 25, nil)
	if got, want := mortal.HydraShot.CritDamageBonus, 1.30; got < want-1e-9 || got > want+1e-9 {
		t.Errorf("Hydra Shot crit damage bonus with Mortal Shots 5/5 = %v, want %v", got, want)
	}

	_, rws, _ := newRunningHunterForTalentTest(t, 60, "ranged_weapon_specialization", 5, proto.Hunter_Options_PetNone, 25, nil)
	if got, want := rws.HydraShot.DamageMultiplier, 1.05; got < want-1e-9 || got > want+1e-9 {
		t.Errorf("Hydra Shot damage multiplier with Ranged Weapon Specialization 5/5 = %v, want %v", got, want)
	}

	_, eff, _ := newRunningHunterForTalentTest(t, 60, "efficiency", 5, proto.Hunter_Options_PetNone, 25, nil)
	if eff.HydraShot.Cost.Multiplier >= plain.HydraShot.Cost.Multiplier {
		t.Error("Efficiency 5/5 did not reduce Hydra Shot's mana cost")
	}
}

func TestHydraShotIsRangedOnly(t *testing.T) {
	sim, built, target := newBareMeleeHunterAtLevel(t, 60)
	if built.HydraShot.CanCast(sim, target) {
		t.Error("Hydra Shot is castable in melee range; the client range is 8-35 yd")
	}
}

func TestHydraShotSingleTargetDealsDamage(t *testing.T) {
	sim, built, target := newBareHunterAtLevel(t, 60)
	built.HydraShot.ApplyEffects(sim, target, built.HydraShot)
	stepUntilDamageLands(sim, built.HydraShot, target)
	if built.HydraShot.SpellMetrics[target.UnitIndex].TotalDamage <= 0 {
		t.Error("Hydra Shot dealt no damage to its initial target")
	}
}

// TestHydraShotChainsFiveTargetsAtSixtyFivePercent checks the chain: up
// to 5 targets, each jump dealing 35% less than the one before. Over
// many seeds the unit-by-unit mean damage ratio converges on 0.65.
func TestHydraShotChainsFiveTargetsAtSixtyFivePercent(t *testing.T) {
	const casts = 60
	totals := make([]float64, 6)
	for seed := int64(1); seed <= casts; seed++ {
		sim, built := newMultiTargetHunter(t, seed, 6)
		target := sim.Encounter.TargetUnits[0]
		built.HydraShot.ApplyEffects(sim, target, built.HydraShot)
		for i := 0; i < 200; i++ {
			if sim.Step() {
				break
			}
		}
		for i, unit := range sim.Encounter.TargetUnits {
			totals[i] += built.HydraShot.SpellMetrics[unit.UnitIndex].TotalDamage
		}
	}
	if totals[5] != 0 {
		t.Errorf("Hydra Shot hit a sixth target for %v damage; the chain stops at 5", totals[5])
	}
	for i := 1; i < 5; i++ {
		ratio := totals[i] / totals[i-1]
		if ratio < 0.55 || ratio > 0.75 {
			t.Errorf("target %d took %.2f of target %d's damage, want about 0.65", i+1, ratio, i)
		}
	}
}
