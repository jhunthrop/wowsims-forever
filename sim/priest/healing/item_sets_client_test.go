package healing

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/common/clientsetbonus"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
	"github.com/wowsims/classic/sim/priest"
)

var (
	oracleSetID = priest.ItemSetGarmentsOfTheOracle.ID
	faithSetID  = priest.ItemSetVestmentsOfFaith.ID
)

const (
	oracleRenewPieces = 5
	faithRenewPieces  = 2
	faithEpiphany     = 8
	faithEpiphanyRow  = 28802
	faithEpiphanyBuff = 28804
	epiphanyProc      = "Epiphany (spellcast)"
	epiphanyTrials    = 20_000
)

func topRenew(t *testing.T, setID int32, talents string, pieces int) (*core.Simulation, *HealingPriest, *core.Spell) {
	t.Helper()
	sim, agent := wornPriest(t, setID, proto.Race_RaceHuman, talents, pieces)
	return sim, agent, topRank(agent.Renew)
}

// Garments of the Oracle 5P: "Increases the duration of your Renew spell by 3 sec."
func TestGarmentsOfTheOracleFivePieceLengthensRenew(t *testing.T) {
	ticks := func(pieces int) int32 {
		_, agent, renew := topRenew(t, oracleSetID, HolyTalents, pieces)
		return renew.Hot(&agent.GetCharacter().Unit).NumberOfTicks
	}
	_, agent, renew := topRenew(t, oracleSetID, HolyTalents, 0)
	tickLength := renew.Hot(&agent.GetCharacter().Unit).TickLength
	extra := time.Duration(core.MustClientSpellRow(clientsetbonus.SpellAt(oracleSetID, oracleRenewPieces)).Effects[0].Points) * time.Millisecond

	if fewer, bare := ticks(oracleRenewPieces-1), ticks(0); fewer != bare {
		t.Errorf("four pieces change Renew from %d to %d ticks", bare, fewer)
	}
	if got, want := ticks(oracleRenewPieces)-ticks(0), int32(extra/tickLength); got != want {
		t.Errorf("five pieces add %d Renew ticks, want %d (%v of %v ticks)", got, want, extra, tickLength)
	}
}

// Vestments of Faith 2P: "Reduces the mana cost of your Renew spell by 12%."
func TestVestmentsOfFaithTwoPieceCutsRenewCost(t *testing.T) {
	// With no talents: Mental Agility's cost cut adds to the set's.
	cost := func(pieces int) float64 {
		_, _, renew := topRenew(t, faithSetID, "", pieces)
		return renew.Cost.GetCurrentCost()
	}
	bare := cost(0)
	if one := cost(faithRenewPieces - 1); one != bare {
		t.Errorf("one piece changes Renew's cost from %v to %v", bare, one)
	}
	cut := core.MustClientSpellRow(clientsetbonus.SpellAt(faithSetID, faithRenewPieces)).Effects[0].Points / 100
	if got, want := cost(faithRenewPieces), bare*(1+cut); !nearlyEqual(got, want) {
		t.Errorf("two pieces cost %v, want %v (%v %+v%%)", got, want, bare, cut*100)
	}
}

func nearlyEqual(a, b float64) bool {
	const epsilon = 1e-6
	return a-b < epsilon && b-a < epsilon
}

// Vestments of Faith 8P: each spell can trigger an Epiphany that raises
// mana regeneration for the client's duration.
func TestVestmentsOfFaithEightPieceEpiphanyRaisesMp5(t *testing.T) {
	_, seven := wornPriest(t, faithSetID, proto.Race_RaceHuman, HolyTalents, faithEpiphany-1)
	if seven.GetCharacter().GetAura(epiphanyProc) != nil {
		t.Fatal("seven pieces already carry Epiphany")
	}
	sim, agent := wornPriest(t, faithSetID, proto.Race_RaceHuman, HolyTalents, faithEpiphany)
	character := agent.GetCharacter()
	proc := character.GetAura(epiphanyProc)
	if proc == nil {
		t.Fatal("eight pieces do not carry Epiphany")
	}
	buffRow := core.MustClientSpellRow(faithEpiphanyBuff)
	chance := float64(core.MustClientSpellRow(faithEpiphanyRow).ProcChance) / 100

	before := character.GetStat(stats.MP5)
	procs := 0
	for i := 0; i < epiphanyTrials; i++ {
		buff := character.GetAura("Epiphany")
		if buff.IsActive() {
			buff.Deactivate(sim)
		}
		proc.OnCastComplete(proc, sim, &core.Spell{ProcMask: core.ProcMaskSpellHealing})
		if buff.IsActive() {
			procs++
			if got := character.GetStat(stats.MP5) - before; got != buffRow.Effects[0].Points {
				t.Fatalf("Epiphany adds %v mp5, want the client's %v", got, buffRow.Effects[0].Points)
			}
			if buff.Duration != time.Duration(buffRow.DurationMS)*time.Millisecond {
				t.Fatalf("Epiphany lasts %v, want %dms", buff.Duration, buffRow.DurationMS)
			}
		}
	}
	if want := chance * epiphanyTrials; float64(procs) < want*0.85 || float64(procs) > want*1.15 {
		t.Errorf("%d Epiphanies in %d heals, want about %.0f", procs, epiphanyTrials, want)
	}
}
