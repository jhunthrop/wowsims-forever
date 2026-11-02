package hunter

import (
	"testing"

	"github.com/wowsims/classic/sim/common/clientdamage/clientdamagetest"
)

// TestPetClawRanksRollTheClientRange: Claw ranks 1-8 (16827-16832, 3010,
// 3009) roll 4-6, 8-12, 12-16, 16-22, 21-29, 26-36, 35-49 and 43-59 (base
// points and variance; Wowhead Forever "4 to 6", "12 to 16", "21 to 29").
// Ranks 1-3 and 5 used rank 4's 16-22.
func TestPetClawRanksRollTheClientRange(t *testing.T) {
	client := clientdamagetest.LoadClient(t, "..", "hunter")
	for rank := 1; rank < len(petClawSpellID); rank++ {
		got := [2]float64{petClawBaseDamageMin[rank], petClawBaseDamageMax[rank]}
		clientdamagetest.AssertRoll(t, client, "Claw", petClawSpellID[rank], 0, petClawLearnLevels[rank-1], got)
	}
}

// TestPetBiteRanksRollTheClientRange: Bite ranks 1-8 (17253, 17255-17261)
// roll 7-9, 16-18, 24-28, 31-37, 40-48, 49-59, 66-80 and 81-99 (base points
// plus or minus half the variance). Ranks 1-3 and 5 used rank 4's 31-37,
// and rank 8 topped out at 91.
func TestPetBiteRanksRollTheClientRange(t *testing.T) {
	client := clientdamagetest.LoadClient(t, "..", "hunter")
	for rank := 1; rank < len(petBiteSpellID); rank++ {
		got := [2]float64{petBiteBaseDamageMin[rank], petBiteBaseDamageMax[rank]}
		clientdamagetest.AssertRoll(t, client, "Bite", petBiteSpellID[rank], 0, petBiteLearnLevels[rank-1], got)
	}
}
