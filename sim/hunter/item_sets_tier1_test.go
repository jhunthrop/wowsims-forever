package hunter

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/common/clientsetbonus/clientsetbonustest"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

const (
	wildstalkerShotsRow int32 = 1301252 // 5P: Aimed Shot and Multi-Shot cooldown -1 s
)

func wildstalkerHunter(t *testing.T, pieces int) *Hunter {
	t.Helper()
	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassHunter,
			Race:               proto.Race_RaceOrc,
			Level:              60,
			Buffs:              core.FullBuffs.Player,
			DistanceFromTarget: 25,
		},
		P1PlayerOptions,
	)
	sim := clientsetbonustest.PrePulledSim(t, player, wildstalkerArmorSetID, pieces)
	agent, ok := sim.Raid.Parties[0].Players[0].(HunterAgent)
	if !ok {
		t.Fatal("the raid's first player is not a hunter agent")
	}
	return agent.GetHunter()
}

// 2P (haste) and 4P (attack power against Beasts) are flat rows.
func TestWildstalkerAutomaticBonusesMatchTheRows(t *testing.T) {
	bare := wildstalkerHunter(t, 0).GetCharacter()
	for _, pieces := range []int{2, 4} {
		clientsetbonustest.AssertAutomaticTotals(t, wildstalkerArmorSetID, pieces, bare, wildstalkerHunter(t, pieces).GetCharacter())
	}
}

// 5P: "Reduces the cooldown on your Aimed Shot and Multi-Shot abilities by
// 1 sec".
func TestWildstalkerFivePieceShortensAimedShotAndMultiShot(t *testing.T) {
	four := wildstalkerHunter(t, 4)
	five := wildstalkerHunter(t, 5)
	want := time.Duration(core.MustClientSpellRow(wildstalkerShotsRow).Effects[0].Points) * time.Millisecond

	cases := []struct {
		name       string
		four, five *core.Spell
	}{
		{"Aimed Shot", four.AimedShot, five.AimedShot},
		{"Multi-Shot", four.MultiShot, five.MultiShot},
	}
	for _, tc := range cases {
		if got := tc.five.CD.Duration - tc.four.CD.Duration; got != want {
			t.Errorf("five pieces change %s's cooldown by %v, the row says %v", tc.name, got, want)
		}
	}
	if four.ArcaneShot.CD.Duration != five.ArcaneShot.CD.Duration {
		t.Error("the bonus leaked onto Arcane Shot")
	}
}
