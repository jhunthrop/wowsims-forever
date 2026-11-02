package item_sets

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

// The client's SpellAuraOptions.ProcTypeMask says which melee attacks a
// bonus procs on: 0x4 is the wearer's melee autoattack, 0x10 a melee
// special. A bonus whose mask has 0x4 alone (Devious Strike, Rogue Armor
// Energize, Valor's rage, Bloodmail) never procs on Shred, Claw or Sinister
// Strike; one with both (Lightning, Revitalize, Spider's Kiss, Firebolt)
// procs on either.
func TestMeleeSetBonusesProcOnTheClientsAttacks(t *testing.T) {
	cases := []struct {
		name    string
		bonusID int32
		want    core.ProcMask
	}{
		{"Defias Leather 4P Devious Strike", 1292028, core.ProcMaskMeleeWhiteHit},
		{"Shadowcraft 5P Rogue Armor Energize", 27787, core.ProcMaskMeleeWhiteHit},
		{"Battlegear of Valor 5P", 450587, core.ProcMaskMeleeWhiteHit},
		{"Bloodmail Regalia 3P", 1299740, core.ProcMaskMeleeWhiteHit},
		{"Stormshroud 2P Lightning", 18979, core.ProcMaskMelee},
		{"Stormshroud 3P Revitalize", 23863, core.ProcMaskMelee},
		{"Spider's Kiss 2P", 17332, core.ProcMaskMelee},
		{"Firebolt", 9233, core.ProcMaskMelee},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := meleeKindOf(tc.bonusID).procMask; got != tc.want {
				t.Errorf("procs on mask %v, want %v (client ProcTypeMask %#x)", got, tc.want, core.MustClientSpellRow(tc.bonusID).ProcTypeMask)
			}
		})
	}
}
