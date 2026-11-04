package core

import (
	"math"
	"testing"
	"time"
)

// Forever's level-gap rule replaces Era's sub-20 cut: a rank keeps its whole
// spell power coefficient until the caster is 17 levels past the level it is
// learned at, then loses 5% of it a level, and keeps none 37 levels past it
// (share = clamp(1 - (L - T - 17) * 0.05, 0, 1)). Blizzard's 1 October beta
// note and the character sheet: "spells cast using spell ranks vastly below
// your current level receive less benefit from your Spell Damage and Spell
// Healing"; the numbers are ForeverChanges' beta measurement on heals,
// levels 18 to 30 (https://foreverchanges.pro/downranking, 2026-10-16).
// Heals and damage spells take the same rule (the note names both stats).
func TestLevelPenaltyIsForeversLevelGap(t *testing.T) {
	for _, c := range []struct {
		rank, caster int
		want         float64
	}{
		{43, 60, 1},
		{42, 60, 0.95},
		{40, 60, 0.85},
		{30, 60, 0.35},
		{23, 60, 0},
		{16, 60, 0},
		{1, 60, 0},
		{16, 30, 1},
		{13, 30, 1},
		{12, 30, 0.95},
		{1, 30, 0.4},
		{4, 30, 0.55},
		{0, 60, 1},
		{10, 0, 1},
	} {
		if got := LevelPenalty(c.rank, c.caster); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("rank learned at %d, caster level %d: share %.4f, want %.4f", c.rank, c.caster, got, c.want)
		}
	}
}

// RegisterSpell scales a cast rank's direct and periodic coefficients by the
// share, for heals and damage alike, at the caster's own level.
func TestCastRanksKeepTheLevelGapShare(t *testing.T) {
	_, caster, _ := newPeriodicCritFixture(t, 0)
	register := func(id int32, level int, school SpellSchool, heal bool) *Spell {
		procMask, flags := ProcMaskSpellDamage, SpellFlag(0)
		if heal {
			procMask, flags = ProcMaskSpellHealing, SpellFlagHelpful
		}
		return caster.RegisterSpell(SpellConfig{
			ActionID:         ActionID{SpellID: id},
			SpellSchool:      school,
			DefenseType:      DefenseTypeMagic,
			Flags:            flags,
			ProcMask:         procMask,
			RequiredLevel:    level,
			ManaCost:         ManaCostOptions{FlatCost: 50},
			Cast:             CastConfig{DefaultCast: Cast{GCD: GCDDefault}},
			DamageMultiplier: 1,
			ThreatMultiplier: 1,
			BonusCoefficient: 0.8,
			Hot: DotConfig{
				Aura:             Aura{Label: "Level Gap Test " + string(rune('A'+id%26))},
				NumberOfTicks:    4,
				TickLength:       3 * time.Second,
				BonusCoefficient: 0.2,
				OnTick:           func(*Simulation, *Unit, *Dot) {},
			},
		})
	}
	for _, c := range []struct {
		id          int32
		level       int
		casterLevel int32
		school      SpellSchool
		heal        bool
		factor      float64
	}{
		{1, 40, 60, SpellSchoolNature, true, 0.85},
		{2, 16, 60, SpellSchoolHoly, true, 0},
		{3, 43, 60, SpellSchoolNature, false, 1},
		{4, 30, 60, SpellSchoolShadow, false, 0.35},
		{5, 0, 60, SpellSchoolNature, true, 1},
		{6, 10, 60, SpellSchoolPhysical, false, 1},
		{7, 12, 30, SpellSchoolNature, true, 0.95},
		{8, 16, 30, SpellSchoolShadow, false, 1},
	} {
		caster.Level = c.casterLevel
		spell := register(c.id, c.level, c.school, c.heal)
		if got, want := spell.BonusCoefficient, 0.8*c.factor; math.Abs(got-want) > 1e-9 {
			t.Errorf("rank %d at caster %d (%v, heal %v): direct coefficient %.4f, want %.4f", c.level, c.casterLevel, c.school, c.heal, got, want)
		}
		if got, want := spell.Hot(caster).BonusCoefficient, 0.2*c.factor; math.Abs(got-want) > 1e-9 {
			t.Errorf("rank %d at caster %d (%v, heal %v): periodic coefficient %.4f, want %.4f", c.level, c.casterLevel, c.school, c.heal, got, want)
		}
	}
}

// A proc carries its client spell level, which is not a learned rank:
// every Seal of Righteousness and Seal of Fury proc is level 1. A spell
// with no cost and no cast keeps its coefficient.
func TestAProcKeepsItsCoefficientAtAnyLevel(t *testing.T) {
	_, caster, _ := newPeriodicCritFixture(t, 0)
	proc := caster.RegisterSpell(SpellConfig{
		ActionID:         ActionID{SpellID: 20415},
		SpellSchool:      SpellSchoolHoly,
		DefenseType:      DefenseTypeMelee,
		ProcMask:         ProcMaskMeleeMHSpecial,
		RequiredLevel:    1,
		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: 0.1,
	})
	if proc.BonusCoefficient != 0.1 {
		t.Errorf("a level-1 proc's coefficient is %v, want its client 0.1", proc.BonusCoefficient)
	}
}

// A single-rank spell whose client amount grows every level up to the
// caster's level (SpellLevels MaxLevel at or above it, as Wrack's 40 to 60)
// is not a rank below the caster: it keeps its whole coefficient.
func TestASpellThatScalesToTheCasterKeepsItsCoefficient(t *testing.T) {
	_, caster, _ := newPeriodicCritFixture(t, 0)
	spell := caster.RegisterSpell(SpellConfig{
		ActionID:            ActionID{SpellID: 1316697},
		SpellSchool:         SpellSchoolShadow,
		DefenseType:         DefenseTypeMagic,
		ProcMask:            ProcMaskSpellDamage,
		RequiredLevel:       40,
		ScalesToCasterLevel: true,
		ManaCost:            ManaCostOptions{FlatCost: 50},
		Cast:                CastConfig{DefaultCast: Cast{GCD: GCDDefault}},
		DamageMultiplier:    1,
		ThreatMultiplier:    1,
		BonusCoefficient:    0.286,
	})
	if spell.BonusCoefficient != 0.286 {
		t.Errorf("a spell that scales to the caster's level: coefficient %v, want 0.286", spell.BonusCoefficient)
	}
}

// A heal another cast triggers (Holy Shock's heal half) carries that cast
// rank's learn level and is that rank, so it takes the cast rank's share
// though it has no cost or cast of its own.
func TestATriggeredRankTakesItsCastRanksShare(t *testing.T) {
	_, caster, _ := newPeriodicCritFixture(t, 0)
	heal := caster.RegisterSpell(SpellConfig{
		ActionID:         ActionID{SpellID: 1311605},
		SpellSchool:      SpellSchoolHoly,
		DefenseType:      DefenseTypeMagic,
		Flags:            SpellFlagHelpful,
		ProcMask:         ProcMaskSpellHealing,
		RequiredLevel:    30,
		TriggeredRank:    true,
		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: 0.429,
	})
	if got, want := heal.BonusCoefficient, 0.429*0.35; math.Abs(got-want) > 1e-9 {
		t.Errorf("a triggered rank learned at 30, caster 60: coefficient %v, want %v", got, want)
	}
}
