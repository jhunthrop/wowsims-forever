package core

import (
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// The mana oils state their mana per five seconds and healing in the
// client's item tooltips (20745, 20747, 20748) and in the enchant spells
// those apply (25114, 25115, 25116: aura 85 and aura 135). Vanilla's 4, 8
// and 12 mana with 0, 0 and 25 healing are not this build's.
func TestManaOilsMatchTheClient(t *testing.T) {
	want := map[proto.WeaponImbue]stats.Stats{
		proto.WeaponImbue_MinorManaOil:     {stats.MP5: 5, stats.HealingPower: 10},
		proto.WeaponImbue_LesserManaOil:    {stats.MP5: 10, stats.HealingPower: 20},
		proto.WeaponImbue_BrilliantManaOil: {stats.MP5: 15, stats.HealingPower: 30},
	}
	for imbue, expected := range want {
		if got := manaOilStats[imbue]; got != expected {
			t.Errorf("%v gives %v, want %v", imbue, got, expected)
		}
	}
}

// Nightfin Soup's well-fed buff is 22 spell damage (spell 1249513 states
// it through an aura 227 effect of 22); the client gives it no mana.
func TestNightfinSoupMatchesTheClient(t *testing.T) {
	want := stats.Stats{stats.SpellDamage: 22}
	if nightfinSoupStats != want {
		t.Errorf("Nightfin Soup gives %v, want %v", nightfinSoupStats, want)
	}
	if nightfinSoupStats[stats.MP5] != 0 {
		t.Error("Nightfin Soup must give no mana per five seconds")
	}
}
