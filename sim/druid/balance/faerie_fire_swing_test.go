package balance

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// Blizzard's Forever notes of 1 October 2026: "Faerie Fire no longer resets
// your swing timer when used."
func TestFaerieFireDoesNotResetTheSwingTimer(t *testing.T) {
	built, _, _ := newBalanceDruidSimInRaid(t, druidTalentsString(t, nil), &proto.EquipmentSpec{}, &proto.RaidBuffs{})
	if built.FaerieFire == nil {
		t.Fatal("no caster Faerie Fire registered")
	}
	if built.FaerieFire.Flags.Matches(core.SpellFlagResetAttackSwing) {
		t.Error("caster Faerie Fire resets the swing timer")
	}
}
