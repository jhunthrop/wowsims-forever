package healing

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

// Client 1.60.1.70755 removed every SkillLineAbility row of Binding Heal
// (401937, 1240770-1240774), so no priest learns it: a healer of any spec
// registers none of its ranks, and a rotation that casts one is refused as
// an unknown spell.
func TestBindingHealIsNotLearnable(t *testing.T) {
	for _, level := range []int32{25, 40, 60} {
		_, agent := healerSim(t, level, "")
		for _, id := range []int32{401937, 1240770, 1240771, 1240772, 1240773, 1240774} {
			if spell := agent.GetSpell(core.ActionID{SpellID: id}); spell != nil {
				t.Errorf("a level-%d priest registers Binding Heal %d", level, id)
			}
		}
	}
}
