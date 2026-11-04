package dpsrogue

import (
	"fmt"
	"testing"
)

const (
	deathMarkRank4ID = 1325758
	shankRank2ID     = 1329549
)

// TestDeathMarkAndShankRunInARotation casts Death Mark (rank 4 at 60, 10
// sec cooldown) at five combo points, Shank on cooldown and Sinister Strike
// otherwise: both new 70755 abilities must cast from an APL, and Death Mark
// at most once per 10 sec.
func TestDeathMarkAndShankRunInARotation(t *testing.T) {
	talents := rogueTalentStringWithRanks(t, map[string]int{"death_mark": 1, "boot_knife": 5})
	apl := fmt.Sprintf(rotationBoilerFmt, `
		{"action":{"condition":{"cmp":{"op":"OpGe","lhs":{"currentComboPoints":{}},"rhs":{"const":{"val":"5"}}}},
		 "castSpell":{"spellId":{"spellId":1325758}}}},
		{"action":{"castSpell":{"spellId":{"spellId":1329549}}}},
		{"action":{"castSpell":{"spellId":{"spellId":11294}}}}`)

	withMark := runRotationWithTalents(t, talents, apl, 60, 20)
	t.Logf("dps %.1f casts %v", withMark.dps, withMark.casts)
	if withMark.casts[shankRank2ID] < 4 {
		t.Errorf("Shank casts = %.2f over 60s, want at least 4 (10 sec cooldown)", withMark.casts[shankRank2ID])
	}
	if got := withMark.casts[deathMarkRank4ID]; got < 2 || got > 7 {
		t.Errorf("Death Mark casts = %.2f over 60s, want between 2 and 7", got)
	}
}
