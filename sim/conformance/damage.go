package conformance

import (
	"fmt"
	"math"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/spellconst"
)

// The damage comparison. The client states a direct-damage or periodic
// effect as a centre (EffectBasePointsF), a Variance (the roll runs from
// centre x (1 - Variance/2) to centre x (1 + Variance/2)) and
// EffectRealPointsPerLevel (added per caster level above the spell's own,
// up to SpellLevels.MaxLevel); spellconst.Spell.DamageRange resolves all
// of that at the preset's level. The engine has no uniform accessor for
// "the base damage this ability rolls" (each ability file keeps its own
// table and rolls it inline), so an ability file MAY declare it through
// core.SpellConfig.ClientBaseDamage and this report reads that. A spell
// that does not declare is reported "not declared", never "match": the
// point of the column is to make the gap visible, not to flatter it.

// Damage statuses, one per row.
const (
	// DamageNone: the client has no direct-damage or periodic-damage effect
	// on this spell, so there is nothing to compare.
	DamageNone = "n/a"
	// DamageUndeclared: the client has one and the ability file declares no
	// ClientBaseDamage.
	DamageUndeclared = "not declared"
	// DamageMatches: declared, and the range (and the coefficient, when the
	// client table states one) agree.
	DamageMatches = "declared, matches"
	// DamageDiffers: declared, and the range or the coefficient disagree.
	DamageDiffers = "declared, differs"
)

const (
	// damageTolerance is how far each end of the range may sit from the
	// client's float before it counts as a difference: the engine tables
	// are whole numbers and the client's centre x variance is not.
	damageTolerance = 1.0
	// coefficientTolerance covers the client's float32 coefficients
	// (0.71399998665 for 0.714).
	coefficientTolerance = 0.005

	// Client effect and aura codes (SpellEffect.Effect / EffectAura).
	effectSchoolDamage = 2
	effectApplyAura    = 6
	effectHeal         = 10
	auraPeriodicDamage = 3
	auraPeriodicHeal   = 8
)

// DamageComparison is the damage half of a Row.
type DamageComparison struct {
	Status string

	ClientMin, ClientMax float64
	// ClientCoefficient is the effect's spell-power share; ClientCoefficientFromTable
	// is false when the client table states 0 and the value is spellconst's
	// vanilla convention, which is not compared.
	ClientCoefficient          float64
	ClientCoefficientFromTable bool

	EngineMin, EngineMax float64
	EngineCoefficient    float64

	// Diff names what differed, empty unless Status is DamageDiffers.
	Diff string
}

// clientDamageEffect is the effect the comparison reads: the spell's
// school-damage effect, else its periodic-damage aura, else its heal
// effect, else its periodic-heal aura. A heal is compared exactly as a
// damage roll is: the client's centre, variance and per-level growth
// against the {min, max} the ability file declares in
// core.SpellConfig.ClientBaseDamage (which for a heal is its base
// healing), and the spell-power coefficient against BonusCoefficient. It
// reports false for a spell with none of them (a buff, a pure utility) and
// for a heal whose client amount is zero (Lay on Hands heals the caster's
// whole health bar, a number no table row states).
func clientDamageEffect(spell spellconst.Spell) (spellconst.Effect, bool) {
	for _, e := range spell.Effects {
		if e.Effect == effectSchoolDamage {
			return e, true
		}
	}
	for _, e := range spell.Effects {
		if e.Effect == effectApplyAura && e.Aura == auraPeriodicDamage {
			return e, true
		}
	}
	for _, e := range spell.Effects {
		if e.Effect == effectHeal && e.Amount > 0 {
			return e, true
		}
	}
	for _, e := range spell.Effects {
		if e.Effect == effectApplyAura && e.Aura == auraPeriodicHeal && e.Amount > 0 {
			return e, true
		}
	}
	return spellconst.Effect{}, false
}

