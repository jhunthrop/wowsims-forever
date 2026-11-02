package warrior

import (
	"math"
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core/proto"
)

// Dual Wield Specialization 23584 is aura 122, the off-hand damage percent
// (+5% a rank): it raises the off-hand weapon's damage for every off-hand
// strike, so Whirlwind's off-hand hit ("weapon damage from both melee
// weapons", 1680) takes it as the white off-hand swing does.
func TestWhirlwindOffHandTakesDualWieldSpecialization(t *testing.T) {
	warrior := &Warrior{Talents: &proto.WarriorTalents{DualWieldSpecialization: 5}}
	if got := warrior.offHandStrikeMultiplier(); math.Abs(got-1.25) > 1e-9 {
		t.Errorf("off-hand strike multiplier at Dual Wield Specialization 5/5 = %v, want 1.25", got)
	}
	warrior.Talents.DualWieldSpecialization = 0
	if got := warrior.offHandStrikeMultiplier(); got != 1 {
		t.Errorf("off-hand strike multiplier untalented = %v, want 1", got)
	}
}

// Impale 16493 is aura 108 misc 15 (crit damage bonus) over family mask
// 0xee604cee, which carries Rend's bit 32; Rend crits under the owner's bleed
// ruling, so its crits take Impale like every other spell in the mask.
func TestRendCritsTakeImpale(t *testing.T) {
	warrior := &Warrior{Talents: &proto.WarriorTalents{Impale: 2}}
	if got := warrior.rendCritDamageBonus(); math.Abs(got-0.2) > 1e-9 {
		t.Errorf("Rend crit damage bonus at Impale 2/2 = %v, want 0.2", got)
	}
}

// Bloodthrill's Overpower window 1282733 has DurationIndex 28, which
// SpellDuration gives as 5000 ms.
func TestBloodthrillWindowIsFiveSeconds(t *testing.T) {
	if bloodthrillWindow != 5*time.Second {
		t.Errorf("Bloodthrill window = %v, want 5s (SpellDuration 28)", bloodthrillWindow)
	}
}
