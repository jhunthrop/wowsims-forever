package druid

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

const (
	swiftmendCooldown = 15 * time.Second
	// swiftmendManaCost is the client's 20% of base mana (spellconst cost_pct).
	swiftmendManaCost = 0.20
	// swiftmendClientPlaceholder is the one point the client states for the
	// spell's heal effect: the amount is scripted from the effect it eats,
	// so the figure only keeps the conformance report honest about it.
	swiftmendClientPlaceholder = 1
)

// registerSwiftmendSpell registers Swiftmend (spell 18562), when the druid
// has the talent: "Instantly heals a target with an active Rejuvenation or
// Regrowth effect for an amount equal to the full duration of the periodic
// effect of one of those spells", which is consumed.
//
// The heal is what the effect's snapshot says it would have landed in all:
// its tick times its tick count, already carrying the caster's healing
// multipliers, so the spell skips them a second time. Which effect it eats
// when both are up is not stated; the one with the least time left goes, so
// the longer, more valuable one stays.
func (druid *Druid) registerSwiftmendSpell() {
	if !druid.Talents.Swiftmend {
		return
	}

	druid.Swiftmend = druid.RegisterSpell(Humanoid, core.SpellConfig{
		ActionID:       core.ActionID{SpellID: 18562},
		SpellCode:      SpellCode_DruidSwiftmend,
		ClassSpellMask: DruidSpellMaskSwiftmend,
		SpellSchool:    core.SpellSchoolNature,
		DefenseType:    core.DefenseTypeMagic,
		ProcMask:       core.ProcMaskSpellHealing,
		Flags:          core.SpellFlagHelpful | core.SpellFlagAPL | core.SpellFlagIgnoreAttackerModifiers,

		RequiredLevel: 1,

		ManaCost: core.ManaCostOptions{BaseCost: swiftmendManaCost},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{GCD: core.GCDDefault},
			CD:          core.Cooldown{Timer: druid.NewTimer(), Duration: swiftmendCooldown},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		ClientBaseDamage: [2]float64{swiftmendClientPlaceholder, swiftmendClientPlaceholder},

		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return druid.swiftmendSource(sim, target) != nil
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			source := druid.swiftmendSource(sim, target)
			total := source.SnapshotBaseDamage * source.SnapshotAttackerMultiplier * float64(source.NumberOfTicks)
			spell.CalcAndDealHealing(sim, target, total, spell.OutcomeHealingCrit)
			source.Cancel(sim)
		},
	})
}

// swiftmendSource is the active Rejuvenation or Regrowth on the target that
// Swiftmend would consume, or nil when there is none.
func (druid *Druid) swiftmendSource(sim *core.Simulation, target *core.Unit) *core.Dot {
	var source *core.Dot
	for _, ranks := range [][]*DruidSpell{druid.Rejuvenation, druid.Regrowth} {
		for _, ds := range ranks {
			if ds == nil {
				continue
			}
			dot := ds.Hot(target)
			if dot != nil && dot.IsActive() && (source == nil || dot.RemainingDuration(sim) < source.RemainingDuration(sim)) {
				source = dot
			}
		}
	}
	return source
}
