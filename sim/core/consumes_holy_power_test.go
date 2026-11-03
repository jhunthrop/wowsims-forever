package core

import (
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// Forever turned Era's Elixir of Greater Firepower into the Elixir of Holy
// Power: item 21546 (ItemSparse "Elixir of Holy Power", level 40) uses
// spell 1310077, whose one effect is aura 13 (spell damage done) of 40 with
// misc value 2, the Holy school, for 30 minutes (duration index 30). The
// crafting spell 26277 is "Elixir of Holy Power" too, so the client carries
// no fire elixir of 40 any more (1.60.1.70753, hotfix push layered in).
func TestElixirOfHolyPowerMatchesTheClient(t *testing.T) {
	character := newGnomeEurekaTestCaster(proto.Class_ClassPaladin)
	applySpellBuffConsumes(character, &proto.Consumes{HolyPowerBuff: proto.HolyPowerBuff_ElixirOfHolyPower})
	if got := character.GetStat(stats.HolyPower); got != 40 {
		t.Errorf("Elixir of Holy Power gives %v Holy spell damage, want 40", got)
	}
	for _, other := range []stats.Stat{stats.SpellDamage, stats.FirePower, stats.HealingPower} {
		if got := character.GetStat(other); got != 0 {
			t.Errorf("Elixir of Holy Power gives %v of stat %v, want 0", got, other)
		}
	}
}

// The engine's Elixir of Greater Firepower is Era's +40 fire item. Item
// 21546 is the Holy elixir in Forever, so the old value applies nothing.
func TestElixirOfGreaterFirepowerIsGone(t *testing.T) {
	character := newGnomeEurekaTestCaster(proto.Class_ClassMage)
	applySpellBuffConsumes(character, &proto.Consumes{FirePowerBuff: proto.FirePowerBuff_ElixirOfGreaterFirepower})
	if got := character.GetStat(stats.FirePower); got != 0 {
		t.Errorf("Elixir of Greater Firepower gives %v fire spell damage, want 0 (Forever's 21546 is Holy)", got)
	}
}
