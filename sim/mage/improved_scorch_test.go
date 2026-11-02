package mage

import (
	"math"
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
	googleproto "google.golang.org/protobuf/proto"
)

const (
	improvedScorchMaxStacks   = 5
	improvedScorchDuration    = 30 * time.Second
	improvedScorchPerStack    = 0.03
	improvedScorchTalentRanks = 3
	// The raid's periodic application is five ticks 1.5 s apart.
	raidScorchRampUp = 10 * time.Second
)

func improvedScorchTalents(t *testing.T) string {
	t.Helper()
	return talentStringWithRank(t, ForeverFrostTalents, "improved_scorch", improvedScorchTalentRanks)
}

func debuffsWithoutImprovedScorch() *proto.Debuffs {
	debuffs := googleproto.Clone(core.FullBuffs.Debuffs).(*proto.Debuffs)
	debuffs.ImprovedScorch = false
	return debuffs
}

// stepUntil runs the sim's event queue up to the given time.
func stepUntil(sim *core.Simulation, until time.Duration) {
	for sim.CurrentTime < until {
		if sim.Step() {
			return
		}
	}
}

func fireDamageTakenMultiplier(target *core.Unit) float64 {
	return target.PseudoStats.SchoolDamageTakenMultiplier[stats.SchoolIndexFire]
}

// The mage's own Scorch and the raid's debuff toggle are one aura on the
// target, not two that happen to share an id: the APL reads one and both
// writers must feed it.
func TestOwnAndRaidImprovedScorchAreOneAura(t *testing.T) {
	sim, mage := newMageAtLevel(t, 60, improvedScorchTalents(t))
	target := sim.Encounter.TargetUnits[0]

	if mage.ImprovedScorchAuras.Get(target) != core.ImprovedScorchAura(target) {
		t.Error("the mage's Improved Scorch aura is not the raid debuff's aura instance")
	}
}

// Forever's Fire Vulnerability (22959, aura 270) is Fire damage taken from
// the mage who applied it, so another raid mage's stacks give this one
// nothing: the raid's Improved Scorch flag leaves the target's Fire damage
// taken where it was, and the mage keeps its own up with Scorch.
func TestRaidImprovedScorchGivesThisMageNothing(t *testing.T) {
	without, _ := newMageWithDebuffs(t, 60, improvedScorchTalents(t), debuffsWithoutImprovedScorch())
	with, _ := newMageWithDebuffs(t, 60, improvedScorchTalents(t), core.FullBuffs.Debuffs)
	stepUntil(without, raidScorchRampUp)
	stepUntil(with, raidScorchRampUp)

	got := fireDamageTakenMultiplier(with.Encounter.TargetUnits[0]) /
		fireDamageTakenMultiplier(without.Encounter.TargetUnits[0])
	if math.Abs(got-1) > 1e-9 {
		t.Errorf("Fire damage taken ratio with the raid's Improved Scorch = %v, want 1", got)
	}
}

// A Scorch cast on a fully stacked debuff adds no stack but renews the
// 30 seconds: the rotation relies on this to keep the debuff up by
// casting Scorch only as it runs low.
func TestOwnScorchRefreshesAFullyStackedDebuff(t *testing.T) {
	sim, mage := newMageWithDebuffs(t, 60, improvedScorchTalents(t), debuffsWithoutImprovedScorch())
	target := sim.Encounter.TargetUnits[0]
	scorch := mage.Scorch[core.MaxTrainerRank(ScorchRanks)]
	aura := mage.ImprovedScorchAuras.Get(target)

	for i := 0; i < improvedScorchMaxStacks; i++ {
		castUntilLanded(t, sim, target, scorch)
	}
	if aura.GetStacks() != improvedScorchMaxStacks {
		t.Fatalf("after five Scorch casts the debuff has %d stacks, want %d", aura.GetStacks(), improvedScorchMaxStacks)
	}

	stepUntil(sim, sim.CurrentTime+20*time.Second)
	if got := aura.RemainingDuration(sim); got > 10*time.Second+time.Millisecond {
		t.Fatalf("20 s after the last cast %v remain, want about 10 s", got)
	}

	castUntilLanded(t, sim, target, scorch)
	if got := aura.GetStacks(); got != improvedScorchMaxStacks {
		t.Errorf("a sixth Scorch left %d stacks, want %d", got, improvedScorchMaxStacks)
	}
	if got := aura.RemainingDuration(sim); got < improvedScorchDuration-time.Second {
		t.Errorf("a sixth Scorch left %v on the debuff, want a renewed %v", got, improvedScorchDuration)
	}
}

// Round 11 (casters M2): Improved Scorch (11095) is aura 109, a trigger
// the spell adds to the target it hits, so a Scorch that misses applies
// no Fire Vulnerability stack. At 3/3 every landed Scorch adds one.
func TestImprovedScorchStacksOnlyOnAHit(t *testing.T) {
	sim, mage := newMageWithDebuffs(t, 60, improvedScorchTalents(t), debuffsWithoutImprovedScorch())
	target := sim.Encounter.TargetUnits[0]
	scorch := topRank(t, mage.Scorch)
	aura := mage.ImprovedScorchAuras.Get(target)

	misses := 0
	for i := 0; i < 400; i++ {
		aura.Deactivate(sim)
		before := scorch.SpellMetrics[target.UnitIndex].Misses
		scorch.ApplyEffects(sim, target, scorch)
		missed := scorch.SpellMetrics[target.UnitIndex].Misses > before
		if missed {
			misses++
		}
		if missed == aura.IsActive() {
			t.Fatalf("cast %d: missed=%v but Improved Scorch active=%v", i, missed, aura.IsActive())
		}
	}
	if misses == 0 {
		t.Fatal("400 Scorches never missed; the test saw no miss to check")
	}
}

// castUntilLanded casts spell until one cast lands: only a landed Scorch
// adds or renews Improved Scorch.
func castUntilLanded(t *testing.T, sim *core.Simulation, target *core.Unit, spell *core.Spell) {
	t.Helper()
	for i := 0; i < 50; i++ {
		before := spell.SpellMetrics[target.UnitIndex].Misses
		castAndLand(sim, target, spell)
		if spell.SpellMetrics[target.UnitIndex].Misses == before {
			return
		}
	}
	t.Fatal("50 casts in a row missed")
}
