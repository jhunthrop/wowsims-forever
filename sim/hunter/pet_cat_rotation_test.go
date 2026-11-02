package hunter

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// Round 9 (physical lane): the cat cast Bite only while its focus regen
// beat Claw's cost / 1.6, 15.6 focus a second. A pet regenerates 26.25
// focus every 5.25 s (5 a second, 6 with Bestial Discipline), so the gate
// never opened and no breakdown had a Bite. Bite (17261: 35 focus, 10 s,
// 81 to 99) deals 2.6 damage a focus against Claw's 2.0 (3009: 25 focus,
// 43 to 59), so a cat bites whenever Bite is ready and claws otherwise.
func TestCatBitesWheneverBiteIsReady(t *testing.T) {
	sim, built, _ := buildSurvivalMeleeHunter(t)
	hp := built.pet
	if hp == nil || hp.specialAbility == nil || hp.focusDump == nil {
		t.Fatal("the test hunter has no cat with Bite and Claw")
	}
	if !hp.specialAbility.CD.IsReady(sim) {
		t.Fatal("Bite is not ready at the pull")
	}
	var tried []*core.Spell
	PetConfigs[proto.Hunter_Options_Cat].CustomRotation(sim, hp, func(spell *core.Spell) bool {
		tried = append(tried, spell)
		return true
	})
	if len(tried) == 0 || tried[0] != hp.specialAbility {
		t.Errorf("with Bite ready the cat tried %v first, want Bite %v", tried, hp.specialAbility.ActionID)
	}
}
