package mage

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
)

func frostbiteTalents(t *testing.T, rank int) string {
	t.Helper()
	return talentStringWithRank(t, ForeverFrostTalents, "frostbite", rank)
}

// Frostbite 11071: "Gives your Chill effects a 5/10/15% chance to Freeze
// the target for 5 sec" (trigger spell 12494, a root of duration index
// 28 = 5000 ms).
func TestFrostbiteChanceFollowsTheTalentRank(t *testing.T) {
	for rank, want := range map[int]float64{1: 0.05, 2: 0.10, 3: 0.15} {
		built, sim, target := newFrostMageSimWithTalents(t, freezableTarget(), frostbiteTalents(t, rank))
		const rolls = 6000
		frozen := 0
		for i := 0; i < rolls; i++ {
			built.FrostbiteFrozenAuras.Get(target).Deactivate(sim)
			built.rollFrostbite(sim, target)
			if built.FrostbiteFrozenAuras.Get(target).IsActive() {
				frozen++
			}
		}
		rate := float64(frozen) / rolls
		if rate < want-0.02 || rate > want+0.02 {
			t.Errorf("rank %d: freeze rate %.3f, want about %.2f", rank, rate, want)
		}
	}
}

func TestFrostbiteIsAbsentWithoutTheTalent(t *testing.T) {
	built, _, _ := newFrostMageSimWithTalents(t, freezableTarget(), frostbiteTalents(t, 0))
	if built.FrostbiteFrozenAuras != nil {
		t.Error("Frostbite registered without the talent")
	}
}

// A raid boss is immune to the root (canFreeze), as it is to Frost Nova's.
func TestFrostbiteCannotFreezeARaidBoss(t *testing.T) {
	built, sim, boss := newFrostMageSimWithTalents(t, bossTarget(), frostbiteTalents(t, 3))
	for i := 0; i < 500; i++ {
		built.rollFrostbite(sim, boss)
	}
	if built.FrostbiteFrozenAuras.Get(boss).IsActive() || built.isTargetFrozen(boss) {
		t.Error("a raid boss was frozen by Frostbite")
	}
}

func TestFrostbiteFreezeLastsFiveSecondsAndCountsAsFrozen(t *testing.T) {
	built, sim, target := newFrostMageSimWithTalents(t, freezableTarget(), frostbiteTalents(t, 3))
	aura := built.FrostbiteFrozenAuras.Get(target)
	if aura.Duration != 5*time.Second {
		t.Errorf("freeze lasts %v, want 5s", aura.Duration)
	}
	if built.isTargetFrozen(target) {
		t.Fatal("target reads Frozen before any freeze")
	}
	aura.Activate(sim)
	if !built.isTargetFrozen(target) {
		t.Error("a Frostbite freeze does not count as Frozen for Shatter and Ice Lance")
	}
}

func TestFrostbiteProcsFromChillSpells(t *testing.T) {
	for name, pick := range map[string]func(*Mage) *core.Spell{
		"Frostbolt":      func(m *Mage) *core.Spell { return m.Frostbolt[FrostboltRanks-1] },
		"Frostfire Bolt": func(m *Mage) *core.Spell { return m.FrostfireBolt[FrostfireBoltRanks] },
		"Cone of Cold":   func(m *Mage) *core.Spell { return m.ConeOfCold[ConeOfColdRanks] },
	} {
		built, sim, target := newFrostMageSimWithTalents(t, freezableTarget(), frostbiteTalents(t, 3))
		spell := pick(built)
		frozen := false
		for i := 0; i < 60 && !frozen; i++ {
			spell.ApplyEffects(sim, target, spell)
			waitForOutcomes(sim)
			frozen = built.FrostbiteFrozenAuras.Get(target).IsActive()
		}
		if !frozen {
			t.Errorf("60 %s casts never froze a freezable target", name)
		}
	}
}

func TestFrostNovaDoesNotRollFrostbite(t *testing.T) {
	built, sim, target := newFrostMageSimWithTalents(t, freezableTarget(), frostbiteTalents(t, 3))
	nova := built.FrostNova[FrostNovaRanks]
	for i := 0; i < 40; i++ {
		built.FrostbiteFrozenAuras.Get(target).Deactivate(sim)
		nova.ApplyEffects(sim, target, nova)
		waitForOutcomes(sim)
		if built.FrostbiteFrozenAuras.Get(target).IsActive() {
			t.Fatal("Frost Nova is not a Chill effect, so it must not roll Frostbite")
		}
	}
}
