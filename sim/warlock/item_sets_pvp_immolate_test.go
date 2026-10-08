package warlock

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/common/clientsetbonus/clientsetbonustest"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

func immolateOf(t *testing.T, setID int32, pieces int) *core.Spell {
	t.Helper()
	player := core.WithSpec(
		&proto.Player{Class: proto.Class_ClassWarlock, Race: proto.Race_RaceOrc, Level: 60},
		&proto.Player_Warlock{Warlock: &proto.Warlock{Options: &proto.WarlockOptions{
			Armor:       proto.WarlockOptions_NoArmor,
			Summon:      proto.WarlockOptions_NoSummon,
			WeaponImbue: proto.WarlockOptions_NoWeaponImbue,
		}}},
	)
	sim := clientsetbonustest.PrePulledSim(t, player, setID, pieces)
	warlock := sim.Raid.Parties[0].Players[0].(WarlockAgent).GetWarlock()
	return warlock.Immolate[len(warlock.Immolate)-1]
}

// The 23047 row carries two modifiers, cast time and global cooldown, each
// -200 ms, although its text names only the cast time.
func TestPvPImmolateBonusShortensTheCastAndTheGlobalCooldown(t *testing.T) {
	reduction := time.Duration(core.MustClientSpellRow(immolateCastTimeBonus).Effects[0].Points) * time.Millisecond
	if reduction >= 0 {
		t.Fatalf("the row's reduction is %v, want negative", reduction)
	}
	for _, setID := range []int32{1760, 1774, 1734, 1746} {
		row, _ := core.ClientSetRow(setID)
		var threshold int
		for _, bonus := range row.Bonuses {
			if bonus.SpellID == immolateCastTimeBonus {
				threshold = int(bonus.Threshold)
			}
		}
		bare := immolateOf(t, setID, threshold-1)
		worn := immolateOf(t, setID, threshold)
		if got, want := worn.CastTime(), bare.CastTime()+reduction; got != want {
			t.Errorf("%s: cast time %v with %d pieces, want %v", row.Name, got, threshold, want)
		}
		if got, want := worn.DefaultCast.GCD, bare.DefaultCast.GCD+reduction; got != want {
			t.Errorf("%s: global cooldown %v with %d pieces, want %v", row.Name, got, threshold, want)
		}
		if bare.CastTime() != ImmolateCastTime {
			t.Errorf("%s: %d pieces change the cast time to %v", row.Name, threshold-1, bare.CastTime())
		}
	}
}
