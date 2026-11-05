package warrior

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
)

// Defensive Stance 7376's +80% and Defiance 12792's +15% are two aura 10
// (SPELL_AURA_MOD_THREAT) effects. cmangos-classic's Aura::HandleModThreat
// applies each with ApplyPercentModFloatVar (a multiply), as TrinityCore does
// and as upstream wowsims/classic's stances.go does; round 5 multiplied
// Righteous Fury's two aura 10 effects on the same evidence. The client's own
// tooltip for 71 and 7376 multiplies them too: "($7376s3+100)*($12792s1/100+1)
// -100", which Wowhead's Forever tooltip renders as 107% (1.8 x 1.15).
//
// The stance's bonus is 80% since Blizzard's 3 November Forever notes
// ("Defensive Stance: Threat bonus increased from 30% to 80%"), live on
// Wowhead's Forever tooltips for 71 and 7376; the cached 1.60.1.70790
// SpellEffect row 686475 still reads 30.
func TestDefensiveStanceAndDefianceThreatMultiply(t *testing.T) {
	warrior := &Warrior{Talents: &proto.WarriorTalents{Defiance: 3}}
	warrior.PseudoStats.CanBlock = true
	if got, want := warrior.defensiveStanceThreat(), 1.8*1.15; math.Abs(got-want) > 1e-9 {
		t.Errorf("Defensive Stance with Defiance 3/3 = %v threat, want %v", got, want)
	}
	warrior.PseudoStats.CanBlock = false
	if got := warrior.defensiveStanceThreat(); math.Abs(got-1.8) > 1e-9 {
		t.Errorf("Defensive Stance without a shield = %v threat, want 1.8", got)
	}
}
