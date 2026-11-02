package sim

import (
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// Blessing of Wisdom (a paladin's blessing, 19854/25290) and Mana Spring
// Totem (a shaman's totem, 10497 casting 10494) are different spells from
// different casters and both land on the same player. Forever fields both
// classes on both factions, so a raid with both gives both.
func TestBlessingOfWisdomAndManaSpringBothApply(t *testing.T) {
	mp5 := func(wisdom bool, spring bool) float64 {
		player := priestPlayer(60)
		if wisdom {
			player.Buffs = &proto.IndividualBuffs{BlessingOfWisdom: proto.TristateEffect_TristateEffectRegular}
		}
		raidBuffs := &proto.RaidBuffs{}
		if spring {
			raidBuffs.ManaSpringTotem = proto.TristateEffect_TristateEffectRegular
		}
		_, final := firstPlayerStats(t, raidBuffs, player)
		return final[stats.MP5]
	}
	none, wisdom, spring, both := mp5(false, false), mp5(true, false), mp5(false, true), mp5(true, true)
	if want := none + (wisdom - none) + (spring - none); both != want {
		t.Errorf("MP5 with both is %.1f, want %.1f (Blessing of Wisdom +%.1f and Mana Spring +%.1f)", both, want, wisdom-none, spring-none)
	}
	if spring == none {
		t.Error("Mana Spring alone added no MP5")
	}
}
