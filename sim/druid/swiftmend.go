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
// has the talent: "Instantly heals a target with your active Rejuvenation
// or Regrowth effect for an amount equal to the full duration of the
// periodic effect of the consumed spell."
//
// The heal is what the effect's snapshot says it would have landed in all:
// its tick times its tick count, already carrying the caster's healing
// multipliers, so the spell skips them a second time. In the beta logs
// (foreverlogs.gg reports 2747, 2752 and 2760, levels 25 and 35, read
// 2026-11-02) every non-critical Swiftmend is exactly 4 Rejuvenation or 7
// Regrowth ticks of the effect on the target.
//
// Forever does not consume the effect, whatever the text says: 55 of the 57
// effects up at a logged Swiftmend ticked on to their natural end, and the
// client row has one effect, the heal, with nothing that removes an aura.
// With both up, all 16 logged Swiftmends healed for the Regrowth, whichever
// had less time left.
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
		Flags:          core.SpellFlagHelpful | core.SpellFlagAPL | core.SpellFlagIgnoreAttackerModifiers | SpellFlagOmen,

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
		},
	})
}

// swiftmendSource is the effect on the target Swiftmend heals for: the
// druid's Regrowth when one is up, else its Rejuvenation, or nil when
// there is neither.
func (druid *Druid) swiftmendSource(_ *core.Simulation, target *core.Unit) *core.Dot {
	for _, ranks := range [][]*DruidSpell{druid.Regrowth, druid.Rejuvenation} {
		for _, ds := range ranks {
			if ds == nil {
				continue
			}
			if dot := ds.Hot(target); dot != nil && dot.IsActive() {
				return dot
			}
		}
	}
	return nil
}
