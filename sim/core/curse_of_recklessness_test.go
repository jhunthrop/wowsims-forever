package core

import "testing"

// Forever's Curse of Recklessness rank 4 (11717) is SpellEffect index 1,
// aura 674 at -505 armor, beside aura 92 (no fleeing). Era's rank carried
// -640 armor and +90 attack power (aura 99); Forever dropped the attack
// power. The Forever tooltip reads "reducing the target's armor by 505".
func TestCurseOfRecklessnessIsForevers(t *testing.T) {
	if curseOfRecklessnessArmorReduction != 505 {
		t.Errorf("Curse of Recklessness armor reduction = %v, want 505", curseOfRecklessnessArmorReduction)
	}
}
