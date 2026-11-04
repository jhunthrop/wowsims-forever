package rogue

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
)

// TestShankRanksFollowTheClient pins Shank's two ranks: 1329283 at 24 with
// 15 Energy, 1329549 at 44 with 25 (SpellLevels and e1 base points).
func TestShankRanksFollowTheClient(t *testing.T) {
	for level, want := range map[int32]struct {
		id     int32
		energy float64
	}{23: {0, 0}, 24: {1329283, 15}, 43: {1329283, 15}, 44: {1329549, 25}, 60: {1329549, 25}} {
		rank := core.HighestRankAtLevel(shankLearnLevels, level)
		if got := shankSpellID[rank]; got != want.id {
			t.Errorf("Shank at %d = %d, want %d", level, got, want.id)
		}
		if rank > 0 && shankEnergy(rank, 0) != want.energy {
			t.Errorf("Shank at %d energizes %v, want %v", level, shankEnergy(rank, 0), want.energy)
		}
	}
}

// TestFlashOfSteelAndBootKnifeCurves pins the 70755 trait curves per rank.
func TestFlashOfSteelAndBootKnifeCurves(t *testing.T) {
	for rank := int32(0); rank <= 5; rank++ {
		if got, want := shankEnergy(2, rank), 25+3*float64(rank); got != want {
			t.Errorf("Flash of Steel %d: Shank Energy %v, want %v", rank, got, want)
		}
		if got, want := shankCooldownWith(rank), 10*time.Second-time.Duration(rank)*400*time.Millisecond; got != want {
			t.Errorf("Flash of Steel %d: Shank cooldown %v, want %v", rank, got, want)
		}
		if got, want := bootKnifeChance(rank), 0.2*float64(rank); got != want {
			t.Errorf("Boot Knife %d: chance %v, want %v", rank, got, want)
		}
	}
}
