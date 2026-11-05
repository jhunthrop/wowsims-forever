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

// Round 23 (items): Ankh of Life's ItemEffect row puts the use in category
// 1153 with a 2 min category cooldown, the category of the healthstones,
// Demonic and Dark Rune, the mana gems and Thistle Tea (their ItemEffect
// rows). Using the Ankh holds a Demonic Rune for 2 minutes, and a rune
// holds the Ankh.
func TestAnkhOfLifeSharesTheConjuredCategory(t *testing.T) {
	player := newPlayer(60, StandardTalents, healerBonusStats, nil)
	player.Equipment = &proto.EquipmentSpec{Items: []*proto.ItemSpec{
		{}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {Id: 1713},
	}}
	resto, sim := newDruid(t, player)
	ankh := resto.GetMajorCooldown(core.ActionID{ItemID: 1713}).Spell
	resto.RemoveHealth(sim, resto.GetStat(stats.Health)/2)
	ankh.Cast(sim, &resto.Unit)
	if got := resto.GetConjuredCD().TimeToReady(sim); got != 2*time.Minute {
		t.Errorf("the conjured items are ready %v after the Ankh, want 2m (category 1153)", got)
	}

	ankh.CD.Reset()
	resto.GetConjuredCD().Set(sim.CurrentTime + 2*time.Minute)
	if ankh.CanCast(sim, &resto.Unit) {
		t.Error("Ankh of Life castable while a rune holds category 1153")
	}
}
