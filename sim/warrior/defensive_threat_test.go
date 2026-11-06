package warrior

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
)

// Defensive Stance 7376's +80% and Defiance 12792's +20% are two aura 10
// (SPELL_AURA_MOD_THREAT) effects. cmangos-classic's Aura::HandleModThreat
// applies each with ApplyPercentModFloatVar (a multiply), as TrinityCore does
// and as upstream wowsims/classic's stances.go does; round 5 multiplied
// Righteous Fury's two aura 10 effects on the same evidence. The client's own
// tooltip for 71 and 7376 multiplies them too: "($7376s3+100)*($12792s1/100+1)
// -100": 116% at 3/3 (1.8 x 1.20).
//
// The stance's 80% and Defiance's 7/13/20% are Blizzard's 3 November Forever
// notes, which the 1.60.1.70790 server push 113326 carries.
func TestDefensiveStanceAndDefianceThreatMultiply(t *testing.T) {
	warrior := &Warrior{Talents: &proto.WarriorTalents{Defiance: 3}}
	warrior.PseudoStats.CanBlock = true
	if got, want := warrior.defensiveStanceThreat(), 1.8*1.20; math.Abs(got-want) > 1e-9 {
		t.Errorf("Defensive Stance with Defiance 3/3 = %v threat, want %v", got, want)
	}
	warrior.PseudoStats.CanBlock = false
	if got := warrior.defensiveStanceThreat(); math.Abs(got-1.8) > 1e-9 {
		t.Errorf("Defensive Stance without a shield = %v threat, want 1.8", got)
	}
}
