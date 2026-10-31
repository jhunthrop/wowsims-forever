package core

import (
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// The 15 October racial rework: Human Ruthless (20597) is crit with all
// spells and attacks whatever the weapon, and Dwarf Artisan's Precision
// (1259719) is hit with all spells and attacks. Neither table build the
// site holds has a weapon requirement on Ruthless (no SpellEquippedItems
// row); the current client text reads 1% for each (Wowhead's Forever
// tooltips, 2026-10-31), ahead of the 70658 tables (2%, and Dwarf still
// "Earth Fury").
func TestRuthlessAndArtisansPrecisionNeedNoWeapon(t *testing.T) {
	for _, c := range []struct {
		race proto.Race
		name string
		stat stats.Stat
	}{
		{proto.Race_RaceHuman, "Ruthless", stats.Crit},
		{proto.Race_RaceDwarf, "Artisan's Precision", stats.Hit},
	} {
		racial := racialByName(t, c.race, c.name)
		if racial.Kind != RacialPassive {
			t.Errorf("%s is %v, want a passive", c.name, racial.Kind)
		}
		character := &Character{}
		before := character.GetStat(c.stat)
		racial.Apply(character)
		if got := character.GetStat(c.stat) - before; got != 1 {
			t.Errorf("%s with no weapon grants %v of %v, want 1%%", c.name, got, c.stat)
		}
	}
}
