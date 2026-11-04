package rogue_test

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

// Blood Splatter (1329256) carries a passive aura so a rotation can ask
// auraIsKnown(1329256): Assassination keeps Rupture up only when the talent
// makes it last 6 s longer (without it Rupture loses at bare 50).
func TestBloodSplatterIsKnownOnlyWithTheTalent(t *testing.T) {
	id := core.ActionID{SpellID: 1329256}
	talented := buildRogueForTalentTest(t, talentStringWithRank(t, zeroRogueTalents, "blood_splatter", 1), "combat_backstab_prebis").GetRogue()
	if talented.GetAuraByID(id) == nil {
		t.Error("a rogue with Blood Splatter has no 1329256 aura")
	}
	untalented := buildRogueForTalentTest(t, zeroRogueTalents, "combat_backstab_prebis").GetRogue()
	if untalented.GetAuraByID(id) != nil {
		t.Error("a rogue without Blood Splatter has its 1329256 aura")
	}
}
