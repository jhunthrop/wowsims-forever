package dpsrogue

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/rogue"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// rogueTalentStringWithRanks is rogueTalentStringWith with a rank per field.
func rogueTalentStringWithRanks(t *testing.T, ranks map[string]int) string {
	t.Helper()
	segments := strings.Split(rogueTalentStringWith(t), "-")
	for name, rank := range ranks {
		fd := (&proto.RogueTalents{}).ProtoReflect().Descriptor().Fields().ByName(protoreflect.Name(name))
		if fd == nil {
			t.Fatalf("RogueTalents has no field named %q", name)
		}
		pos := int(fd.Number()) - 1
		treeIdx := 0
		for treeIdx < len(rogue.TalentTreeSizes) && pos >= rogue.TalentTreeSizes[treeIdx] {
			pos -= rogue.TalentTreeSizes[treeIdx]
			treeIdx++
		}
		chars := []rune(segments[treeIdx])
		chars[pos] = rune('0' + rank)
		segments[treeIdx] = string(chars)
	}
	return strings.Join(segments, "-")
}

// TestShankIsBaselineAtItsClientValues pins Shank rank 2 (1329549, level
// 44): 5 Energy, a 10 sec cooldown, and +25 Energy (e1) on every cast.
func TestShankIsBaselineAtItsClientValues(t *testing.T) {
	sim, built := buildRogueForTest(t, rogueTalentStringWith(t))
	if built.Shank == nil {
		t.Fatal("level-60 rogue has no Shank")
	}
	if got := built.Shank.ActionID.SpellID; got != 1329549 {
		t.Errorf("Shank id = %d, want rank 2's 1329549", got)
	}
	if got := built.Shank.DefaultCast.Cost; got != 5 {
		t.Errorf("Shank cost = %v, want 5 Energy", got)
	}
	if got := built.Shank.CD.Duration; got != 10*time.Second {
		t.Errorf("Shank cooldown = %v, want 10s", got)
	}
	target := sim.Encounter.TargetUnits[0]
	built.SpendEnergy(sim, built.CurrentEnergy(), built.Shank.EnergyMetrics())
	built.Shank.ApplyEffects(sim, target, built.Shank)
	if got := built.CurrentEnergy(); math.Abs(got-25) > 1e-9 {
		t.Errorf("Energy after one Shank from empty = %v, want 25", got)
	}
	if got := built.ComboPoints(); got != 0 {
		t.Errorf("untalented Shank awarded %d combo points, want 0", got)
	}
}

// TestFlashOfSteelAddsEnergyAndCutsCooldown pins 1329284's curves at rank
// 5: +15 Energy and -2.0 sec.
func TestFlashOfSteelAddsEnergyAndCutsCooldown(t *testing.T) {
	sim, built := buildRogueForTest(t, rogueTalentStringWithRanks(t, map[string]int{"flash_of_steel": 5}))
	if got := built.Shank.CD.Duration; got != 8*time.Second {
		t.Errorf("Shank cooldown with Flash of Steel 5 = %v, want 8s", got)
	}
	built.SpendEnergy(sim, built.CurrentEnergy(), built.Shank.EnergyMetrics())
	built.Shank.ApplyEffects(sim, sim.Encounter.TargetUnits[0], built.Shank)
	if got := built.CurrentEnergy(); math.Abs(got-40) > 1e-9 {
		t.Errorf("Energy after one Shank with Flash of Steel 5 = %v, want 25+15", got)
	}
}

// TestBootKnifeAwardsTwoComboPoints pins 1329286 rank 5 (100%) and its
// trigger 1329285 (2 combo points).
func TestBootKnifeAwardsTwoComboPoints(t *testing.T) {
	sim, built := buildRogueForTest(t, rogueTalentStringWithRanks(t, map[string]int{"boot_knife": 5}))
	built.Shank.ApplyEffects(sim, sim.Encounter.TargetUnits[0], built.Shank)
	if got := built.ComboPoints(); got != 2 {
		t.Errorf("combo points after a Boot Knife 5 Shank = %d, want 2", got)
	}
}

// TestShankProcsNoPoison: Shank's Attributes_3 suppress caster and target
// procs, so a hundred Shanks never apply Instant Poison.
func TestShankProcsNoPoison(t *testing.T) {
	sim, built := buildRogueForTest(t, rogueTalentStringWith(t))
	target := sim.Encounter.TargetUnits[0]
	for i := 0; i < 100; i++ {
		built.Shank.ApplyEffects(sim, target, built.Shank)
	}
	if built.InstantPoison != nil && built.InstantPoison.SpellMetrics[target.UnitIndex].Casts != 0 {
		t.Errorf("Shank applied Instant Poison %d times, want 0", built.InstantPoison.SpellMetrics[target.UnitIndex].Casts)
	}
}

// TestVenomIsAShankPassive: 70755's Venom (1329328) is no finisher; Shank
// applies 1310703 for 6 sec, +30% poison damage and +20 points of poison
// chance, and both revert when it ends.
func TestVenomIsAShankPassive(t *testing.T) {
	sim, built := buildRogueForTest(t, rogueTalentStringWith(t, "venom"))
	target := sim.Encounter.TargetUnits[0]

	baseMultiplier := built.InstantPoison.DamageMultiplier
	baseProcChance := built.GetInstantPoisonProcChance()

	built.Shank.ApplyEffects(sim, target, built.Shank)
	if !built.VenomAura.IsActive() {
		t.Fatal("Venom's aura is not active after Shank")
	}
	if got := built.VenomAura.Duration; got != 6*time.Second {
		t.Errorf("Venom duration = %v, want 6s", got)
	}
	if got, want := built.InstantPoison.DamageMultiplier, baseMultiplier*1.30; math.Abs(got-want) > 1e-9 {
		t.Errorf("Instant Poison DamageMultiplier while Venom is up = %v, want %v", got, want)
	}
	if got, want := built.GetInstantPoisonProcChance(), baseProcChance+0.20; math.Abs(got-want) > 1e-9 {
		t.Errorf("Instant Poison proc chance while Venom is up = %v, want %v", got, want)
	}

	built.VenomAura.Deactivate(sim)
	if got := built.InstantPoison.DamageMultiplier; got != baseMultiplier {
		t.Errorf("Instant Poison DamageMultiplier after Venom = %v, want %v", got, baseMultiplier)
	}
	if got := built.GetInstantPoisonProcChance(); got != baseProcChance {
		t.Errorf("Instant Poison proc chance after Venom = %v, want %v", got, baseProcChance)
	}
}

// TestNoVenomWithoutTheTalent: Shank without the point leaves no Venom.
func TestNoVenomWithoutTheTalent(t *testing.T) {
	sim, built := buildRogueForTest(t, rogueTalentStringWith(t))
	built.Shank.ApplyEffects(sim, sim.Encounter.TargetUnits[0], built.Shank)
	if built.VenomAura != nil {
		t.Error("untalented rogue has a Venom aura")
	}
}
