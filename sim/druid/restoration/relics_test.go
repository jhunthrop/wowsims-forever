package restoration

import (
	"testing"
	"time"

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

// Idol of Synthesis 272429 (Pix Xizzix, client 1.60.1.70790 items/druid.json):
// "Each of your heal over time effects on the target reduces the remaining
// cooldown on Swiftmend by 1 sec when you cast it." Its equip spell is a
// server-scripted dummy, so the engine models the text by hand; an
// unmodelled relic leaves the gear search an empty idol slot at 60 before
// the raids open.
func TestIdolOfSynthesisCutsSwiftmendsRemainingCooldownPerHoT(t *testing.T) {
	talents := talentsString(t, map[string]int{"swiftmend": 1})
	remaining := func(relic int32, hots int) time.Duration {
		player := newPlayer(60, talents, stats.Stats{}, nil)
		player.Equipment = core.RelicEquipment(relic)
		resto, sim := newDruid(t, player)
		self := &resto.Unit
		if hots >= 1 {
			resto.Rejuvenation[10].ApplyEffects(sim, self, resto.Rejuvenation[10].Spell)
		}
		if hots >= 2 {
			resto.Regrowth[9].ApplyEffects(sim, self, resto.Regrowth[9].Spell)
		}
		if !resto.Swiftmend.Cast(sim, self) {
			t.Fatal("Swiftmend did not cast")
		}
		return resto.Swiftmend.CD.TimeToReady(sim)
	}
	if got := remaining(0, 2); got != 15*time.Second {
		t.Errorf("without the idol Swiftmend is ready again in %v, want 15s", got)
	}
	if got := remaining(druid.IdolOfSynthesis, 1); got != 14*time.Second {
		t.Errorf("with the idol and one effect up Swiftmend is ready again in %v, want 14s", got)
	}
	if got := remaining(druid.IdolOfSynthesis, 2); got != 13*time.Second {
		t.Errorf("with the idol and both effects up Swiftmend is ready again in %v, want 13s", got)
	}
}
