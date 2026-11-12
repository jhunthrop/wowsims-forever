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
//
// With both up it heals for the larger of the two full durations. In the
// launch logs (foreverlogs.gg reports 2823 and 2838, levels 22 to 27, read
// 2026-11-11) every one of the 15 decidable non-critical Swiftmends with
// both effects up matched the bigger total: 11 the Rejuvenation's four
// ticks (where Regrowth's seven ticks were smaller), 4 the Regrowth's seven
// (where they were bigger); which effect had less time left, or was cast
// later, explains neither set. The beta logs' 16 Regrowth reads were at
// levels where Regrowth's total was the larger.
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
			spell.CalcAndDealHealing(sim, target, hotFullDuration(source), spell.OutcomeHealingCrit)
			druid.cutSwiftmendCooldownPerHoT(sim, target, spell)
		},
	})
}

// hotFullDuration is what a heal over time lands over its whole run: its
// snapshot tick times its tick count.
func hotFullDuration(dot *core.Dot) float64 {
	return dot.SnapshotBaseDamage * dot.SnapshotAttackerMultiplier * float64(dot.NumberOfTicks)
}

// swiftmendSource is the effect on the target Swiftmend heals for: whichever
// of the druid's own Regrowth and Rejuvenation on it heals more over its
// full duration (Regrowth on a tie), or nil when there is neither.
func (druid *Druid) swiftmendSource(_ *core.Simulation, target *core.Unit) *core.Dot {
	var best *core.Dot
	for _, ranks := range [][]*DruidSpell{druid.Regrowth, druid.Rejuvenation} {
		for _, ds := range ranks {
			if ds == nil {
				continue
			}
			dot := ds.Hot(target)
			if dot == nil || !dot.IsActive() {
				continue
			}
			if best == nil || hotFullDuration(dot) > hotFullDuration(best) {
				best = dot
			}
		}
	}
	return best
}

// swiftmendHoTCount is how many of the druid's own heal over time effects
// (Rejuvenation, Regrowth, Wild Growth) are on the target.
func (druid *Druid) swiftmendHoTCount(target *core.Unit) int {
	count := 0
	for _, ranks := range [][]*DruidSpell{druid.Regrowth, druid.Rejuvenation, druid.WildGrowth} {
		for _, ds := range ranks {
			if ds == nil {
				continue
			}
			if dot := ds.Hot(target); dot != nil && dot.IsActive() {
				count++
			}
		}
	}
	return count
}

// cutSwiftmendCooldownPerHoT is Idol of Synthesis (items.go): "Each of your
// heal over time effects on the target reduces the remaining cooldown on
// Swiftmend by 1 sec when you cast it." The cast has already started the
// cooldown (core's cast flow sets it before the effects run), so the cut
// comes straight off its ready time.
func (druid *Druid) cutSwiftmendCooldownPerHoT(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
	if druid.swiftmendCooldownCutPerHoT == 0 {
		return
	}
	cut := time.Duration(druid.swiftmendHoTCount(target)) * druid.swiftmendCooldownCutPerHoT
	spell.CD.Set(max(sim.CurrentTime, spell.CD.ReadyAt()-cut))
}
