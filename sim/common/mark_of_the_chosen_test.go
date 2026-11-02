package common

import (
	"testing"
	"time"
)

// Mark of the Chosen 17774: 21969 "Has a $h% chance when struck in combat of
// increasing all stats by $21970s1" with SpellAuraOptions 132914 (ProcChance 2,
// ProcCategoryRecovery 120000), and 21970 is aura 29 at 21 on every stat. The
// engine carried the vanilla 25 with no proc cooldown.
func TestMarkOfTheChosenMatchesTheClient(t *testing.T) {
	if markOfTheChosenStats != 21 {
		t.Errorf("Mark of the Chosen stats = %v, want 21 (21970)", markOfTheChosenStats)
	}
	if markOfTheChosenProcCooldown != 120*time.Second {
		t.Errorf("Mark of the Chosen proc cooldown = %v, want 120s (SpellAuraOptions 132914)", markOfTheChosenProcCooldown)
	}
}
