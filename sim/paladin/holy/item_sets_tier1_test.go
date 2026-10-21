package holy

import (
	"testing"

	"github.com/wowsims/classic/sim/common/clientsetbonus"
	"github.com/wowsims/classic/sim/common/clientsetbonus/clientsetbonustest"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/healsim"
	"github.com/wowsims/classic/sim/paladin"
)

var justiceArmorSetID = paladin.ItemSetJusticeArmor.ID

// justiceArmorPaladin is a holy paladin wearing pieces of Justice Armor.
func justiceArmorPaladin(t *testing.T, race proto.Race, talents map[string]int, pieces int) *paladin.Paladin {
	t.Helper()
	player := fight{talents: talents}.player(t)
	player.Race = race
	clientsetbonustest.Wear(player, justiceArmorSetID, pieces)
	sim := core.NewSim(healsim.Request(player, idleRaid(), defaultSeconds, 1), simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()
	agent, ok := sim.Raid.Parties[0].Players[0].(*HolyPaladin)
	if !ok {
		t.Fatal("the raid's first player is not a holy paladin")
	}
	return agent.GetPaladin()
}

// The flat bonuses are read on a Dwarf with no talents, so no racial or
// talent multiplier scales the set's amounts.
func TestJusticeArmorFlatBonusesMatchTheRows(t *testing.T) {
	clientsetbonustest.AssertAutomaticTotalsAtTwoAndFour(t, justiceArmorSetID, func(pieces int) *core.Character {
		return &justiceArmorPaladin(t, proto.Race_RaceDwarf, nil, pieces).Character
	})
}

// 5P: "Reduces the cooldown on your Holy Shock spell by 1 sec".
func TestJusticeArmorFivePieceShortensHolyShock(t *testing.T) {
	clientsetbonustest.AssertCooldownBonus(t, justiceArmorSetID, clientsetbonus.FivePieces, func(pieces int) *core.Spell {
		pal := justiceArmorPaladin(t, proto.Race_RaceHuman, map[string]int{"holy_shock": 1}, pieces)
		return clientsetbonustest.SpellWithMask(t, &pal.Unit, paladin.PaladinSpellMaskHolyShock)
	})
}
