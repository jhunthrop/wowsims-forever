package restoration

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
	"github.com/wowsims/classic/sim/druid"
)

// Idol of Swiftness 279250 (client spell 1306488 "Swiftmend Cooldown
// Reduction", aura 107 op 11, -3000 ms, family word 1 0x2, which is
// Swiftmend's SpellClassOptions mask): "Reduces the cooldown of your
// Swiftmend spell by 3 sec". It is an Enchanting relic, and an unregistered
// effect leaves the gear search an empty idol slot.
func TestIdolOfSwiftnessCutsSwiftmendsCooldownOnly(t *testing.T) {
	talents := talentsString(t, map[string]int{"swiftmend": 1})
	fingerprints := func(relic int32) map[string]core.SpellFingerprint {
		player := newPlayer(60, talents, stats.Stats{}, nil)
		player.Equipment = core.RelicEquipment(relic)
		resto, _ := newDruid(t, player)
		return core.SpellFingerprints(&resto.Unit)
	}
	changed := core.ChangedSpells(fingerprints(0), fingerprints(druid.IdolOfSwiftness))
	if len(changed) == 0 {
		t.Fatal("Idol of Swiftness changed no spell")
	}
	for _, change := range changed {
		if change.After.ClassSpellMask&druid.DruidSpellMaskSwiftmend == 0 {
			t.Errorf("%s changed but is not Swiftmend: %+v", change.Key, change.After)
		}
		if got := change.Before.CooldownDuration - change.After.CooldownDuration; got.Seconds() != 3 {
			t.Errorf("%s cooldown moved by %v, want 3s", change.Key, got)
		}
	}
}
