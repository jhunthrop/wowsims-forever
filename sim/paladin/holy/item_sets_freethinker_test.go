package holy

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/common/clientsetbonus/clientsetbonustest"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/healsim"
	"github.com/wowsims/classic/sim/paladin"
)

const (
	freethinkersArmorSetID int32 = 475
	freethinkersHolyLight  int32 = 24457
	freethinkersPieces           = 3
)

// freethinkersPaladin is a holy paladin wearing pieces of Freethinker's Armor.
func freethinkersPaladin(t *testing.T, pieces int) *paladin.Paladin {
	t.Helper()
	player := fight{}.player(t)
	player.Race = proto.Race_RaceHuman
	clientsetbonustest.Wear(player, freethinkersArmorSetID, pieces)
	sim := core.NewSim(healsim.Request(player, idleRaid(), defaultSeconds, 1), simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()
	agent, ok := sim.Raid.Parties[0].Players[0].(*HolyPaladin)
	if !ok {
		t.Fatal("the raid's first player is not a holy paladin")
	}
	return agent.GetPaladin()
}

// Freethinker's Armor 3P: "Reduces the casting time of your Holy Light spell by 0.1 sec."
func TestFreethinkersArmorThreePieceShortensHolyLight(t *testing.T) {
	castTime := func(pieces int) time.Duration {
		pal := freethinkersPaladin(t, pieces)
		return clientsetbonustest.SpellWithMask(t, &pal.Unit, paladin.PaladinSpellMaskHolyLight).DefaultCast.CastTime
	}
	change := time.Duration(core.MustClientSpellRow(freethinkersHolyLight).Effects[0].Points) * time.Millisecond
	bare := castTime(0)
	if two := castTime(freethinkersPieces - 1); two != bare {
		t.Errorf("two pieces change Holy Light from %v to %v", bare, two)
	}
	if got := castTime(freethinkersPieces); got != bare+change {
		t.Errorf("three pieces cast Holy Light in %v, want %v (%v %+v)", got, bare+change, bare, change)
	}
}
