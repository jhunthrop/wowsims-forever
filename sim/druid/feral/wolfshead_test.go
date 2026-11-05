package feral

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/druid"
)

// Wolfshead Helm (8345, equip spell 17768): "You gain an additional 5 Rage
// from activating Enrage and an additional 5 Energy from activating Shifting
// Power." Blizzard's Forever notes of 8 October 2026: "Wolfshead Helm's
// effect has changed to granting 5 additional Energy when shifting power is
// used." The energy rides on Shifting Power (17768 effect 1, the server's
// energize 1310990 for 5), not on a shift into Cat Form.
func wolfsheadGear() *proto.EquipmentSpec {
	return &proto.EquipmentSpec{Items: []*proto.ItemSpec{{Id: druid.WolfsheadHelm}}}
}

func TestWolfsheadHelmAddsFiveEnergyToShiftingPower(t *testing.T) {
	talents := feralTalentsString(t, map[string]int{"shifting_power": 1})
	cat, sim, target := newFeralDruidSimWearing(t, 60, talents, wolfsheadGear())
	if head := cat.Equipment.Head(); head == nil || head.ID != druid.WolfsheadHelm {
		t.Fatal("the test druid is not wearing Wolfshead Helm")
	}
	cat.SpendEnergy(sim, cat.CurrentEnergy(), cat.NewEnergyMetrics(core.ActionID{SpellID: 1}))

	cat.ShiftingPower.Cast(sim, target)

	if got, want := cat.CurrentEnergy(), druid.ShiftingPowerEnergy+5; got != want {
		t.Errorf("Shifting Power with Wolfshead Helm left %v energy, want %v", got, want)
	}
}

func TestWolfsheadHelmGivesNoEnergyOnAShiftIntoCatForm(t *testing.T) {
	cat, sim, target := newFeralDruidSimWearing(t, 60, feralTalentsString(t, nil), wolfsheadGear())
	cat.CancelShapeshift(sim)
	cat.GCD.Reset()

	cat.CatForm.Cast(sim, target)

	if got := cat.CurrentEnergy(); got != 0 {
		t.Errorf("a shift into Cat Form with Wolfshead Helm and no Furor left %v energy, want 0", got)
	}
}
