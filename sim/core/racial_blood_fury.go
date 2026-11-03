package core

import (
	"time"

	"github.com/wowsims/classic/sim/core/stats"
)

// Blood Fury (client 20572): "Increases Attack Power and Spell Power by 10%
// for 15 s", 2 min cooldown, and no global cooldown (SpellCooldowns
// StartRecoveryTime 0). Its three effects are aura 166 (attack power
// percent), 167 (ranged attack power percent) and 317 (spell power
// percent), each with base points 10. Aura 317 carries misc value 0: no
// school mask and no damage-or-healing selector (the same client's other
// 317 row, Totemic Wrath 77747, is a party spell power buff), so the spell
// power effect raises healing as well as spell damage. They are percentages of the unit's
// current totals, so the aura multiplies the stats through the dynamic stat
// dependency system, after strength, agility and every other source have
// been added, and the bonus follows later changes to those totals.
const (
	bloodFurySpellID    int32 = 20572
	bloodFuryDuration         = 15 * time.Second
	bloodFuryCooldown         = 2 * time.Minute
	bloodFuryMultiplier       = 1.10
)

// bloodFuryMultipliedStats are the stats the dependency system can multiply:
// total attack power, ranged attack power, spell power, spell damage and
// healing power. Nothing feeds spell damage from healing power (stats/deps.go),
// so no bonus is multiplied twice.
var bloodFuryMultipliedStats = []stats.Stat{
	stats.AttackPower,
	stats.RangedAttackPower,
	stats.SpellPower,
	stats.SpellDamage,
	stats.HealingPower,
}

// schoolPowerStats are the per-school spell power stats. They sit outside
// the dependency system, so Blood Fury adds its percentage of them as a
// flat bonus taken when the aura is gained.
var schoolPowerStats = []stats.Stat{
	stats.ArcanePower,
	stats.FirePower,
	stats.FrostPower,
	stats.HolyPower,
	stats.NaturePower,
	stats.ShadowPower,
}

func applyBloodFury(character *Character) {
	actionID := ActionID{SpellID: bloodFurySpellID}
	var schoolPowerBonus stats.Stats

	aura := character.RegisterAura(Aura{
		Label:    "Blood Fury",
		ActionID: actionID,
		Duration: bloodFuryDuration,
		OnGain: func(aura *Aura, sim *Simulation) {
			schoolPowerBonus = stats.Stats{}
			for _, stat := range schoolPowerStats {
				schoolPowerBonus[stat] = character.GetStat(stat) * (bloodFuryMultiplier - 1)
			}
			character.AddStatsDynamic(sim, schoolPowerBonus)
		},
		OnExpire: func(aura *Aura, sim *Simulation) {
			character.AddStatsDynamic(sim, schoolPowerBonus.Invert())
		},
	})
	for _, stat := range bloodFuryMultipliedStats {
		aura.AttachStatDependency(character.NewDynamicMultiplyStat(stat, bloodFuryMultiplier))
	}

	spell := character.RegisterSpell(SpellConfig{
		ActionID: actionID,
		Flags:    SpellFlagNoOnCastComplete,
		Cast: CastConfig{
			CD: Cooldown{
				Timer:    character.NewTimer(),
				Duration: bloodFuryCooldown,
			},
		},
		ApplyEffects: func(sim *Simulation, _ *Unit, _ *Spell) {
			aura.Activate(sim)
		},
	})

	character.AddMajorCooldown(MajorCooldown{
		Spell:    spell,
		Type:     CooldownTypeDPS,
		SelfBuff: true,
	})
}
