package restoration

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
	"github.com/wowsims/classic/sim/healsim"
)

// Two Mana Spring Totems do not stack: the party member stands in one
// Mana Spring aura, the strongest. Under a raid that already carries the
// raid's Mana Spring Totem (25 mp5 at 60), a shaman's own plain totem adds
// nothing, and a Restorative Totems one only its excess over the raid's.
// The engine added the shaman's own totem on top of the raid buff, so an
// Elemental shaman (whose rotation drops Mana Spring) had 50 mp5.
func TestOwnManaSpringDoesNotStackWithTheRaids(t *testing.T) {
	raidMP5 := manaSpringTopRankRestore * manaSpringMP5PerRestore
	for name, tc := range map[string]struct {
		talents string
		want    float64
	}{
		"plain":                {"", 0},
		"Restorative Totems 5": {restoTalents(map[int]int{posRestorativeTotems: 5}), raidMP5*1.25 - raidMP5},
	} {
		req := healsim.Request(newHealerPlayer(60, tc.talents, nil), quietRaid(), 60, 1)
		req.Raid.Buffs = &proto.RaidBuffs{ManaSpringTotem: proto.TristateEffect_TristateEffectRegular}
		sim := core.NewSim(req, simsignals.CreateSignals())
		sim.Reset()
		sim.PrePull()
		healer := sim.Raid.Parties[0].Players[healsim.HealerIndex].(*RestorationShaman)

		before := healer.GetStat(stats.MP5)
		healer.ManaSpringTotem[4].Cast(sim, &healer.Unit)
		if got := healer.GetStat(stats.MP5) - before; math.Abs(got-tc.want) > 1e-6 {
			t.Errorf("%s: own Mana Spring under the raid's added %.3f mp5, want %.3f", name, got, tc.want)
		}
	}
}
