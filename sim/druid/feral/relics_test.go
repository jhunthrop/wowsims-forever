package feral

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/druid"
)

func feralFingerprints(t *testing.T, relicID int32) map[string]core.SpellFingerprint {
	t.Helper()
	built, _, _ := newFeralDruidSimWearing(t, 60, feralTalentsString(t, map[string]int{"shifting_power": 1}), core.RelicEquipment(relicID))
	return core.SpellFingerprints(&built.Unit)
}

func changedByRelic(t *testing.T, relicID int32) []core.SpellChange {
	t.Helper()
	changed := core.ChangedSpells(feralFingerprints(t, 0), feralFingerprints(t, relicID))
	if len(changed) == 0 {
		t.Fatalf("relic %d changed no spell", relicID)
	}
	return changed
}

// Idol of the Dream (client spell 446212): Rip lasts 2 seconds longer, one
// more tick of its 2 second period.
func TestIdolOfTheDreamExtendsRipByOneTickOnly(t *testing.T) {
	for _, change := range changedByRelic(t, druid.IdolOfTheDream) {
		if change.After.ClassSpellMask&druid.DruidSpellMaskRip == 0 {
			t.Errorf("%s changed but is not Rip: %+v", change.Key, change.After)
		}
		if got := change.After.DotTicks - change.Before.DotTicks; got != 1 {
			t.Errorf("%s dot ticks moved by %d, want 1", change.Key, got)
		}
	}
}

// The extra tick must survive Rip's own per-cast tick count, which is the
// client's fixed six ticks (12 s) whatever the combo points: plus the idol's
// one.
func TestIdolOfTheDreamAddsATickToEveryRipCast(t *testing.T) {
	for _, tc := range []struct {
		relic int32
		want  time.Duration
	}{{0, 12 * time.Second}, {druid.IdolOfTheDream, 14 * time.Second}} {
		built, sim, target := newFeralDruidSimWearing(t, 60, P1Talents, core.RelicEquipment(tc.relic))
		built.AddComboPoints(sim, 5, target, built.NewComboPointMetrics(core.ActionID{SpellID: 1}))
		built.Rip.Cast(sim, target)
		dot := built.Rip.Dot(target)
		if !dot.IsActive() {
			t.Fatalf("relic %d: Rip did not apply", tc.relic)
		}
		if got := dot.Aura.Duration; got != tc.want {
			t.Errorf("relic %d: Rip at 5 combo points lasts %v, want %v", tc.relic, got, tc.want)
		}
	}
}

// Howling Idol (client spell 1291059): Shifting Power cooldown -1 second.
func TestHowlingIdolShortensShiftingPowerCooldownOnly(t *testing.T) {
	for _, change := range changedByRelic(t, druid.HowlingIdol) {
		if change.After.ClassSpellMask&druid.DruidSpellMaskShiftingPower == 0 {
			t.Errorf("%s changed but is not Shifting Power: %+v", change.Key, change.After)
		}
		if got := change.Before.CooldownDuration - change.After.CooldownDuration; got != 1*time.Second {
			t.Errorf("%s cooldown shortened by %v, want 1s", change.Key, got)
		}
	}
}