// compareDamage compares one engine spell's declared base damage against
// the client's at casterLevel.
func compareDamage(clientSpell spellconst.Spell, casterLevel int, engine *core.Spell) DamageComparison {
	effect, ok := clientDamageEffect(clientSpell)
	if !ok {
		return DamageComparison{Status: DamageNone}
	}
	minDamage, maxDamage, _ := clientSpell.DamageRange(effect.Index, casterLevel)
	result := DamageComparison{
		ClientMin:                  minDamage,
		ClientMax:                  maxDamage,
		ClientCoefficient:          effect.ResolvedSPCoefficient,
		ClientCoefficientFromTable: effect.CoefficientSource == "table",
		EngineMin:                  engine.ClientBaseDamage[0],
		EngineMax:                  engine.ClientBaseDamage[1],
		EngineCoefficient:          engine.BonusCoefficient,
	}
	if engine.ClientBaseDamage == [2]float64{} {
		result.Status = DamageUndeclared
		return result
	}
	result.Diff = damageDiff(result)
	result.Status = DamageMatches
	if result.Diff != "" {
		result.Status = DamageDiffers
	}
	return result
}

func damageDiff(c DamageComparison) string {
	var diffs []string
	if math.Abs(c.ClientMin-c.EngineMin) > damageTolerance || math.Abs(c.ClientMax-c.EngineMax) > damageTolerance {
		diffs = append(diffs, fmt.Sprintf("damage %.0f-%.0f->%.0f-%.0f", c.ClientMin, c.ClientMax, c.EngineMin, c.EngineMax))
	}
	if c.ClientCoefficientFromTable && math.Abs(c.ClientCoefficient-c.EngineCoefficient) > coefficientTolerance {
		diffs = append(diffs, fmt.Sprintf("coefficient %.3f->%.3f", c.ClientCoefficient, c.EngineCoefficient))
	}
	return joinDiffs(diffs)
}

// damageCells renders a comparison as the three golden cells: range,
// coefficient, status (with the diff when there is one).
func damageCells(c DamageComparison) (rangeCell, coefficientCell, statusCell string) {
	if c.Status == DamageNone {
		return "n/a", "n/a", DamageNone
	}
	engineRange := "-"
	if c.Status != DamageUndeclared {
		engineRange = fmt.Sprintf("%.2f-%.2f", c.EngineMin, c.EngineMax)
	}
	coefficientCell = fmt.Sprintf("%.3f→%.3f", c.ClientCoefficient, c.EngineCoefficient)
	if !c.ClientCoefficientFromTable {
		coefficientCell = fmt.Sprintf("%.3f (convention)→%.3f", c.ClientCoefficient, c.EngineCoefficient)
	}
	statusCell = c.Status
	if c.Diff != "" {
		statusCell += ": " + c.Diff
	}
	return fmt.Sprintf("%.2f-%.2f→%s", c.ClientMin, c.ClientMax, engineRange), coefficientCell, statusCell
}

// DamageCounts tallies a class's rows by damage status.
type DamageCounts struct {
	Declared, Matching, Differing, Undeclared, NotApplicable int
}

// summaryLevel is the level the SUMMARY counts are taken at, the one every
// preset reaches (the same convention the rest of SUMMARY.md uses).
const summaryLevel = 60

// countDamage tallies the rows built at summaryLevel. A row is one spell
// rank of one spec, so a spell two specs both register counts twice, as
// the rest of the report does.
func countDamage(rows []Row) DamageCounts {
	var counts DamageCounts
	for _, r := range rows {
		if r.Level != summaryLevel {
			continue
		}
		switch r.Damage.Status {
		case DamageNone:
			counts.NotApplicable++
		case DamageUndeclared:
			counts.Undeclared++
		case DamageMatches:
			counts.Declared++
			counts.Matching++
		case DamageDiffers:
			counts.Declared++
			counts.Differing++
		}
	}
	return counts
}
