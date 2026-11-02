package restoration

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// Ankh of Life (1713), worn in published healer sets, has a use: ItemEffect
// 97284, spell 1317424 "Heal" for 720 with variance 0.2 (648 to 792) on
// the user's target, on a 5 min item cooldown.
func TestAnkhOfLifeHealsOnUse(t *testing.T) {
	player := newPlayer(60, StandardTalents, healerBonusStats, nil)
	player.Equipment = &proto.EquipmentSpec{Items: []*proto.ItemSpec{
		{}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {Id: 1713},
	}}
	resto, sim := newDruid(t, player)
	mcd := resto.GetMajorCooldown(core.ActionID{ItemID: 1713})
	if mcd == nil {
		t.Fatal("Ankh of Life registers no use")
	}
	resto.RemoveHealth(sim, resto.GetStat(stats.Health)/2)
	before := resto.CurrentHealth()
	mcd.Spell.Cast(sim, &resto.Unit)
	if got := resto.CurrentHealth() - before; got < 648 || got > 792*1.5 {
		t.Errorf("Ankh of Life healed %.0f, want 648 to 792 (or a crit of it)", got)
	}
	if got := mcd.Spell.CD.TimeToReady(sim); got != 5*time.Minute {
		t.Errorf("Ankh of Life's cooldown is %v, want 5m", got)
	}
}
