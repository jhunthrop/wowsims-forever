package priest

import (
	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

// bindingHealRanks are trainables "Binding Heal" (401937, 1240770-1240774),
// which client 1.60.1.70707's hotfix push teaches from the Holy skill line
// at 25, 32, 38, 44, 50 and 56 (the 30 October hotfix turned the old Binding
// Heal talent into Holy Binding). Each rank's two effects carry the same
// amount, coefficient and variance as the Flash Heal of its level; Wowhead
// Forever's tooltip for rank 6 (read 2026-11-01): "Heal yourself and a
// friendly target for 778 to 914", 380 mana, 1.5 sec cast.
var bindingHealRanks = []healRank{
	{spellID: 401937, level: 25, manaCost: 155, castMS: 1500, coefficient: 0.429,
		effect: clientdamage.Effect{Amount: 247, Variance: 0.195804, PerLevel: 2.2, SpellLevel: 25, MaxLevel: 31}},
	{spellID: 1240770, level: 32, manaCost: 185, castMS: 1500, coefficient: 0.429,
		effect: clientdamage.Effect{Amount: 312, Variance: 0.183333, PerLevel: 2.5, SpellLevel: 32, MaxLevel: 37}},
	{spellID: 1240771, level: 38, manaCost: 215, castMS: 1500, coefficient: 0.429,
		effect: clientdamage.Effect{Amount: 383, Variance: 0.177677, PerLevel: 2.8, SpellLevel: 38, MaxLevel: 43}},
	{spellID: 1240772, level: 44, manaCost: 265, castMS: 1500, coefficient: 0.429,
		effect: clientdamage.Effect{Amount: 507, Variance: 0.17284, PerLevel: 3.3, SpellLevel: 44, MaxLevel: 49}},
	{spellID: 1240773, level: 50, manaCost: 315, castMS: 1500, coefficient: 0.429,
		effect: clientdamage.Effect{Amount: 644, Variance: 0.170455, PerLevel: 3.7, SpellLevel: 50, MaxLevel: 55}},
	{spellID: 1240774, level: 56, manaCost: 380, castMS: 1500, coefficient: 0.429,
		effect: clientdamage.Effect{Amount: 825, Variance: 0.164972, PerLevel: 4.2, SpellLevel: 56, MaxLevel: 61}},
}

// registerBindingHeal heals the target and the priest, each half with its
// own roll and its own crit. Cast on the priest, it heals the priest once.
// unconfirmed: whether the two halves roll one crit or two (two, as two
// effects of one spell do here elsewhere).
func (priest *Priest) registerBindingHeal() {
	priest.BindingHeal = priest.registerHealRanks(bindingHealRanks, func(rank int, entry healRank) core.SpellConfig {
		config := priest.healSpellConfig(entry, rank, SpellCode_PriestBindingHeal, PriestSpellMaskBindingHeal)
		config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			priest.healTarget(sim, spell, target, priest.roll(sim, entry))
			if target != &priest.Unit {
				priest.healTarget(sim, spell, &priest.Unit, priest.roll(sim, entry))
			}
		}
		return config
	})
}
