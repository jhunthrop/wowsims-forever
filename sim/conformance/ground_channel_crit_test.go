package conformance

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

// Forever's ground-targeted channels deal their damage through a server
// tick spell, not through the channel's own periodic aura: the channel is a
// dummy (aura 226) and each pulse casts a direct-damage row the client
// carries separately. Blizzard ticks 1279976 to 1279980 and 1279949,
// Hurricane 1278759, 1278965 and 1278968, Rain of Fire 1282380 to 1282385
// and Volley 1279715, 1279719 and 1279721 are plain school damage
// (Effect 2) with no SPELL_ATTR2_CANT_CRIT (70888 and 70938 SpellMisc), so
// each pulse rolls crit like any direct spell. The logs agree: Blizzard
// 902 crits in 2,739 ticks, Rain of Fire 200 in 3,311, Volley (DefenseType
// 3, ranged) 52 in 478 (design/reviews/2026-11-13-sim-accuracy/crit-flags.md).
// The channels' own PERIODIC_CAN_CRIT flag never applies: they carry no
// periodic damage effect for it to mark.
func TestGroundChannelTicksCrit(t *testing.T) {
	for _, tc := range []struct {
		preset  string
		spellID int32
		defense core.DefenseType // checked when set: the crit table the tick rolls on
	}{
		{"Mage", 10187, core.DefenseTypeNone},          // Blizzard rank 6
		{"BalanceDruid", 17402, core.DefenseTypeNone},  // Hurricane rank 3
		{"SMRuinWarlock", 11678, core.DefenseTypeNone}, // Rain of Fire rank 4
		{"Hunter", 14295, core.DefenseTypeRanged},      // Volley rank 3
	} {
		t.Run(tc.preset, func(t *testing.T) {
			preset, ok := presetByLabel(tc.preset)
			if !ok {
				t.Fatalf("no preset %q", tc.preset)
			}
			built, err := buildCharacter(preset, 60, "")
			if err != nil {
				t.Fatal(err)
			}
			spell := built.GetSpell(core.ActionID{SpellID: tc.spellID})
			if spell == nil || spell.AOEDot() == nil {
				t.Fatalf("%s has no channel %d with an area dot", tc.preset, tc.spellID)
			}
			if !spell.AOEDot().CanCrit {
				t.Errorf("%s %d ticks never crit; the client's tick rows may crit and the logs show crits", tc.preset, tc.spellID)
			}
			if tc.defense != core.DefenseTypeNone && spell.DefenseType != tc.defense {
				t.Errorf("%s %d DefenseType = %v, want %v (the tick row's SpellCategories DefenseType)", tc.preset, tc.spellID, spell.DefenseType, tc.defense)
			}
		})
	}
}
