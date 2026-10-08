package dpsrogue

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/rogue"
	googleproto "google.golang.org/protobuf/proto"
)

// The 1.60.1.70009 client's Flametongue Totem (spells 8227 to 16387, area
// auras 8230 to 15036, proc spells 8253 to 16389) is an aura on every party
// member: each main-hand auto attack that lands adds fire damage of the
// rank's amount times the weapon's speed over 100. The raid buff
// FlametongueTotem is how a request states that; the client's beta notes
// say it does not stack with the Windfury Totem aura.

const flametongueTotemRank4Proc int32 = 16389

func raidBuffsWithTotems(windfury, flametongue bool) *proto.RaidBuffs {
	buffs := googleproto.Clone(core.FullBuffs.Raid).(*proto.RaidBuffs)
	buffs.WindfuryTotem = windfury
	buffs.FlametongueTotem = flametongue
	return buffs
}

// flametongueProcCasts runs the rogue's auto attacks for the encounter and
// returns how many times the totem's proc spell fired and the damage it
// dealt.
func flametongueProcCasts(t *testing.T, built *rogue.Rogue, sim *core.Simulation) (int32, float64) {
	t.Helper()
	for !sim.Step() {
	}
	spell := built.GetSpell(core.ActionID{SpellID: flametongueTotemRank4Proc})
	if spell == nil {
		return 0, 0
	}
	metrics := spell.SpellMetrics[sim.GetTargetUnit(0).UnitIndex]
	return metrics.Casts, metrics.TotalDamage
}

func TestFlametongueTotemRaidBuffGivesTheRogueTheTotemAura(t *testing.T) {
	_, without := buildRogueWithRaidBuffs(t, "", raidBuffsWithTotems(false, false))
	if without.GetAura(core.FlametongueTotemAuraLabel) != nil {
		t.Fatal("a rogue without the Flametongue Totem buff has its aura")
	}
	_, with := buildRogueWithRaidBuffs(t, "", raidBuffsWithTotems(false, true))
	if with.GetAura(core.FlametongueTotemAuraLabel) == nil {
		t.Fatal("the Flametongue Totem raid buff did not register the aura on the rogue")
	}
}

func TestFlametongueTotemAddsFireDamageToMainHandAutoAttacks(t *testing.T) {
	sim, built := buildRogueWithRaidBuffs(t, "", raidBuffsWithTotems(false, true))
	casts, damage := flametongueProcCasts(t, built, sim)
	if casts == 0 || damage <= 0 {
		t.Fatalf("Flametongue Totem proc casts = %d, damage = %v; want procs on every main-hand auto", casts, damage)
	}
	// Rank 4 hits for 1363 x 1.8 / 100 = 24.5 before crits and resists; the
	// average over the run sits within a wide band of it.
	expected := 1363 * testMHWeapon.SwingSpeed / 100
	if average := damage / float64(casts); average < 0.5*expected || average > 1.5*expected {
		t.Errorf("average proc damage = %.1f, want about %.1f", average, expected)
	}
}

func TestFlametongueTotemDoesNotProcBesideTheWindfuryTotem(t *testing.T) {
	sim, built := buildRogueWithRaidBuffs(t, "", raidBuffsWithTotems(true, true))
	if casts, _ := flametongueProcCasts(t, built, sim); casts != 0 {
		t.Fatalf("Flametongue Totem fired %d times beside the Windfury Totem, want 0", casts)
	}
}
