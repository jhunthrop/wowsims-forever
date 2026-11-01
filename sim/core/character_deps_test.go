package core

import (
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// Forever's items state the damage third of a healing bonus as its own stat:
// 320 rows of the 1.60.1.70707 client's ItemSparse carry healing (stat 41)
// beside spell damage (stat 42) at a third of it, and the server tooltip
// prints both ("Increases healing done by up to 17 and damage done by up to
// 6", Torch of Light 279246). A healing-only row prints no damage at all
// (Rejuvenating Gem 19395). So the healing stat itself grants no damage; a
// hidden conversion would count the stated third twice.
func TestHealingPowerGrantsNoHiddenSpellDamage(t *testing.T) {
	character := newTestCharacter(t, CharacterMaxLevel, proto.Class_ClassPriest, proto.Race_RaceHuman)
	result := character.SortAndApplyStatDependencies(stats.Stats{
		stats.HealingPower: 300,
		stats.SpellDamage:  50,
	})
	if result[stats.HealingPower] != 300 {
		t.Errorf("HealingPower = %v, want 300", result[stats.HealingPower])
	}
	if result[stats.SpellDamage] != 50 {
		t.Errorf("SpellDamage = %v, want the stated 50 (healing adds none of its own)", result[stats.SpellDamage])
	}
}
