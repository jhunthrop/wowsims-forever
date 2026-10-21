package clientsetbonustest

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/common/clientsetbonus"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// PrePulledSim builds a one-player, one-boss sim around player wearing the
// first pieces pieces of a client set (synthetic zero-stat items, so only
// the set's bonuses change the character), reset and pre-pulled. The player
// needs its class, race, level, spec and options; its equipment and
// database are replaced.
func PrePulledSim(t *testing.T, player *proto.Player, setID int32, pieces int) *core.Simulation {
	t.Helper()
	Wear(player, setID, pieces)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 60, Targets: []*proto.Target{core.DefaultTargetProtoLvl60}},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()
	return sim
}

// Wear puts the first pieces pieces of a client set on player (synthetic
// zero-stat items) and returns it, for a test that builds its own sim.
func Wear(player *proto.Player, setID int32, pieces int) *proto.Player {
	player.Equipment, player.Database = core.ClientSetTestGear(setID, pieces)
	return player
}

// automaticPieces are the piece counts whose flat bonuses a set test checks.
var automaticPieces = []int{2, 4}

// AssertAutomaticTotalsAtTwoAndFour wears two and four pieces of the set
// and checks the flat bonuses against the rows. build returns the
// character wearing that many pieces.
func AssertAutomaticTotalsAtTwoAndFour(t *testing.T, setID int32, build func(pieces int) *core.Character) {
	t.Helper()
	bare := build(0)
	for _, pieces := range automaticPieces {
		AssertAutomaticTotals(t, setID, pieces, bare, build(pieces))
	}
}

// AssertCooldownBonus checks a cooldown bonus of the set against its row:
// one piece short of the threshold the spell's cooldown is the bare one,
// and at the threshold it moves by the row's amount (floored at zero).
// spellAt returns the spell on a character wearing that many pieces.
func AssertCooldownBonus(t *testing.T, setID, threshold int32, spellAt func(pieces int) *core.Spell) {
	t.Helper()
	change := time.Duration(core.MustClientSpellRow(clientsetbonus.SpellAt(setID, threshold)).Effects[0].Points) * time.Millisecond
	bare := spellAt(0).CD.Duration
	if short := spellAt(int(threshold) - 1).CD.Duration; short != bare {
		t.Errorf("%d pieces of set %d change the cooldown from %v to %v; the bonus starts at %d", threshold-1, setID, bare, short, threshold)
	}
	want := max(0, bare+change)
	if got := spellAt(int(threshold)).CD.Duration; got != want {
		t.Errorf("%d pieces of set %d give a cooldown of %v, want %v (%v %+v)", threshold, setID, got, want, bare, change)
	}
}

// SpellWithMask is the first spell of the unit whose class spell mask
// includes mask: a test's way to find a spell the class does not export.
func SpellWithMask(t *testing.T, unit *core.Unit, mask uint64) *core.Spell {
	t.Helper()
	for _, spell := range unit.Spellbook {
		if spell.ClassSpellMask&mask != 0 {
			return spell
		}
	}
	t.Fatalf("no spell with class mask %d is registered", mask)
	return nil
}
