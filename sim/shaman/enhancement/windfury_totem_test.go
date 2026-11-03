package enhancement

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

// Windfury Totem in the 1.60.1.70753 client: the totem (8512, 10613, 10614)
// summons an NPC whose aura is 8515, 10609 or 10612, an area party aura
// (SpellEffect effect 65, aura 42) with no duration. Vanilla's 10-second
// weapon enchant (8514, 10607, 10611) is not in the client at all, so the
// bonus ends the moment the totem does.
func TestWindfuryTotemBonusEndsWithTheTotem(t *testing.T) {
	sim, built := newIsolatedTalentShaman(t, shamanTalentString(nil, nil, nil))
	windfury := built.WindfuryTotem[len(built.WindfuryTotem)-1]
	grace := built.GraceOfAirTotem[len(built.GraceOfAirTotem)-1]

	aura := built.GetAuraByID(core.ActionID{SpellID: 10612})
	if aura == nil {
		t.Fatal("the shaman has no Windfury Totem aura under the client's id 10612")
	}
	if aura.Duration != core.NeverExpires {
		t.Errorf("the Windfury Totem aura lasts %v on its own; the client's area aura has no duration", aura.Duration)
	}

	windfury.ApplyEffects(sim, &built.Unit, windfury)
	if !aura.IsActive() {
		t.Fatal("dropping Windfury Totem did not put its aura up")
	}
	grace.ApplyEffects(sim, &built.Unit, grace)
	if aura.IsActive() {
		t.Errorf("the Windfury Totem aura is still up %v after Grace of Air replaced the totem", aura.RemainingDuration(sim))
	}
}
