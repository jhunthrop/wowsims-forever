package enhancement

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

// A rotation reads the Maelstrom Weapon rank through a dormant marker
// aura, keyed by the talent spell (408498) and tagged with the rank, the
// way mage reads Missile Barrage. Below 5/5 a five-stack Lightning Bolt
// still costs mana, so Enhancement's Earth Shock keeps half its mana there
// (sim-accuracy round 19). The untagged id stays the stacking buff.
func TestMaelstromWeaponTriggerCarriesItsRank(t *testing.T) {
	for rank := 1; rank <= 5; rank++ {
		_, built := newIsolatedTalentShaman(t, shamanTalentString(nil, map[int]int{enhNodeMaelstromWeapon: rank}, nil))
		for tag := int32(1); tag <= 5; tag++ {
			aura := built.GetAuraByID(core.ActionID{SpellID: 408498, Tag: tag})
			if (aura != nil) != (tag == int32(rank)) {
				t.Errorf("Maelstrom Weapon %d/5: rank aura tagged %d registered = %v", rank, tag, aura != nil)
			}
			// The rank marker only answers auraIsKnown: it is never up, so it
			// adds no 100%-uptime "Maelstrom Weapon" line to the aura metrics
			// (sim-accuracy round 20).
			if aura != nil && aura.IsActive() {
				t.Errorf("Maelstrom Weapon %d/5: the rank marker aura tagged %d is active", rank, tag)
			}
		}
		if buff := built.GetAuraByID(core.ActionID{SpellID: 408498}); buff != built.MaelstromWeaponAura {
			t.Errorf("Maelstrom Weapon %d/5: the untagged id is not the stacking buff", rank)
		}
	}
}
