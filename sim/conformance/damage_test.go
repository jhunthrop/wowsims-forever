package conformance

import (
	"strings"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/spellconst"
)

// fireball12 is the client's own Fireball rank 12 (spell 25306).
func fireball12() spellconst.Spell {
	return spellconst.Spell{
		ID: 25306, Name: "Fireball", Rank: 12, SpellLevel: 60, MaxLevel: 64,
		Effects: []spellconst.Effect{
			{Index: 0, Effect: 2, Amount: 483, Variance: 0.24188791215, PointsPerLevel: 3, SPCoefficient: 1, ResolvedSPCoefficient: 1, CoefficientSource: "table"},
			{Index: 1, Effect: 6, Aura: 3, Amount: 15, ResolvedSPCoefficient: 0.1, CoefficientSource: "convention"},
		},
	}
}

func engineSpell(low, high, coefficient float64) *core.Spell {
	return &core.Spell{ClientBaseDamage: [2]float64{low, high}, BonusCoefficient: coefficient}
}

func TestCompareDamage_UndeclaredIsNeverAMatch(t *testing.T) {
	got := compareDamage(fireball12(), 60, engineSpell(0, 0, 1))
	if got.Status != DamageUndeclared {
		t.Fatalf("status = %q, want %q", got.Status, DamageUndeclared)
	}
	if got.ClientMin < 424 || got.ClientMax > 542 {
		t.Errorf("client range %.2f-%.2f, want about 424.58-541.42", got.ClientMin, got.ClientMax)
	}
}

func TestCompareDamage_DeclaredMatchesWithinRounding(t *testing.T) {
	got := compareDamage(fireball12(), 60, engineSpell(425, 541, 1))
	if got.Status != DamageMatches {
		t.Fatalf("status = %q (%s), want %q", got.Status, got.Diff, DamageMatches)
	}
}

func TestCompareDamage_VanillaTooltipRollDiffers(t *testing.T) {
	got := compareDamage(fireball12(), 60, engineSpell(596, 760, 1))
	if got.Status != DamageDiffers || !strings.Contains(got.Diff, "damage 425-541->596-760") {
		t.Fatalf("status %q diff %q, want a damage difference naming both ranges", got.Status, got.Diff)
	}
}

func TestCompareDamage_CoefficientDiffersOnlyWhenTheTableStatesIt(t *testing.T) {
	got := compareDamage(fireball12(), 60, engineSpell(425, 541, 0.5))
	if got.Status != DamageDiffers || !strings.Contains(got.Diff, "coefficient 1.000->0.500") {
		t.Fatalf("status %q diff %q, want a coefficient difference", got.Status, got.Diff)
	}
	convention := fireball12()
	convention.Effects[0].CoefficientSource = "convention"
	if got := compareDamage(convention, 60, engineSpell(425, 541, 0.5)); got.Status != DamageMatches {
		t.Errorf("a convention-derived client coefficient must not be compared, got %q (%s)", got.Status, got.Diff)
	}
}

func TestCompareDamage_RangeFollowsThePresetLevel(t *testing.T) {
	got := compareDamage(fireball12(), 62, engineSpell(425, 541, 1))
	if got.Status != DamageDiffers {
		t.Errorf("at level 62 the client rolls 430-548; an own-level table must differ, got %q", got.Status)
	}
}

func TestCompareDamage_NoDamageEffectIsNotApplicable(t *testing.T) {
	buff := spellconst.Spell{Effects: []spellconst.Effect{{Index: 0, Effect: 6, Aura: 29, Amount: 8}}}
	if got := compareDamage(buff, 60, engineSpell(0, 0, 0)); got.Status != DamageNone {
		t.Errorf("status = %q, want %q", got.Status, DamageNone)
	}
}

func TestCompareDamage_PeriodicOnlySpellUsesItsPeriodicEffect(t *testing.T) {
	dot := spellconst.Spell{SpellLevel: 60, Effects: []spellconst.Effect{{Index: 0, Effect: 6, Aura: 3, Amount: 150, ResolvedSPCoefficient: 0.2, CoefficientSource: "table"}}}
	got := compareDamage(dot, 60, engineSpell(150, 150, 0.2))
	if got.Status != DamageMatches {
		t.Errorf("status = %q (%s), want %q", got.Status, got.Diff, DamageMatches)
	}
}

func TestCountDamageOnlyCountsLevelSixty(t *testing.T) {
	rows := []Row{
		{Level: 60, Damage: DamageComparison{Status: DamageMatches}},
		{Level: 60, Damage: DamageComparison{Status: DamageDiffers}},
		{Level: 60, Damage: DamageComparison{Status: DamageUndeclared}},
		{Level: 60, Damage: DamageComparison{Status: DamageNone}},
		{Level: 50, Damage: DamageComparison{Status: DamageMatches}},
	}
	want := DamageCounts{Declared: 2, Matching: 1, Differing: 1, Undeclared: 1, NotApplicable: 1}
	if got := countDamage(rows); got != want {
		t.Errorf("countDamage = %+v, want %+v", got, want)
	}
}

// Flametongue Totem's proc (spell 16389) is a dummy effect whose base
// points the server script scales by weapon speed. Like an absorb it is
// compared only when the ability file declares the base amount.
func flametongueTotemProc() spellconst.Spell {
	return spellconst.Spell{
		ID: 16389, Name: "Flametongue Totem Proc", Rank: 4, SpellLevel: 58, MaxLevel: 66,
		Effects: []spellconst.Effect{{Index: 0, Effect: 3, Amount: 1363}},
	}
}

func TestCompareDamage_ADeclaredDummyAmountIsCompared(t *testing.T) {
	if got := compareDamage(flametongueTotemProc(), 60, engineSpell(1363, 1363, 0)); got.Status != DamageMatches {
		t.Errorf("status = %q (%s), want %q", got.Status, got.Diff, DamageMatches)
	}
	if got := compareDamage(flametongueTotemProc(), 60, engineSpell(1000, 1000, 0)); got.Status != DamageDiffers {
		t.Errorf("a wrong declared dummy amount read %q, want %q", got.Status, DamageDiffers)
	}
}

func TestCompareDamage_AnUndeclaredDummyStaysNotApplicable(t *testing.T) {
	if got := compareDamage(flametongueTotemProc(), 60, engineSpell(0, 0, 0)); got.Status != DamageNone {
		t.Errorf("status = %q, want %q: no dummy spell may become 'not declared' unasked", got.Status, DamageNone)
	}
}

func TestCompareDamage_ADummyOutsideTheWeaponSpeedListIsNotRead(t *testing.T) {
	other := spellconst.Spell{ID: 1310707, SpellLevel: 30, Effects: []spellconst.Effect{{Index: 0, Effect: 3, Amount: 20}}}
	if got := compareDamage(other, 60, engineSpell(23, 23, 0)); got.Status != DamageNone {
		t.Errorf("status = %q, want %q: only the weapon-speed dummies are read", got.Status, DamageNone)
	}
}
