package shaman_test

import (
	"testing"

	"github.com/wowsims/classic/sim/common/clientsetbonus/clientsetbonustest"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/shaman"
	"github.com/wowsims/classic/sim/shaman/elemental"
)

const shockCritBonus = 22804

func init() { elemental.RegisterElementalShaman() }

func shamanWearing(t *testing.T, setID int32, pieces int) *shaman.Shaman {
	t.Helper()
	player := core.WithSpec(
		&proto.Player{Class: proto.Class_ClassShaman, Race: proto.Race_RaceTroll, Level: 60},
		&proto.Player_ElementalShaman{ElementalShaman: &proto.ElementalShaman{Options: &proto.ElementalShaman_Options{}}},
	)
	sim := clientsetbonustest.PrePulledSim(t, player, setID, pieces)
	return sim.Raid.Parties[0].Players[0].(shaman.ShamanAgent).GetShaman()
}

func bonusThreshold(t *testing.T, setID int32) int {
	t.Helper()
	row, _ := core.ClientSetRow(setID)
	for _, bonus := range row.Bonuses {
		if bonus.SpellID == shockCritBonus {
			return int(bonus.Threshold)
		}
	}
	t.Fatalf("set %d has no shock crit bonus", setID)
	return 0
}

// ranksOf is every rank the shaman has registered, from the families.
func ranksOf(families ...[]*core.Spell) []*core.Spell {
	var spells []*core.Spell
	for _, ranks := range families {
		for _, spell := range ranks {
			if spell != nil {
				spells = append(spells, spell)
			}
		}
	}
	return spells
}

func shocks(s *shaman.Shaman) []*core.Spell {
	return ranksOf(s.EarthShock, s.FlameShock, s.FrostShock)
}

func TestPvPShockCritBonusReachesEveryShockAndNothingElse(t *testing.T) {
	bonus := core.MustClientSpellRow(shockCritBonus).Effects[0].Points * core.CritRatingPerCritChance
	for _, setID := range []int32{538, 1757, 1758, 1759, 1731, 1732, 1733} {
		row, _ := core.ClientSetRow(setID)
		threshold := bonusThreshold(t, setID)
		bare := shamanWearing(t, setID, threshold-1)
		worn := shamanWearing(t, setID, threshold)

		bareShocks, wornShocks := shocks(bare), shocks(worn)
		if len(wornShocks) == 0 {
			t.Fatalf("%s: the shaman has no shock spells", row.Name)
		}
		for i, spell := range wornShocks {
			if got := spell.BonusCritRating - bareShocks[i].BonusCritRating; got != bonus {
				t.Errorf("%s: %d pieces add %v crit rating to spell %d, want %v", row.Name, threshold, got, spell.ActionID.SpellID, bonus)
			}
		}
		bareBolts, wornBolts := ranksOf(bare.LightningBolt), ranksOf(worn.LightningBolt)
		if len(wornBolts) == 0 {
			t.Fatalf("%s: the shaman has no Lightning Bolt", row.Name)
		}
		for i, spell := range wornBolts {
			if got := spell.BonusCritRating - bareBolts[i].BonusCritRating; got != 0 {
				t.Errorf("%s: Lightning Bolt gains %v crit rating", row.Name, got)
			}
		}
	}
}
