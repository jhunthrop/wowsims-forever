package rogue_test

import "testing"

// Feint trains at 16 and Vanish at 22 (trainables 1966, 1856). Both had no
// cast-time level check, so a rogue below those levels could cast a spell the
// trainer has not taught. Registration is the one truth: a spell the rogue has
// not learned is not registered, so spellIsKnown reads false and no APL line
// can cast it.
func TestFeintAndVanishAreNotKnownBelowTheirTrainerLevel(t *testing.T) {
	for _, tc := range []struct {
		level         int32
		feint, vanish bool
	}{{15, false, false}, {16, true, false}, {21, true, false}, {22, true, true}} {
		_, built := buildRogueSimAtLevel(t, zeroRogueTalents, "blank", tc.level)
		r := built.GetRogue()
		if got := r.Feint != nil; got != tc.feint {
			t.Errorf("level %d: Feint registered = %v, want %v", tc.level, got, tc.feint)
		}
		if got := r.Vanish != nil; got != tc.vanish {
			t.Errorf("level %d: Vanish registered = %v, want %v", tc.level, got, tc.vanish)
		}
	}
}
