package core

import "strconv"

type ShieldConfig struct {
	SelfOnly bool // Set to true to only create the self-shield.

	Spell *Spell

	Aura
}

// Rerpresents an absorption effect, e.g. Power Word: Shield.
type Shield struct {
	Spell *Spell

	// Embed Aura so we can use IsActive/Refresh/etc directly.
	*Aura

	// remaining is how much damage the shield can still absorb.
	remaining float64
}

// Remaining is the damage the shield can still absorb; 0 once it has
// expired or been used up.
func (shield *Shield) Remaining() float64 {
	return shield.remaining
}

func (shield *Shield) Apply(sim *Simulation, shieldAmount float64) {
	caster := shield.Spell.Unit
	target := shield.Aura.Unit
	//attackTable := caster.AttackTables[target.UnitIndex][shield.Spell.CastType]

	// Shields are not affected by healing pseudostats the same way heals are.
	// So we only apply the spell-specific multiplier and shield-specific multiplier.
	shieldAmount *= shield.Spell.DamageMultiplier * caster.PseudoStats.ShieldDealtMultiplier

	shield.Aura.Deactivate(sim)
	shield.Aura.Activate(sim)
	shield.remaining = shieldAmount
	target.activeShields = append(target.activeShields, shield)

	threat := 0.0 // TODO
	shield.Spell.SpellMetrics[target.UnitIndex].TotalThreat += threat
	shield.Spell.SpellMetrics[target.UnitIndex].TotalShielding += shieldAmount
	shield.Spell.SpellMetrics[target.UnitIndex].Hits++

	if sim.Log != nil {
		caster.Log(sim, "%s %s Hit for %0.3f shielding. (Threat: %0.3f)", target.LogLabel(), shield.Spell.ActionID, shieldAmount, threat)
	}
}

func newShield(config Shield) *Shield {
	shield := &Shield{}
	*shield = config

	return shield
}

// AbsorbDamage lets the unit's active shields soak up damage before it
// reaches the health bar and returns what is left over. The absorbed part
// counts as the shielding spell's effective healing: a shield nothing
// hits protects nobody. A shield that is used up falls off.
func (unit *Unit) AbsorbDamage(sim *Simulation, damage float64) float64 {
	for _, shield := range unit.activeShields {
		if damage <= 0 {
			break
		}
		absorbed := min(damage, shield.remaining)
		if absorbed <= 0 {
			continue
		}
		shield.remaining -= absorbed
		damage -= absorbed
		shield.Spell.SpellMetrics[unit.UnitIndex].TotalEffectiveHealing += absorbed
		if shield.remaining <= 0 {
			shield.Aura.Deactivate(sim)
		}
	}
	return damage
}

// dropShield forgets a shield that expired or was used up.
func (unit *Unit) dropShield(shield *Shield) {
	shield.remaining = 0
	kept := unit.activeShields[:0]
	for _, active := range unit.activeShields {
		if active != shield {
			kept = append(kept, active)
		}
	}
	unit.activeShields = kept
}

type ShieldArray []*Shield

func (shields ShieldArray) Get(target *Unit) *Shield {
	return shields[target.UnitIndex]
}

func (spell *Spell) createShields(config ShieldConfig) {
	if config.Aura.Label == "" {
		return
	}

	if config.Spell == nil {
		config.Spell = spell
	}
	shield := Shield{
		Spell: config.Spell,
	}

	auraConfig := config.Aura
	if auraConfig.ActionID.IsEmptyAction() {
		auraConfig.ActionID = shield.Spell.ActionID
	}

	caster := shield.Spell.Unit
	if config.SelfOnly {
		spell.selfShield = registerShield(caster, auraConfig, shield)
	} else {
		auraConfig.Label += "-" + strconv.Itoa(int(caster.UnitIndex))
		if spell.shields == nil {
			spell.shields = make([]*Shield, len(caster.Env.AllUnits))
		}
		for _, target := range caster.Env.AllUnits {
			if !caster.IsOpponent(target) {
				spell.shields[target.UnitIndex] = registerShield(target, auraConfig, shield)
			}
		}
	}
}

// registerShield puts the shield's aura on unit and makes the aura's end
// the shield's end.
func registerShield(unit *Unit, auraConfig Aura, template Shield) *Shield {
	shield := newShield(template)
	userOnExpire := auraConfig.OnExpire
	auraConfig.OnExpire = func(aura *Aura, sim *Simulation) {
		unit.dropShield(shield)
		if userOnExpire != nil {
			userOnExpire(aura, sim)
		}
	}
	shield.Aura = unit.GetOrRegisterAura(auraConfig)
	return shield
}
