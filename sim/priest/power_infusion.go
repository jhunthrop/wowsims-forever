package priest

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

const (
	powerInfusionSpellID   = 10060
	powerInfusionManaPct   = 0.20
	powerInfusionCooldown  = 3 * time.Minute
	powerInfusionDuration  = 15 * time.Second
	powerInfusionDoneBonus = 1.2
)

// RegisterPowerInfusion registers Power Infusion on target, the ally the
// player picked, as a major cooldown the rotation's autocast fires. It
// does nothing without the talent or a friendly target. The client's
// effect is +20% spell damage and healing done for 15 s (the old mana
// discount is gone).
func (priest *Priest) RegisterPowerInfusion(target *core.Unit) {
	if !priest.Talents.PowerInfusion || target == nil || priest.IsOpponent(target) {
		return
	}
	actionID := core.ActionID{SpellID: powerInfusionSpellID, Tag: priest.Index}
	aura := target.GetOrRegisterAura(core.Aura{
		Label:    "Power Infusion-" + actionID.String(),
		ActionID: actionID,
		Duration: powerInfusionDuration,
		OnGain: func(aura *core.Aura, _ *core.Simulation) {
			aura.Unit.PseudoStats.HealingDealtMultiplier *= powerInfusionDoneBonus
			aura.Unit.PseudoStats.SchoolDamageDealtMultiplier.MultiplyMagicSchools(powerInfusionDoneBonus)
		},
		OnExpire: func(aura *core.Aura, _ *core.Simulation) {
			aura.Unit.PseudoStats.HealingDealtMultiplier /= powerInfusionDoneBonus
			aura.Unit.PseudoStats.SchoolDamageDealtMultiplier.MultiplyMagicSchools(1 / powerInfusionDoneBonus)
		},
	})

	priest.PowerInfusion = priest.RegisterSpell(core.SpellConfig{
		ActionID:  actionID,
		SpellCode: SpellCode_PriestPowerInfusion,
		Flags:     SpellFlagPriest | core.SpellFlagHelpful | core.SpellFlagNoOnCastComplete | core.SpellFlagAPL,

		ManaCost: core.ManaCostOptions{BaseCost: powerInfusionManaPct},
		Cast: core.CastConfig{
			CD: core.Cooldown{Timer: priest.NewTimer(), Duration: powerInfusionCooldown},
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			aura.Activate(sim)
		},
	})

	priest.AddMajorCooldown(core.MajorCooldown{
		Spell:    priest.PowerInfusion,
		Priority: core.CooldownPriorityBloodlust,
		Type:     core.CooldownTypeDPS,
	})
}
