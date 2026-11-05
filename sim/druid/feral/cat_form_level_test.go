package feral

import (
	"testing"

	"github.com/wowsims/classic/sim/druid"
)

// Cat Form (768) is trained at 20 (client 1.60.1.70790 SkillLineAbility,
// trainables/druid.json). Below that a feral druid has no Cat Form: it
// starts the fight in caster form and cannot shift, so nothing that needs
// the form (Faerie Fire's free feral cast from 18) is cast in it.
func TestCatFormIsLearnedAtTwenty(t *testing.T) {
	for _, c := range []struct {
		level  int32
		inForm bool
	}{{19, false}, {20, true}} {
		built, sim, target := newUndebuffedFeralSim(t, c.level)
		if got := built.InForm(druid.Cat); got != c.inForm {
			t.Errorf("level %d starts in Cat Form %v, want %v", c.level, got, c.inForm)
		}
		if got := built.CatForm != nil; got != c.inForm {
			t.Errorf("level %d has the Cat Form spell %v, want %v", c.level, got, c.inForm)
		}
		if !c.inForm && built.FaerieFireFeral != nil && built.FaerieFireFeral.CanCast(sim, target) {
			t.Errorf("level %d can cast the feral-form Faerie Fire without Cat Form", c.level)
		}
	}
}

// Below 20 the druid swings its own weapon (unarmed here), not the paw.
func TestFeralBelowTwentySwingsItsOwnWeapon(t *testing.T) {
	built, _, _ := newUndebuffedFeralSim(t, 19)
	if got, want := built.AutoAttacks.MH().BaseDamageMax, built.WeaponFromMainHand().BaseDamageMax; got != want {
		t.Errorf("a level-19 feral swings for up to %v, want its own weapon's %v", got, want)
	}
}
