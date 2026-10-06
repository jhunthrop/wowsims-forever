package priest

import "testing"

// foreverShadowTalentsApplied is every talent applyDeclarativeShadowTalents
// reads. Keep it in step with talents.go: a talent that is applied but
// absent from the client's generated tables is one this test cannot
// protect, the same convention sim/mage/talents_test.go's
// foreverFrostTalentsApplied establishes.
var foreverShadowTalentsApplied = []string{
	"twin_disciplines",
	"improved_mind_flay",
	"devouring_contagion",
}

func TestEveryDeclarativeShadowTalentExistsInTheClientsTree(t *testing.T) {
	for _, name := range foreverShadowTalentsApplied {
		if _, ok := TalentNodeIDs[name]; !ok {
			t.Errorf("talents.go applies %q, which is not in the client's tree", name)
		}
		if len(TalentSpellIDs[name]) == 0 {
			t.Errorf("%q has no rank spell ids", name)
		}
	}
}

// Twin Disciplines, Improved Mind Flay and Devouring Contagion's masks
// must be non-zero and pairwise distinct, or applyDeclarativeShadowTalents'
// AddStaticMod calls bind to the wrong spells (or nothing at all: bit 0
// is reserved so ClassMask is never read as "no filter").
func TestPriestSpellMasksAreNonZeroAndDistinct(t *testing.T) {
	named := map[string]uint64{
		"PriestSpellMaskMindFlay":        PriestSpellMaskMindFlay,
		"PriestSpellMaskDevouringPlague": PriestSpellMaskDevouringPlague,
		"PriestSpellMaskShadowWordPain":  PriestSpellMaskShadowWordPain,
		"PriestSpellMaskShadowWordDeath": PriestSpellMaskShadowWordDeath,
	}
	for name, mask := range named {
		if mask == 0 {
			t.Errorf("%s is zero, which AddStaticMod's ClassMask reads as \"no filter\"", name)
		}
		for other, otherMask := range named {
			if name < other && mask == otherMask {
				t.Errorf("%s and %s share mask %#x", name, other, mask)
			}
		}
	}

	if PriestSpellMaskInstantShadowDamage&PriestSpellMaskMindFlay == 0 ||
		PriestSpellMaskInstantShadowDamage&PriestSpellMaskDevouringPlague == 0 ||
		PriestSpellMaskInstantShadowDamage&PriestSpellMaskShadowWordPain == 0 ||
		PriestSpellMaskInstantShadowDamage&PriestSpellMaskShadowWordDeath == 0 {
		t.Error("PriestSpellMaskInstantShadowDamage must carry all four instant-cast Shadow damage spells")
	}
}

// The per-rank constants are read directly off the client's rank
// descriptions (talents/priest.json, build 1.60.1.70009); this pins the
// values against a sign/scale mistake (e.g. reading a percentage point
// as a fraction of itself).
func TestDeclarativeShadowTalentPerRankValuesMatchTheClient(t *testing.T) {
	// "Increases the damage and healing of your instant cast spells by
	// 1%/2%/3%/4%/5%": a flat 1% per rank, five ranks.
	if got, want := twinDisciplinesDamagePerRank, 0.01; got != want {
		t.Errorf("twinDisciplinesDamagePerRank = %v, want %v", got, want)
	}
	// "Your Mind Flay now deals 10%/20% more damage...": 10% a rank,
	// two ranks.
	if got, want := improvedMindFlayDamagePerRank, 0.10; got != want {
		t.Errorf("improvedMindFlayDamagePerRank = %v, want %v", got, want)
	}
	// "Reduces the mana cost of your Devouring Plague by 25%/50%": -25
	// percentage points a rank, matching SpellMod_PowerCost_Pct's own
	// convention ("-5% = -5", IntValue added straight to Cost.Multiplier).
	if got, want := int64(devouringContagionCostPctPerRank), int64(-25); got != want {
		t.Errorf("devouringContagionCostPctPerRank = %v, want %v", got, want)
	}
}

// An unvalidated or stale talent string (core.FillTalentsProto does not
// check a rank against its talent's real maximum) must clamp rather than
// let a garbage rank scale a modifier past where the talent could ever
// actually reach. This is not hypothetical here: sim/priest/shadow's own
// P1Talents, authored before the client talent-tree rewrite, reads
// DevouringContagion as rank 5 against a max of 2 - see rankOf's comment
// in talents.go.
func TestRankOfClampsAnOverRankToTheTalentsMaxRank(t *testing.T) {
	for _, c := range []struct {
		talent  string
		maxRank int32
	}{
		{"twin_disciplines", 5},
		{"improved_mind_flay", 2},
		{"devouring_contagion", 2},
	} {
		if got := rankOf(c.talent, 9); got != c.maxRank {
			t.Errorf("rankOf(%q, 9) = %d, want the talent's max rank %d", c.talent, got, c.maxRank)
		}
		// A rank inside the table still reads its own value, not the max.
		if got := rankOf(c.talent, 1); got != 1 {
			t.Errorf("rankOf(%q, 1) = %d, want 1", c.talent, got)
		}
	}
}
