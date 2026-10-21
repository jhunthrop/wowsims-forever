package core

import (
	"math"
	"slices"
)

// percentPerFraction converts the engine's hit fractions to percent points.
const percentPerFraction = 100

// SpellHitProfile is where a caster's spell hit stands against its first
// target, in percent points of hit chance.
//
// A spell's hit is the unit's hit stat plus whatever its talents and runes
// add to it, which is per spell (Elemental Precision on fire and frost,
// Arcane Focus on arcane, Suppression on affliction): Hit is the figure
// every damaging spell has, and SchoolHit the best any one of them has.
// The two differ only for a caster whose talents add school hit.
type SpellHitProfile struct {
	// Hit is the player's hit from gear, buffs and the stat-wide bonuses,
	// which every damaging spell gets.
	Hit float64
	// SchoolHit is the highest total hit of any damaging spell the player
	// can cast, at least Hit.
	SchoolHit float64
	// SchoolNames are the schools (for example "Fire") of the spells that
	// reach SchoolHit, empty when SchoolHit equals Hit.
	SchoolNames []string
	// Cap is the hit at which the target's base spell miss is gone down to
	// the residual every spell keeps.
	Cap float64
}

// ToSpellCap is the hit still worth full value to a spell with only the
// stat-wide bonuses.
func (p SpellHitProfile) ToSpellCap() float64 { return math.Max(p.Cap-p.Hit, 0) }

// ToSchoolCap is the hit still worth full value to the spells that carry a
// school bonus, which is less than ToSpellCap by the bonus.
func (p SpellHitProfile) ToSchoolCap() float64 { return math.Max(p.Cap-p.SchoolHit, 0) }

// HasSchoolBonus is true when some spells get more hit than the rest.
func (p SpellHitProfile) HasSchoolBonus() bool { return p.SchoolHit > p.Hit }

var schoolDisplayNames = map[SpellSchool]string{
	SpellSchoolArcane: "Arcane",
	SpellSchoolFire:   "Fire",
	SpellSchoolFrost:  "Frost",
	SpellSchoolHoly:   "Holy",
	SpellSchoolNature: "Nature",
	SpellSchoolShadow: "Shadow",
}

// castsDamageAtTarget is true for a spell the player casts on purpose to
// damage the enemy with a magic hit roll: the APL-castable spells that carry
// spell damage in a non-physical school. Procs and pets are excluded.
func castsDamageAtTarget(spell *Spell) bool {
	return spell.Flags.Matches(SpellFlagAPL) &&
		spell.ProcMask.Matches(ProcMaskSpellDamage) &&
		spell.SpellSchool != SpellSchoolPhysical &&
		spell.SpellSchool != SpellSchoolNone
}

// computeSpellHitProfile reads the hit of every damaging spell the unit
// has against the target, from the engine's own SpellHitChance, and the cap
// from the attack table's base spell miss. ok is false for a unit with no
// such spell.
func computeSpellHitProfile(unit *Unit, target *Unit) (SpellHitProfile, bool) {
	var profile SpellHitProfile
	found := false
	for _, spell := range unit.Spellbook {
		if !castsDamageAtTarget(spell) {
			continue
		}
		hit := spellHitPercent(spell, target)
		if !found {
			profile.Hit, profile.SchoolHit, found = hit, hit, true
		}
		profile.Hit = math.Min(profile.Hit, hit)
		profile.SchoolHit = math.Max(profile.SchoolHit, hit)
	}
	if !found {
		return profile, false
	}
	profile.Cap = spellHitCap(unit, target)
	profile.SchoolNames = schoolsReaching(unit, target, profile)
	return profile, true
}

// hitPercentPrecision rounds away the float noise of converting the
// engine's hit fraction to percent, so equal hits compare equal.
const hitPercentPrecision = 1e6

func spellHitPercent(spell *Spell, target *Unit) float64 {
	return math.Round(spell.SpellHitChance(target)*percentPerFraction*hitPercentPrecision) / hitPercentPrecision
}

// spellHitCap is the hit, in percent points, past which a spell cannot miss
// less: the target's base spell miss less the residual every spell keeps.
func spellHitCap(unit *Unit, target *Unit) float64 {
	table := NewAttackTable(unit, target, nil)
	return (table.BaseSpellMissChance - minSpellMissChance) * percentPerFraction
}

func schoolsReaching(unit *Unit, target *Unit, profile SpellHitProfile) []string {
	if !profile.HasSchoolBonus() {
		return nil
	}
	var names []string
	for _, spell := range unit.Spellbook {
		if !castsDamageAtTarget(spell) || spellHitPercent(spell, target) < profile.SchoolHit {
			continue
		}
		if name, ok := schoolDisplayNames[spell.SpellSchool]; ok && !slices.Contains(names, name) {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names
}
