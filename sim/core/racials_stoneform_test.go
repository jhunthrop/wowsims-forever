package core

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core/proto"
)

// Stoneform (20594, 70755 client and Wowhead Forever, read 2026-11-04):
// 10% less Physical damage taken (aura 87, -10, school mask 1) for 8 s
// (DurationIndex 31), a 3 minute cooldown, and the global cooldown
// (SpellCooldowns StartRecoveryTime 1500, StartRecoveryCategory 133).
func TestStoneformMatchesTheClientRow(t *testing.T) {
	dwarf, _ := racialPlayer(t, proto.Race_RaceDwarf, proto.MobType_MobTypeUnknown)
	stoneform := dwarf.GetSpell(ActionID{SpellID: 20594})
	if stoneform == nil {
		t.Fatal("Stoneform is not registered under 20594")
	}
	if stoneform.CD.Duration != 3*time.Minute {
		t.Errorf("Stoneform cooldown is %v, want 3m", stoneform.CD.Duration)
	}
	if stoneform.DefaultCast.GCD != GCDDefault {
		t.Errorf("Stoneform global cooldown is %v, want %v", stoneform.DefaultCast.GCD, GCDDefault)
	}
	if aura := dwarf.GetAura("Stoneform"); aura == nil || aura.Duration != 8*time.Second {
		t.Errorf("Stoneform aura = %v, want an 8 s aura", aura)
	}
}
