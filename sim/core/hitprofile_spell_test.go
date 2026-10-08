package core

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core/stats"
)

func spellHitTestUnits(hitPercent float64) (caster, boss *Unit) {
	caster = &Unit{Type: PlayerUnit, Level: 60, stats: stats.Stats{stats.Hit: hitPercent * HitRatingPerHitChance}}
	boss = &Unit{Type: EnemyUnit, Level: 63}
	return caster, boss
}

func registerDamageSpell(unit *Unit, school SpellSchool, bonusHitPercent float64) *Spell {
	return unit.RegisterSpell(SpellConfig{
		ActionID:         ActionID{SpellID: 1000 + int32(school)},
		SpellSchool:      school,
		ProcMask:         ProcMaskSpellDamage,
		Flags:            SpellFlagAPL,
		BonusHitRating:   bonusHitPercent * HitRatingPerHitChance,
		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		ApplyEffects:     func(*Simulation, *Unit, *Spell) {},
	})
}

// The base spell miss against a target three levels up is 17% (target.go,
// NewAttackTable), 1% of which no hit removes: the cap is 16 points.
func TestSpellHitCapAgainstALevelThreeBossIsSixteenPoints(t *testing.T) {
	caster, boss := spellHitTestUnits(0)
	if got := spellHitCap(caster, boss); math.Abs(got-16) > 1e-9 {
		t.Fatalf("spell hit cap = %v, want 16", got)
	}
}

func TestSpellHitProfileReadsTheStatHit(t *testing.T) {
	caster, boss := spellHitTestUnits(5)
	registerDamageSpell(caster, SpellSchoolFire, 0)

	p, ok := computeSpellHitProfile(caster, boss)
	if !ok || p.Hit != 5 || p.SchoolHit != 5 || p.HasSchoolBonus() {
		t.Fatalf("profile = %+v, ok %v, want hit 5 with no school bonus", p, ok)
	}
	if got := p.ToSpellCap(); math.Abs(got-11) > 1e-9 {
		t.Errorf("ToSpellCap = %v, want 11", got)
	}
	if len(p.SchoolNames) != 0 {
		t.Errorf("school names = %v, want none without a bonus", p.SchoolNames)
	}
}

func TestSpellHitProfileSplitsTheSchoolBonus(t *testing.T) {
	caster, boss := spellHitTestUnits(4)
	registerDamageSpell(caster, SpellSchoolFire, 3)
	registerDamageSpell(caster, SpellSchoolFrost, 3)
	registerDamageSpell(caster, SpellSchoolArcane, 0)

	p, _ := computeSpellHitProfile(caster, boss)
	if p.Hit != 4 || p.SchoolHit != 7 || !p.HasSchoolBonus() {
		t.Fatalf("profile = %+v, want hit 4, school hit 7", p)
	}
	if p.ToSpellCap() != 12 || p.ToSchoolCap() != 9 {
		t.Errorf("to cap = %v / school %v, want 12 / 9", p.ToSpellCap(), p.ToSchoolCap())
	}
	if len(p.SchoolNames) != 2 || p.SchoolNames[0] != "Fire" || p.SchoolNames[1] != "Frost" {
		t.Errorf("school names = %v, want [Fire Frost]", p.SchoolNames)
	}
}

func TestSpellHitDistanceNeverGoesNegative(t *testing.T) {
	caster, boss := spellHitTestUnits(14)
	registerDamageSpell(caster, SpellSchoolShadow, 5)
	registerDamageSpell(caster, SpellSchoolFire, 0)

	p, _ := computeSpellHitProfile(caster, boss)
	if p.ToSpellCap() != 2 || p.ToSchoolCap() != 0 {
		t.Errorf("to cap = %v / school %v, want 2 / 0", p.ToSpellCap(), p.ToSchoolCap())
	}
}

func TestSpellHitProfileIgnoresSpellsThatAreNotCastDamage(t *testing.T) {
	caster, boss := spellHitTestUnits(2)
	if _, ok := computeSpellHitProfile(caster, boss); ok {
		t.Fatal("a unit with no spells has no spell profile")
	}
	caster.RegisterSpell(SpellConfig{
		ActionID: ActionID{SpellID: 1}, SpellSchool: SpellSchoolPhysical, ProcMask: ProcMaskMeleeMHSpecial,
		Flags: SpellFlagAPL, DamageMultiplier: 1, ThreatMultiplier: 1, ApplyEffects: func(*Simulation, *Unit, *Spell) {},
	})
	caster.RegisterSpell(SpellConfig{
		ActionID: ActionID{SpellID: 2}, SpellSchool: SpellSchoolFire, ProcMask: ProcMaskSpellProc,
		DamageMultiplier: 1, ThreatMultiplier: 1, ApplyEffects: func(*Simulation, *Unit, *Spell) {},
	})
	if _, ok := computeSpellHitProfile(caster, boss); ok {
		t.Fatal("a weapon special and a proc are not cast spell damage")
	}
}

func TestSpellMissFloorIsOnePercent(t *testing.T) {
	if minSpellMissChance != 0.01 {
		t.Fatalf("residual spell miss = %v, want 1%%", minSpellMissChance)
	}
}
