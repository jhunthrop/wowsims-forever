package paladin

import "github.com/wowsims/classic/sim/core"

// damageAbsorb is a pool of damage that its owner soaks up before it
// reaches health: Seal of Fury's shield and Templar's Bulwark. The core
// has no absorb of its own (core.Shield only records the grant), so the
// pool is a core.DynamicDamageTakenModifier that spends itself on the
// post-outcome damage of every hit taken, which is also what makes the
// attacker's damage metrics, the tank's damage taken and its TMI read the
// damage that got through.
//
// The pool lives exactly as long as its aura: it is only spent while the
// aura is active, and Grant replaces whatever was left of the last grant
// (an absorb does not stack with itself in the client).
type damageAbsorb struct {
	aura      *core.Aura
	source    *core.Spell // credited with the shielding in the metrics; the spell of the last grant
	remaining float64

	// onDepleted runs when a hit empties the pool, after the pool and its
	// aura are gone.
	onDepleted func(sim *core.Simulation, spell *core.Spell, result *core.SpellResult)
}

// newDamageAbsorb attaches a pool to unit. It must be called during
// construction, before the environment is finalized (core refuses a
// damage-taken modifier added later).
func newDamageAbsorb(unit *core.Unit, aura *core.Aura,
	onDepleted func(*core.Simulation, *core.Spell, *core.SpellResult)) *damageAbsorb {
	absorb := &damageAbsorb{aura: aura, onDepleted: onDepleted}
	unit.AddDynamicDamageTakenModifier(absorb.soak(unit))
	return absorb
}

// Grant starts (or restarts) the aura with a pool of amount, credited to
// source in the metrics.
func (absorb *damageAbsorb) Grant(sim *core.Simulation, source *core.Spell, amount float64) {
	absorb.source = source
	absorb.remaining = amount
	absorb.aura.Activate(sim)
}

// Remaining is what is left of the pool, 0 once the aura has ended.
func (absorb *damageAbsorb) Remaining() float64 {
	if !absorb.aura.IsActive() {
		return 0
	}
	return absorb.remaining
}

func (absorb *damageAbsorb) soak(unit *core.Unit) core.DynamicDamageTakenModifier {
	return func(sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
		if result.Damage <= 0 || absorb.Remaining() <= 0 {
			return
		}

		soaked := min(result.Damage, absorb.remaining)
		result.Damage -= soaked
		absorb.remaining -= soaked
		absorb.source.SpellMetrics[unit.UnitIndex].TotalShielding += soaked

		if absorb.remaining > 0 {
			return
		}
		absorb.aura.Deactivate(sim)
		if absorb.onDepleted != nil {
			absorb.onDepleted(sim, spell, result)
		}
	}
}
