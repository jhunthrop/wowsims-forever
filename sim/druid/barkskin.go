package druid

import (
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

// Barkskin (spell 22812, learned at 44; build 1.60.1.70009): on the global
// cooldown, no cost, a 60 second cooldown, and for 15 seconds a damage-taken
// aura (aura 87) of -20% on the physical school. Usable in every form, so
// the bear casts it without shifting.
const (
	barkskinSpellID        = 22812
	barkskinLearnLevel     = 44
	barkskinDuration       = 15 * time.Second
	barkskinCooldown       = time.Minute
	barkskinPhysicalDamage = 0.8
)

func (druid *Druid) registerBarkskinCD() {
	if int(druid.Level) < barkskinLearnLevel {
		return
	}
	actionID := core.ActionID{SpellID: barkskinSpellID}

	druid.BarkskinAura = druid.RegisterAura(core.Aura{
		Label:    "Barkskin",
		ActionID: actionID,
		Duration: barkskinDuration,
	}).AttachMultiplicativePseudoStatBuff(&druid.PseudoStats.SchoolDamageTakenMultiplier[stats.SchoolIndexPhysical], barkskinPhysicalDamage)

	druid.Barkskin = druid.RegisterSpell(Any, core.SpellConfig{
		ActionID: actionID,
		Flags:    core.SpellFlagAPL,

		RequiredLevel: barkskinLearnLevel,

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			CD: core.Cooldown{
				Timer:    druid.NewTimer(),
				Duration: barkskinCooldown,
			},
			IgnoreHaste: true,
		},
		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			druid.BarkskinAura.Activate(sim)
		},
	})

	druid.AddMajorCooldown(core.MajorCooldown{
		Spell: druid.Barkskin.Spell,
		Type:  core.CooldownTypeSurvival,
	})
}
