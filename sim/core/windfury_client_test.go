package core

import (
	"testing"
	"time"
)

// Since push 113887 the rank's attack power is the totem aura's dummy
// (8515, 10609, 10612 effect 1: 95, 179, 246) and the rank's spell (8516,
// 10608, 10610) is the strike. Vanilla Classic's 122/229/315 is not this
// build's.
func TestWindfuryTotemStrikeMatchesClient(t *testing.T) {
	if want := [WindfuryRanks + 1]float64{0, 95, 179, 246}; WindfuryBonusAP != want {
		t.Errorf("Windfury Totem attack power by rank = %v, want %v", WindfuryBonusAP, want)
	}
	if want := [WindfuryRanks + 1]int32{0, 8516, 10608, 10610}; WindfuryStrikeSpellId != want {
		t.Errorf("Windfury Totem strike ids = %v, want %v", WindfuryStrikeSpellId, want)
	}
}

// The totem's proc aura states ProcChance 20 and a 100 ms
// ProcCategoryRecovery (spells 8515 and 10612 in SpellAuraOptions); the
// engine's 1.5 s was vanilla Classic's.
func TestWindfuryTotemProcMatchesClient(t *testing.T) {
	if windfuryProcChance != 0.2 {
		t.Errorf("Windfury Totem proc chance = %v, want 0.2", windfuryProcChance)
	}
	if windfuryProcICD != 100*time.Millisecond {
		t.Errorf("Windfury Totem proc cooldown = %v, want 100ms", windfuryProcICD)
	}
}
