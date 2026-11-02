package hunter

import (
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
)

// Round 9 (physical lane): a level-20 hunter's best ammo is Sharp Arrow or
// Heavy Shot, 3.5 damage a second (Requires Level 10; Wowhead Forever item
// 2515 "Adds 3.5 damage per second", read 2026-11-02). The ammo enum had
// nothing below Razor Arrow's 7.5 (level 25), so the site gave a level-20
// hunter level-52 Thorium arrows.
func TestAmmoDamagePerSecondIsTheClients(t *testing.T) {
	for ammo, want := range map[proto.Hunter_Options_Ammo]float64{
		proto.Hunter_Options_AmmoNone:           0,
		proto.Hunter_Options_SharpArrow:         3.5,
		proto.Hunter_Options_HeavyShot:          3.5,
		proto.Hunter_Options_RazorArrow:         7.5,
		proto.Hunter_Options_SolidShot:          7.5,
		proto.Hunter_Options_JaggedArrow:        13,
		proto.Hunter_Options_AccurateSlugs:      13,
		proto.Hunter_Options_MithrilGyroShot:    15,
		proto.Hunter_Options_IceThreadedArrow:   16.5,
		proto.Hunter_Options_ThoriumHeadedArrow: 17.5,
	} {
		if got := ammoDPS(ammo); got != want {
			t.Errorf("%v: %v damage a second, want %v", ammo, got, want)
		}
	}
}
