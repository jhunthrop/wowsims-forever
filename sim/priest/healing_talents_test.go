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

// TestHealingMasksAreDistinctFromTheShadowOnes: Twin Disciplines unions
// the Shadow and healing masks, so a shared bit would make a
// Shadow talent touch a heal.
func TestHealingMasksAreDistinctFromTheShadowOnes(t *testing.T) {
	if PriestSpellMaskInstantHealing&PriestSpellMaskInstantShadowDamage != 0 {
		t.Error("the instant healing and instant Shadow damage masks share a bit")
	}
	instants := PriestSpellMaskInstantShadowDamage | PriestSpellMaskInstantHealing
	for name, mask := range map[string]uint64{
		"Smite": PriestSpellMaskSmite, "Holy Fire": PriestSpellMaskHolyFire,
		"Flash Heal": PriestSpellMaskFlashHeal, "Greater Heal": PriestSpellMaskGreaterHeal,
	} {
		if instants&mask != 0 {
			t.Errorf("%s has a cast time but sits in the instant-cast mask", name)
		}
	}
}

// TestMentalAgilityFollowsTheClientMask: Mental Agility's text says
// "Smite, Holy Fire, and instant cast spells", but its cost modifier
// (14520, SpellEffect 687728, aura 108 misc 14) names spells by class
// mask: Smite, Holy Fire, Shadow Word: Pain, Devouring Plague, Renew,
// Power Word: Shield, Holy Nova, Prayer of Mending, Vampiric Embrace and
// Inner Fire (588-10952, mask 0x2) among the spells the package registers. Mind Flay (mask 0 bit 23),
// Shadow Word: Death (mask 1 bit 1), Penance (mask 1 bit 23) and
// Desperate Prayer (mask 0 bit 24) are outside it.
func TestMentalAgilityFollowsTheClientMask(t *testing.T) {
	for name, mask := range map[string]uint64{
		"Smite": PriestSpellMaskSmite, "Holy Fire": PriestSpellMaskHolyFire,
		"Shadow Word: Pain": PriestSpellMaskShadowWordPain, "Devouring Plague": PriestSpellMaskDevouringPlague,
		"Renew": PriestSpellMaskRenew, "Power Word: Shield": PriestSpellMaskPowerWordShield,
		"Holy Nova": PriestSpellMaskHolyNova, "Prayer of Mending": PriestSpellMaskPrayerOfMending,
		"Vampiric Embrace": PriestSpellMaskVampiricEmbrace, "Inner Fire": PriestSpellMaskInnerFire,
	} {
		if PriestSpellMaskMentalAgility&mask == 0 {
			t.Errorf("Mental Agility leaves out %s, which its client mask names", name)
		}
	}
	for name, mask := range map[string]uint64{
		"Mind Flay": PriestSpellMaskMindFlay, "Shadow Word: Death": PriestSpellMaskShadowWordDeath,
		"Penance": PriestSpellMaskPenance, "Desperate Prayer": PriestSpellMaskDesperatePrayer,
		"Flash Heal": PriestSpellMaskFlashHeal, "Mind Blast": PriestSpellMaskMindBlast,
	} {
		if PriestSpellMaskMentalAgility&mask != 0 {
			t.Errorf("Mental Agility cuts %s, which its client mask does not name", name)
		}
	}
}

// TestTwinDisciplinesRaisesPrayerOfMendingsHeal: Twin Disciplines' healing
// modifier (1225132 eff0 op 0, mask 0x19480020/0x18012/0/0x200000) names
// Prayer of Mending's heal by its one word-3 bit: the three heal spells
// 401863, 1313379 and 1313381 carry class mask 0/0/0/0x200000, and no other
// priest spell carries that bit. The cast (1240827) and the jump (401880)
// do not, which is why round 8 read it the other way; the engine heals
// through the cast, so the cast's mask must carry the modifier.
func TestTwinDisciplinesRaisesPrayerOfMendingsHeal(t *testing.T) {
	if PriestSpellMaskTwinDisciplines&PriestSpellMaskPrayerOfMending == 0 {
		t.Error("Twin Disciplines leaves out Prayer of Mending, whose heal spells its client mask names")
	}
	for name, mask := range map[string]uint64{
		"Renew": PriestSpellMaskRenew, "Penance": PriestSpellMaskPenance, "Holy Nova": PriestSpellMaskHolyNova,
		"Desperate Prayer": PriestSpellMaskDesperatePrayer, "Shadow Word: Pain": PriestSpellMaskShadowWordPain,
	} {
		if PriestSpellMaskTwinDisciplines&mask == 0 {
			t.Errorf("Twin Disciplines leaves out %s, which its client masks name", name)
		}
	}
}
