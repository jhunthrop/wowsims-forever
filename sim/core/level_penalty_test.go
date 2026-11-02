package core

import (
	"math"
	"testing"
	"time"
)

// A spell learned below level 20 gains less from spell power: its bonus
// coefficient is scaled by 1 - (20 - level) * 0.0375 (cmangos-classic
// Unit::CalculateLevelPenalty, vmangos SpellCaster::CalculateLevelPenalty,
// applied to direct and periodic damage and healing alike). The client's
// coefficients carry no penalty of their own: Healing Touch rank 1 (5185,
// level 1) is 0.429, its cast time's 1.5 / 3.5.
func TestSpellsLearnedBelowTwentyGainLessFromSpellPower(t *testing.T) {
	_, caster, _ := newPeriodicCritFixture(t, 0)
	register := func(id int32, level int, school SpellSchool) *Spell {
		return caster.RegisterSpell(SpellConfig{
			ActionID:         ActionID{SpellID: id},
			SpellSchool:      school,
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
