package core

import (
	"math"
	"testing"
	"time"
)

// A damage spell learned below level 20 gains less from spell power: its
// bonus coefficient is scaled by 1 - (20 - level) * 0.0375 (cmangos-classic
// Unit::CalculateLevelPenalty, vmangos SpellCaster::CalculateLevelPenalty).
// The client's coefficients carry no penalty of their own: Healing Touch
// rank 1 (5185, level 1) is 0.429, its cast time's 1.5 / 3.5.
func TestSpellsLearnedBelowTwentyGainLessFromSpellPower(t *testing.T) {
	_, caster, _ := newPeriodicCritFixture(t, 0)
	register := func(id int32, level int, school SpellSchool) *Spell {
		return caster.RegisterSpell(SpellConfig{
			ActionID:         ActionID{SpellID: id},
			SpellSchool:      school,
			DefenseType:      DefenseTypeMagic,
			ProcMask:         ProcMaskSpellDamage,
			RequiredLevel:    level,
			ManaCost:         ManaCostOptions{FlatCost: 50},
			Cast:             CastConfig{DefaultCast: Cast{GCD: GCDDefault}},
			DamageMultiplier: 1,
			ThreatMultiplier: 1,
			BonusCoefficient: 0.8,
			Hot: DotConfig{
				Aura:             Aura{Label: "Level Penalty Test " + string(rune('A'+id%26))},
				NumberOfTicks:    4,
				TickLength:       3 * time.Second,
				BonusCoefficient: 0.2,
				OnTick:           func(*Simulation, *Unit, *Dot) {},
			},
		})
	}
	for _, c := range []struct {
		id     int32
		level  int
		school SpellSchool
		factor float64
	}{
		{1, 16, SpellSchoolNature, 0.85},
		{2, 1, SpellSchoolHoly, 0.2875},
		{3, 20, SpellSchoolNature, 1},
		{4, 0, SpellSchoolNature, 1},
		{5, 10, SpellSchoolPhysical, 1},
	} {
		spell := register(c.id, c.level, c.school)
		if got, want := spell.BonusCoefficient, 0.8*c.factor; math.Abs(got-want) > 1e-9 {
			t.Errorf("level %d %v spell: direct coefficient %.4f, want %.4f", c.level, c.school, got, want)
		}
		if got, want := spell.Hot(caster).BonusCoefficient, 0.2*c.factor; math.Abs(got-want) > 1e-9 {
			t.Errorf("level %d %v spell: periodic coefficient %.4f, want %.4f", c.level, c.school, got, want)
		}
	}
}

// A heal keeps its whole coefficient at any learn level. Forever's beta logs
// (foreverlogs.gg, levels 27 to 35) fit no sub-20 penalty on heals: a
// druid's Rejuvenation ranks 3 (level 16) and 5 (28), two druids, tick
// 59.6 and 81.7, and 60.8 and 83.7, which is the same healing power under
// one multiplier (1.16 and 1.20) only with the whole 0.2; with the penalty
// the multiplier falls to 0.77 and 0.81, below any talent's. A druid's
// Healing Touch ranks 2 and 5, a shaman's Healing Wave ranks 1 and 4 and
// two paladins' Holy Light ranks 3 to 5 say the same (multipliers 0.50 to
// 0.67 under the penalty).
func TestAHealLearnedBelowTwentyKeepsItsCoefficient(t *testing.T) {
	_, caster, _ := newPeriodicCritFixture(t, 0)
	for _, level := range []int{1, 16} {
		heal := caster.RegisterSpell(SpellConfig{
			ActionID:         ActionID{SpellID: int32(100 + level)},
			SpellSchool:      SpellSchoolNature,
			DefenseType:      DefenseTypeMagic,
			Flags:            SpellFlagHelpful,
			ProcMask:         ProcMaskSpellHealing,
			RequiredLevel:    level,
			ManaCost:         ManaCostOptions{FlatCost: 50},
			Cast:             CastConfig{DefaultCast: Cast{GCD: GCDDefault}},
			DamageMultiplier: 1,
			ThreatMultiplier: 1,
			BonusCoefficient: 0.8,
			Hot: DotConfig{
				Aura:             Aura{Label: "Heal Level Test " + string(rune('A'+level))},
				NumberOfTicks:    4,
				TickLength:       3 * time.Second,
				BonusCoefficient: 0.2,
				OnTick:           func(*Simulation, *Unit, *Dot) {},
			},
		})
		if heal.BonusCoefficient != 0.8 {
			t.Errorf("a heal learned at %d: direct coefficient %.4f, want 0.8", level, heal.BonusCoefficient)
		}
		if got := heal.Hot(caster).BonusCoefficient; got != 0.2 {
			t.Errorf("a heal learned at %d: periodic coefficient %.4f, want 0.2", level, got)
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
