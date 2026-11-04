package mage

import (
	"strings"
	"testing"

	"github.com/wowsims/classic/sim/core"
)

// missileBarrageRankTwoTalents is ForeverFrostTalents with Missile Barrage
// at 2/2 instead of 1/2.
var missileBarrageRankTwoTalents = strings.Replace(ForeverFrostTalents, "050005123", "050005223", 1)

// A rotation reads the Missile Barrage rank through the talent's own aura,
// keyed by the talent spell (400588) and tagged with the rank: a leveling
// Arcane mage with one rank still leads with Arcane Missiles, and only both
// ranks make Blast-first the better play (round 18, levels 22 and 23).
func TestMissileBarrageTalentAuraCarriesItsRank(t *testing.T) {
	for _, tc := range []struct {
		talents string
		rank    int32
	}{
		{ForeverFrostTalents, 1},
		{missileBarrageRankTwoTalents, 2},
	} {
		built, _, _ := newFrostMageSimWithTalents(t, core.DefaultTargetProtoLvl60, tc.talents)
		for tag := int32(1); tag <= 2; tag++ {
			aura := built.GetAuraByID(core.ActionID{SpellID: missileBarrageTalentSpellId, Tag: tag})
			if (aura != nil) != (tag == tc.rank) {
				t.Errorf("Missile Barrage %d/2: talent aura tagged %d registered = %v", tc.rank, tag, aura != nil)
			}
		}
	}
}
