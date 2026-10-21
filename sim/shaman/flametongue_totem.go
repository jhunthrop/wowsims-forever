package shaman

import (
	"github.com/wowsims/classic/sim/core"
)

// Flametongue Totem's client rows are quoted in sim/core/flametongue_totem_test.go;
// the effect itself is core.FlametongueTotemAura, shared with the raid buff.
// The cast is a fire totem: 1000 ms GCD, 5 minutes, and a mana cost per rank
// from SpellPower (90, 140, 200, 275).
var flametongueTotemManaCost = [...]float64{90, 140, 200, 275}

func (shaman *Shaman) registerFlametongueTotemSpell() {
	shaman.FlametongueTotem = make([]*core.Spell, 0, len(core.FlametongueTotemRanks))
	buffAura := core.FlametongueTotemAura(&shaman.Character)

	for index, rank := range core.FlametongueTotemRanks {
		if int(shaman.Level) < rank.Level {
			break
		}
		spell := shaman.newFlametongueTotemSpellConfig(index+1, flametongueTotemManaCost[index], buffAura)
		shaman.FlametongueTotem = append(shaman.FlametongueTotem, shaman.RegisterSpell(spell))
	}
	shaman.FireTotems = append(shaman.FireTotems, shaman.FlametongueTotem...)
}

func (shaman *Shaman) newFlametongueTotemSpellConfig(rank int, manaCost float64, buffAura *core.Aura) core.SpellConfig {
	rankRow := core.FlametongueTotemRanks[rank-1]

	var lifetime *core.Aura
	if buffAura != nil {
		lifetime = shaman.registerBuffTotemLifetime(core.FlametongueTotemAuraLabel, rank, buffAura)
	} else {
		// Flametongue Weapon on the main hand disables the totem's benefit
		// to its own shaman: the totem stands, the buff does not.
		lifetime = shaman.RegisterAura(newTotemLifetimeConfig(core.FlametongueTotemAuraLabel, rank))
	}

	spell := shaman.newTotemSpellConfig(manaCost, rankRow.SpellID)
	spell.SpellSchool = core.SpellSchoolFire
	spell.RequiredLevel = rankRow.Level
	spell.Rank = rank
	spell.RelatedSelfBuff = lifetime
	spell.ApplyEffects = func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
		shaman.endStandingFireTotem(sim)
		shaman.dropStandingTotem(sim, FireTotem, spell, lifetime)
	}
	return spell
}
