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

// The raid debuff is a full-uptime stand-in for a raid mage: five stacks
// within the ramp-up, then held to the end of the fight.
func TestRaidImprovedScorchReachesFiveStacksAndStays(t *testing.T) {
	sim, _ := newMageWithDebuffs(t, 60, improvedScorchTalents(t), core.FullBuffs.Debuffs)
	aura := core.ImprovedScorchAura(sim.Encounter.TargetUnits[0])

	for _, at := range []time.Duration{raidScorchRampUp, 2 * improvedScorchDuration, 100 * time.Second} {
		stepUntil(sim, at)
		if !aura.IsActive() || aura.GetStacks() != improvedScorchMaxStacks {
			t.Fatalf("at %v the raid debuff has active=%v stacks=%d, want active with %d",
				at, aura.IsActive(), aura.GetStacks(), improvedScorchMaxStacks)
		}
	}
}

// Five stacks make Fire damage taken 15 percent higher; with the raid
// debuff the engine's number must be exactly that, on top of whatever
// else the target carries.
func TestRaidImprovedScorchRaisesFireDamageTakenByFifteenPercent(t *testing.T) {
	without, _ := newMageWithDebuffs(t, 60, improvedScorchTalents(t), debuffsWithoutImprovedScorch())
	with, _ := newMageWithDebuffs(t, 60, improvedScorchTalents(t), core.FullBuffs.Debuffs)
	stepUntil(without, raidScorchRampUp)
	stepUntil(with, raidScorchRampUp)

	got := fireDamageTakenMultiplier(with.Encounter.TargetUnits[0]) /
		fireDamageTakenMultiplier(without.Encounter.TargetUnits[0])
	want := 1 + improvedScorchMaxStacks*improvedScorchPerStack
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("Fire damage taken ratio with five stacks = %v, want %v", got, want)
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
		castAndLand(sim, target, scorch)
	}
	if aura.GetStacks() != improvedScorchMaxStacks {
		t.Fatalf("after five Scorch casts the debuff has %d stacks, want %d", aura.GetStacks(), improvedScorchMaxStacks)
	}

	stepUntil(sim, sim.CurrentTime+20*time.Second)
	if got := aura.RemainingDuration(sim); got > 10*time.Second+time.Millisecond {
		t.Fatalf("20 s after the last cast %v remain, want about 10 s", got)
	}

	castAndLand(sim, target, scorch)
	if got := aura.GetStacks(); got != improvedScorchMaxStacks {
		t.Errorf("a sixth Scorch left %d stacks, want %d", got, improvedScorchMaxStacks)
	}
	if got := aura.RemainingDuration(sim); got < improvedScorchDuration-time.Second {
		t.Errorf("a sixth Scorch left %v on the debuff, want a renewed %v", got, improvedScorchDuration)
	}
}
