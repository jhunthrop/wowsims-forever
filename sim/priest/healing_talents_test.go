package priest

import "testing"

// healingTalentsApplied is every talent healing_talents.go reads by name.
// Keep it in step with that file, as foreverShadowTalentsApplied is for the
// Shadow ones.
var healingTalentsApplied = []string{
	"improved_renew",
	"improved_power_word_shield",
	"improved_healing",
	"mental_agility",
	"divine_fury",
	"soul_warding",
	"spiritual_healing",
	"renewed_hope",
	"divine_aegis",
	"inspiration",
	"litany_of_light",
	"improved_inner_fire",
	"spiritual_guidance",
	"mental_strength",
}

func TestEveryHealingTalentExistsInTheClientsTree(t *testing.T) {
	for _, name := range healingTalentsApplied {
		if _, ok := TalentNodeIDs[name]; !ok {
			t.Errorf("the healing talents read %q, which is not in the client's tree", name)
		}
	}
}

// TestPerRankTablesCoverEveryRankOfTheirTalent ties the tables of
// non-linear talents to the tree: one entry per rank, and rank 0 (no
// talent) worth nothing.
func TestPerRankTablesCoverEveryRankOfTheirTalent(t *testing.T) {
	tables := map[string]int{
		"improved_power_word_shield": len(improvedPowerWordShieldAbsorb),
		"mental_agility":             len(mentalAgilityCostPct),
		"spiritual_healing":          len(spiritualHealingHealing),
		"inspiration":                len(inspirationArmor),
	}
	for name, entries := range tables {
		if want := len(TalentSpellIDs[name]) + 1; entries != want {
			t.Errorf("%s table has %d entries, want %d (rank 0 and one per rank)", name, entries, want)
		}
	}
	if improvedPowerWordShieldAbsorb[0] != 0 || mentalAgilityCostPct[0] != 0 || spiritualHealingHealing[0] != 0 || inspirationArmor[0] != 0 {
		t.Error("a per-rank table gives a bonus with no points spent")
	}
}

// TestHealingMasksAreDistinctFromTheShadowOnes: Twin Disciplines and Mental
// Agility union the Shadow and healing masks, so a shared bit would make a
// Shadow talent touch a heal.
func TestHealingMasksAreDistinctFromTheShadowOnes(t *testing.T) {
	if PriestSpellMaskInstantHealing&PriestSpellMaskInstantShadowDamage != 0 {
		t.Error("the instant healing and instant Shadow damage masks share a bit")
	}
	instants := PriestSpellMaskInstantCast
	for name, mask := range map[string]uint64{
		"Smite": PriestSpellMaskSmite, "Holy Fire": PriestSpellMaskHolyFire,
		"Flash Heal": PriestSpellMaskFlashHeal, "Greater Heal": PriestSpellMaskGreaterHeal,
	} {
		if instants&mask != 0 {
			t.Errorf("%s has a cast time but sits in the instant-cast mask", name)
		}
	}
}
