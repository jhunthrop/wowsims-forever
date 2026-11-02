package hunter

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

// Round 9 (physical lane): Savage Strikes (19159, aura 107 misc 7, crit
// chance) carries class mask 66 / 528384 / 2097152: Raptor Strike and
// Mongoose Bite (mask 0 bit 2), Wing Clip (mask 0 bit 64), Counterattack
// (mask 1 bit 524288), Lacerate (mask 1 bit 4096) and Strider Kick (mask 2
// bit 2097152), each read from its own SpellClassOptions row. Wowhead
// Forever: "all your melee abilities". The engine gave it to Raptor Strike
// and Mongoose Bite only.
func TestSavageStrikesReachesEveryMeleeAbility(t *testing.T) {
	_, built, _ := buildSurvivalMeleeHunter(t)
	want := built.savageStrikesCritRating()
	if want <= 0 {
		t.Fatalf("the test build takes no Savage Strikes (crit rating %v)", want)
	}
	for _, c := range []struct {
		name  string
		bonus func() (float64, bool)
	}{
		{"Raptor Strike", func() (float64, bool) { return spellBonusCrit(built.RaptorStrikeHit) }},
		{"Mongoose Bite", func() (float64, bool) { return spellBonusCrit(built.MongooseBite) }},
		{"Wing Clip", func() (float64, bool) { return spellBonusCrit(built.WingClip) }},
		{"Counterattack", func() (float64, bool) { return spellBonusCrit(built.Counterattack) }},
		{"Lacerate", func() (float64, bool) { return spellBonusCrit(built.Lacerate) }},
		{"Strider Kick", func() (float64, bool) { return spellBonusCrit(built.StriderKick) }},
	} {
		got, ok := c.bonus()
		if !ok {
			t.Errorf("%s is not registered on the test build", c.name)
			continue
		}
		if got < want {
			t.Errorf("%s bonus crit rating = %v, want Savage Strikes' %v", c.name, got, want)
		}
	}
}

// spellBonusCrit is a registered spell's own bonus crit rating.
func spellBonusCrit(spell *core.Spell) (float64, bool) {
	if spell == nil {
		return 0, false
	}
	return spell.BonusCritRating, true
}
