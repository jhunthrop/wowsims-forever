package core

import "testing"

// Round 9 (auras lane): the three reactive raid buffs held one rank's amount
// at every level. Each row is the client's rank for that level (1.60.1.70707
// SpellEffect with the hotfix push, SpellLevels; Wowhead Forever's tooltips
// agree).

// Thorns 467, 782, 1075, 8914, 9756, 9910 (levels 6, 14, 24, 34, 44, 54):
// 3, 6, 9, 12, 15 and 18 Nature damage (aura 15). The engine gave 18 at
// every level.
func TestThornsFollowsTheClientRanks(t *testing.T) {
	checkRanks(t, "Thorns", ThornsRanks, []levelWant{
		{5, 0}, {6, 3}, {14, 6}, {20, 6}, {24, 9}, {30, 9}, {34, 12}, {44, 15}, {53, 15}, {54, 18}, {60, 18},
	})
}

// Retribution Aura 7294, 10298, 10299, 10300, 10301 (levels 16, 26, 36, 46,
// 56): 5, 8, 12, 16 and 20 Holy damage (aura 15). The engine gave 20 at
// every level.
func TestRetributionAuraFollowsTheClientRanks(t *testing.T) {
	checkRanks(t, "Retribution Aura", RetributionAuraRanks, []levelWant{
		{15, 0}, {16, 5}, {20, 5}, {26, 8}, {30, 8}, {36, 12}, {40, 12}, {46, 16}, {50, 16}, {56, 20}, {60, 20},
	})
}

// Stoneskin Totem 8071, 8154, 8155, 10406, 10407, 10408 (levels 4, 14, 24,
// 34, 44, 54) place the Stoneskin auras 8072, 8156, 8157, 10403, 10404,
// 10405: Physical damage taken reduced by 4, 7, 11, 16, 22 and 30 (aura 14,
// misc 1). The engine gave 30 at every level.
func TestStoneskinFollowsTheClientRanks(t *testing.T) {
	checkRanks(t, "Stoneskin", StoneskinRanks, []levelWant{
		{3, 0}, {4, 4}, {14, 7}, {20, 7}, {24, 11}, {30, 11}, {34, 16}, {44, 22}, {50, 22}, {54, 30}, {60, 30},
	})
}

// The Stoneskin aura's expire added its reduction a second time instead of
// taking it back, and every aura expires at the end of an iteration, so the
// reduction grew by one rank's amount each iteration of a run. Physical
// damage taken is reduced for melee and ranged attacks alike (aura 14 is
// school-wide).
func TestStoneskinExpiresCleanly(t *testing.T) {
	unit := newBareTestUnit()
	unit.Level = 30
	aura := StoneskinTotemAura(unit, 0)
	sim := &Simulation{}
	for iteration := 0; iteration < 3; iteration++ {
		aura.Activate(sim)
		for _, defense := range []DefenseType{DefenseTypeMelee, DefenseTypeRanged} {
			if got := unit.PseudoStats.BonusDamageTakenAfterModifiers[defense]; got != -11 {
				t.Fatalf("iteration %d: Stoneskin at level 30 reduces defense type %d by %v, want -11", iteration, defense, got)
			}
		}
		aura.Deactivate(sim)
		for _, defense := range []DefenseType{DefenseTypeMelee, DefenseTypeRanged} {
			if got := unit.PseudoStats.BonusDamageTakenAfterModifiers[defense]; got != 0 {
				t.Fatalf("iteration %d: after Stoneskin expires defense type %d is %v, want 0", iteration, defense, got)
			}
		}
	}
}

// A party's Mana Tide Totem (the custom panel's mana_tide_totems) restored
// 290 mana a tick at every level. Mana Tide Totem is Restoration tier 3 in
// Forever (level 25); its ticks 16191, 17355, 17360 and 1325257 restore 88,
// 133, 197 and 290 mana every 3 s (aura 24) from the totem ranks learned at
// 1 (with the talent), 38, 48 and 58.
func TestPartyManaTideFollowsTheClientRanks(t *testing.T) {
	for _, c := range []levelWant{{20, 0}, {24, 0}, {25, 88}, {37, 88}, {38, 133}, {47, 133}, {48, 197}, {57, 197}, {58, 290}, {60, 290}} {
		if got := PartyManaTidePerTick(int32(c.level)); got != c.want {
			t.Errorf("party Mana Tide at level %d = %v a tick, want %v", c.level, got, c.want)
		}
	}
}
