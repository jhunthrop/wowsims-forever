package mage

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
)

// highestRegistered is the top rank of spells a level-60 mage trains; the
// generated rank tables can run past what the level learns.
func highestRegistered(t *testing.T, name string, spells []*core.Spell) *core.Spell {
	t.Helper()
	for rank := len(spells) - 1; rank >= 1; rank-- {
		if spells[rank] != nil {
			return spells[rank]
		}
	}
	t.Fatalf("%s has no registered rank", name)
	return nil
}

// Ice Barrier since client 1.60.1.70532: +3% Frost damage for a minute
// from the cast and another +3% while the shield holds, which in a sim
// is the whole minute.
func TestIceBarrierRaisesFrostDamageForAMinute(t *testing.T) {
	built, sim, _ := newFrostMageSimWithTalents(t, bossTarget(), ForeverFrostTalents)
	barrier := highestRegistered(t, "Ice Barrier", built.IceBarrier)
	frostbolt := highestRegistered(t, "Frostbolt", built.Frostbolt)
	fireball := highestRegistered(t, "Fireball", built.Fireball)
	frostBefore := frostbolt.DamageMultiplierAdditive
	fireBefore := fireball.DamageMultiplierAdditive

	barrier.ApplyEffects(sim, &built.Unit, barrier)
	want := float64(iceBarrierFrostDamagePct+iceBarrierShieldedFrostDamagePct) / 100
	if got := frostbolt.DamageMultiplierAdditive - frostBefore; got < want-1e-9 || got > want+1e-9 {
		t.Errorf("Ice Barrier raised Frostbolt's additive damage by %.4f, want %.4f", got, want)
	}
	if got := fireball.DamageMultiplierAdditive; got != fireBefore {
		t.Errorf("Ice Barrier changed Fireball's additive damage to %.4f from %.4f; it is Frost damage only", got, fireBefore)
	}

	// A recast replaces the shield rather than stacking a second one.
	barrier.ApplyEffects(sim, &built.Unit, barrier)
	if got := frostbolt.DamageMultiplierAdditive - frostBefore; got < want-1e-9 || got > want+1e-9 {
		t.Errorf("a second Ice Barrier left Frostbolt's additive damage %.4f up, want %.4f", got, want)
	}

	advanceBy(sim, iceBarrierDuration+time.Second)
	if got := frostbolt.DamageMultiplierAdditive; got != frostBefore {
		t.Errorf("Frostbolt's additive damage is %.4f a minute after Ice Barrier, want %.4f: both halves lapse", got, frostBefore)
	}
}
