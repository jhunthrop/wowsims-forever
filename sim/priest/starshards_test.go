package priest

import "testing"

// TestTwinDisciplinesRaisesStarshards: Twin Disciplines' periodic modifier
// (1225132 eff1, aura 108 misc 22, mask 0xa08060/0/0/0x10) names Starshards
// by its class mask bit (10797 and 19296 to 19305: 0x200000), and Blizzard's
// 16 October notes say so ("Twin Disciplines now correctly benefits Shadow
// Word: Death, Mind Flay, Starshards and Penance"). The engine's Starshards
// carried no class mask, so no talent modifier could reach it.
func TestTwinDisciplinesRaisesStarshards(t *testing.T) {
	if PriestSpellMaskTwinDisciplines&PriestSpellMaskStarshards == 0 {
		t.Error("Twin Disciplines leaves out Starshards, which its client mask names")
	}
	config := (&Priest{}).newStarshardsSpellConfig(StarshardsRanks, 0)
	if config.ClassSpellMask != PriestSpellMaskStarshards {
		t.Errorf("Starshards carries class mask %#x, want %#x", config.ClassSpellMask, PriestSpellMaskStarshards)
	}
}
