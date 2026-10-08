package paladin

import (
	"fmt"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

// blessingEffect is what one blessing does to one unit, for an amount.
type blessingEffect struct {
	gain func(sim *core.Simulation, amount float64)
	lose func(sim *core.Simulation, amount float64)
}

// blessingKind is a blessing and its Greater version.
type blessingKind struct {
	label string
	// raidAuraLabel is the aura the raid-buff option puts on a unit for the
	// same blessing; a cast does nothing to a unit that already has it
	// active, so the two never stack. Empty when the raid option leaves no
	// aura to check (Wisdom's raid option is plain mana per five seconds,
	// so enabling it and casting Wisdom together does add both).
	raidAuraLabel string
	// ranked is whether the client numbers the spells (Might, Wisdom).
	ranked  bool
	spells  []blessingSpell
	greater []blessingSpell
	// newEffect builds the effect on a unit; deps and multipliers are made
	// once per unit here.
	newEffect func(target *core.Unit) blessingEffect
}

var blessingKinds = []blessingKind{
	{
		label: "Blessing of Might", raidAuraLabel: "Blessing of Might", ranked: true,
		spells: blessingOfMightSpells, greater: greaterBlessingOfMightSpells,
		newEffect: statsEffect(stats.AttackPower),
	},
	{
		label: "Blessing of Wisdom", ranked: true,
		spells: blessingOfWisdomSpells, greater: greaterBlessingOfWisdomSpells,
		newEffect: statsEffect(stats.MP5),
	},
	{
		label: "Blessing of Kings", raidAuraLabel: "Blessing of Kings",
		spells: blessingOfKingsSpells, greater: greaterBlessingOfKingsSpells,
		newEffect: kingsEffect,
	},
	{
		label:  "Blessing of Salvation",
		spells: blessingOfSalvationSpells, greater: greaterBlessingOfSalvationSpells,
		newEffect: salvationEffect,
	},
}

// statsEffect adds the amount of one stat, as the raid-buff option does.
func statsEffect(stat stats.Stat) func(*core.Unit) blessingEffect {
	return func(target *core.Unit) blessingEffect {
		add := func(sim *core.Simulation, amount float64) {
			target.AddBuildPhaseStatDynamic(sim, stat, amount)
		}
		return blessingEffect{
			gain: add,
			lose: func(sim *core.Simulation, amount float64) { add(sim, -amount) },
		}
	}
}

// kingsEffect raises Stamina, Agility, Strength, Intellect and Spirit by the
// amount percent, the stats the raid-buff Blessing of Kings raises.
func kingsEffect(target *core.Unit) blessingEffect {
	multiplier := 1 + blessingOfKingsStatBonus/100.0
	deps := []*stats.StatDependency{
		target.NewDynamicMultiplyStat(stats.Stamina, multiplier),
		target.NewDynamicMultiplyStat(stats.Agility, multiplier),
		target.NewDynamicMultiplyStat(stats.Strength, multiplier),
		target.NewDynamicMultiplyStat(stats.Intellect, multiplier),
		target.NewDynamicMultiplyStat(stats.Spirit, multiplier),
	}
	return blessingEffect{
		gain: func(sim *core.Simulation, _ float64) {
			for _, dep := range deps {
				target.EnableBuildPhaseStatDep(sim, dep)
			}
		},
		lose: func(sim *core.Simulation, _ float64) {
			for _, dep := range deps {
				target.DisableBuildPhaseStatDep(sim, dep)
			}
		},
	}
}

// salvationEffect lowers the threat the unit generates by the amount percent.
func salvationEffect(target *core.Unit) blessingEffect {
	multiplier := 1 - blessingOfSalvationThreatReduction/100.0
	return blessingEffect{
		gain: func(*core.Simulation, float64) { target.PseudoStats.ThreatMultiplier *= multiplier },
		lose: func(*core.Simulation, float64) { target.PseudoStats.ThreatMultiplier /= multiplier },
	}
}

// registeredBlessing is one paladin's cast-applied blessing: an aura per
// raid unit, and the amount each one currently carries.
type registeredBlessing struct {
	kind    blessingKind
	auras   core.AuraArray
	amounts []float64
}

// registerBlessings registers the blessings the paladin has learned.
func (paladin *Paladin) registerBlessings() {
	for _, kind := range blessingKinds {
		paladin.registerBlessing(kind)
	}
}

func (paladin *Paladin) registerBlessing(kind blessingKind) {
	if !blessingLearned(kind, paladin.Level) {
		return
	}
	blessing := &registeredBlessing{
		kind:    kind,
		amounts: make([]float64, len(paladin.Env.AllUnits)),
	}
	blessing.auras = paladin.NewRaidAuraArray(func(target *core.Unit) *core.Aura {
		return blessing.newAura(paladin, target)
	})
	paladin.registerBlessingCasts(blessing, kind.spells, false)
	paladin.registerBlessingCasts(blessing, kind.greater, true)
}

func blessingLearned(kind blessingKind, level int32) bool {
	return level >= int32(kind.spells[0].level)
}

// newAura registers the blessing's aura on a unit. The label carries the
// caster so two paladins never collide; the action id is the first rank.
func (blessing *registeredBlessing) newAura(paladin *Paladin, target *core.Unit) *core.Aura {
	effect := blessing.kind.newEffect(target)
	applied := 0.0
	return target.RegisterAura(core.Aura{
		Label:    fmt.Sprintf("%s (paladin %d)", blessing.kind.label, paladin.UnitIndex),
		ActionID: core.ActionID{SpellID: blessing.kind.spells[0].spellID},
		Duration: blessingDuration,
		OnGain: func(_ *core.Aura, sim *core.Simulation) {
			applied = blessing.amounts[target.UnitIndex]
			effect.gain(sim, applied)
		},
		OnExpire: func(_ *core.Aura, sim *core.Simulation) {
			effect.lose(sim, applied)
		},
	})
}

func (paladin *Paladin) registerBlessingCasts(blessing *registeredBlessing, spells []blessingSpell, raidWide bool) {
	for i, spell := range spells {
		if paladin.Level < int32(spell.level) {
			break
		}
		rankNumber := 0
		if blessing.kind.ranked {
			rankNumber = i + 1
		}
		paladin.registerBlessingCast(blessing, spell, rankNumber, raidWide)
	}
}

func (paladin *Paladin) registerBlessingCast(blessing *registeredBlessing, blessingSpell blessingSpell, rankNumber int, raidWide bool) {
	spell := paladin.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: blessingSpell.spellID},
		SpellSchool: core.SpellSchoolHoly,
		ProcMask:    core.ProcMaskEmpty,
		Flags:       core.SpellFlagHelpful | core.SpellFlagAPL | core.SpellFlagNoOnCastComplete,

		RequiredLevel: blessingSpell.level,
		Rank:          rankNumber,

		ManaCost: blessingSpell.manaCostOptions(),
		Cast: core.CastConfig{
			DefaultCast: core.Cast{GCD: core.GCDDefault},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, _ *core.Spell) {
			if !raidWide {
				blessing.apply(sim, target, blessingSpell.amount)
				return
			}
			for _, unit := range paladin.Env.AllUnits {
				if unit.Type != core.EnemyUnit {
					blessing.apply(sim, unit, blessingSpell.amount)
				}
			}
		},
	})
	spell.RelatedSelfBuff = blessing.auras.Get(&paladin.Unit)
}

// apply puts the blessing on a unit, replacing a different amount already
// there, and does nothing for a unit the raid-buff option already covers.
func (blessing *registeredBlessing) apply(sim *core.Simulation, target *core.Unit, amount float64) {
	if blessing.coveredByRaidBuff(target) {
		return
	}
	aura := blessing.auras.Get(target)
	if aura.IsActive() && blessing.amounts[target.UnitIndex] != amount {
		aura.Deactivate(sim)
	}
	blessing.amounts[target.UnitIndex] = amount
	aura.Activate(sim)
}

func (blessing *registeredBlessing) coveredByRaidBuff(target *core.Unit) bool {
	if blessing.kind.raidAuraLabel == "" {
		return false
	}
	raidAura := target.GetAura(blessing.kind.raidAuraLabel)
	return raidAura != nil && raidAura.IsActive()
}
