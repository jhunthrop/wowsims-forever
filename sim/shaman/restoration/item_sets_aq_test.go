package restoration

import (
	"math"
	"testing"
	"time"

	"github.com/wowsims/classic/sim/common/clientsetbonus"
	"github.com/wowsims/classic/sim/common/clientsetbonus/clientsetbonustest"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
	"github.com/wowsims/classic/sim/healsim"
)

// The client ItemSets of the Tier 2.5 and Tier 3 shaman sets under test.
const (
	stormcallersGarbSetID   int32 = 501
	earthshattererSetID     int32 = 527
	stormcallerChainHealPcs       = 5
	earthshattererSpringPcs       = 4
	stormcallerChainHealRow int32 = 26122
	earthshattererSpringRow int32 = 29171
)

func wornShaman(t *testing.T, setID int32, pieces int) (*core.Simulation, *RestorationShaman) {
	t.Helper()
	player := newHealerPlayer(60, "", nil)
	clientsetbonustest.Wear(player, setID, pieces)
	sim := core.NewSim(healsim.Request(player, quietRaid(), 60, 1), simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()
	healer, ok := sim.Raid.Parties[0].Players[healsim.HealerIndex].(*RestorationShaman)
	if !ok {
		t.Fatal("the raid's first player is not a Restoration shaman")
	}
	return sim, healer
}

// Stormcaller's Garb 5P: "-0.4 seconds on the casting time of your Chain Heal spell."
func TestStormcallersGarbFivePieceShortensChainHeal(t *testing.T) {
	castTime := func(pieces int) time.Duration {
		_, healer := wornShaman(t, stormcallersGarbSetID, pieces)
		return healer.ChainHeal[len(healer.ChainHeal)-1].DefaultCast.CastTime
	}
	change := time.Duration(core.MustClientSpellRow(stormcallerChainHealRow).Effects[0].Points) * time.Millisecond
	bare := castTime(0)
	if four := castTime(stormcallerChainHealPcs - 1); four != bare {
		t.Errorf("four pieces change Chain Heal from %v to %v", bare, four)
	}
	if got := castTime(stormcallerChainHealPcs); got != bare+change {
		t.Errorf("five pieces cast Chain Heal in %v, want %v (%v %+v)", got, bare+change, bare, change)
	}
}

// The Earthshatterer 4P: "Increases the mana gained from your Mana Spring totems by 25%."
func TestEarthshattererFourPieceRaisesManaSpring(t *testing.T) {
	gained := func(pieces int) float64 {
		sim, healer := wornShaman(t, earthshattererSetID, pieces)
		before := healer.GetStat(stats.MP5)
		healer.ManaSpringTotem[4].Cast(sim, &healer.Unit)
		return healer.GetStat(stats.MP5) - before
	}
	bare := gained(0)
	if three := gained(earthshattererSpringPcs - 1); math.Abs(three-bare) > 1e-9 {
		t.Errorf("three pieces change Mana Spring from %v to %v mp5", bare, three)
	}
	rise := clientsetbonus.PercentModifier(earthshattererSpringRow, clientsetbonus.ModOpAllEffects)
	if got, want := gained(earthshattererSpringPcs), bare*(1+rise); math.Abs(got-want) > 1e-6 {
		t.Errorf("four pieces give Mana Spring %v mp5, want %v (%v +%v%%)", got, want, bare, rise*100)
	}
}
