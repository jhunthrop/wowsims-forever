package warrior

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
)

// Defensive Stance 7376's +30% and Defiance 12792's +15% are two aura 10
// (SPELL_AURA_MOD_THREAT) effects. cmangos-classic's Aura::HandleModThreat
// applies each with ApplyPercentModFloatVar (a multiply), as TrinityCore does
// and as upstream wowsims/classic's stances.go does (1.3 x 1.15); round 5
// multiplied Righteous Fury's two aura 10 effects on the same evidence.
func TestDefensiveStanceAndDefianceThreatMultiply(t *testing.T) {
	warrior := &Warrior{Talents: &proto.WarriorTalents{Defiance: 3}}
	warrior.PseudoStats.CanBlock = true
	if got, want := warrior.defensiveStanceThreat(), 1.3*1.15; math.Abs(got-want) > 1e-9 {
		t.Errorf("Defensive Stance with Defiance 3/3 = %v threat, want %v", got, want)
	}
	warrior.PseudoStats.CanBlock = false
	if got := warrior.defensiveStanceThreat(); math.Abs(got-1.3) > 1e-9 {
		t.Errorf("Defensive Stance without a shield = %v threat, want 1.3", got)
	}
}
