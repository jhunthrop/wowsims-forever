package tank

import (
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/druid"
)

// Wolfshead Helm's rage half (equip spell 17768 effect 0: aura 108 on
// Enrage's class mask 0x80000, 50 tenths of a point): "You gain an
// additional 5 Rage from activating Enrage". Blizzard's 8 October notes:
// "Rage portion of the helm's effect remains the same."
func TestWolfsheadHelmAddsFiveRageToEnrage(t *testing.T) {
	gear := &proto.EquipmentSpec{Items: []*proto.ItemSpec{{Id: druid.WolfsheadHelm}}}
	bear, sim, target := newBearSimWearing(t, 60, nil, gear)
	rageBefore := bear.CurrentRage()

	bear.Enrage.Cast(sim, target)

	if got := bear.CurrentRage() - rageBefore; got != 15 {
		t.Errorf("Enrage with Wolfshead Helm gave %v instant rage, want 15", got)
	}
}
