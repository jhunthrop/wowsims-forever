package shaman

import (
	"fmt"

	"github.com/wowsims/classic/sim/core"
)

const WindfuryTotemRanks = 3

var WindfuryTotemSpellId = [WindfuryTotemRanks + 1]int32{0, 8512, 10613, 10614}

// WindfuryTotemAuraId is the totem's own aura on the party, per rank: an
// area aura (SpellEffect effect 65, aura 42, the 20% proc of 8516, 10608
// or 10610) with no duration in the 1.60.1.70753 client. It holds only
// while the totem stands; vanilla's 10-second weapon enchant (8514, 10607,
// 10611) is not in the client.
var WindfuryTotemAuraId = [WindfuryTotemRanks + 1]int32{0, 8515, 10609, 10612}
var WindfuryTotemManaCost = [WindfuryTotemRanks + 1]float64{0, 115, 175, 250}
var WindfuryTotemLevel = [WindfuryTotemRanks + 1]int{0, 32, 42, 52}

func (shaman *Shaman) registerWindfuryTotemSpell() {
	shaman.WindfuryTotem = make([]*core.Spell, WindfuryTotemRanks+1)

	for rank := 1; rank <= WindfuryTotemRanks; rank++ {
		config := shaman.newWindfuryTotemSpellConfig(rank)

		if config.RequiredLevel <= int(shaman.Level) {
			shaman.WindfuryTotem[rank] = shaman.RegisterSpell(config)
		}
	}

	shaman.AirTotems = append(
		shaman.AirTotems,
		core.FilterSlice(shaman.WindfuryTotem, func(spell *core.Spell) bool { return spell != nil })...,
	)
}

func (shaman *Shaman) newWindfuryTotemSpellConfig(rank int) core.SpellConfig {
	spellId := WindfuryTotemSpellId[rank]
	// TODO: The sim won't respect the value of a totem dropped via the APL. It uses hard-coded values from buffs.go
	manaCost := WindfuryTotemManaCost[rank]
	level := WindfuryTotemLevel[rank]

	// A trackable aura for the standing totem: it is up exactly as long as
	// the totem is.
	buffAura := shaman.RegisterAura(core.Aura{
		ActionID: core.ActionID{SpellID: WindfuryTotemAuraId[rank]},
		Label:    fmt.Sprintf("Windfury (Rank %d)", rank),
		Duration: core.NeverExpires,
	})

	lifetime := shaman.registerBuffTotemLifetime("Windfury Totem", rank, buffAura)

	spell := shaman.newTotemSpellConfig(manaCost, spellId)
	spell.RequiredLevel = level
	spell.Rank = rank
	spell.RelatedSelfBuff = lifetime
	spell.ApplyEffects = func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
		shaman.dropStandingTotem(sim, AirTotem, spell, lifetime)
	}
	return spell
}

const GraceOfAirTotemRanks = 3

var GraceOfAirTotemSpellId = [GraceOfAirTotemRanks + 1]int32{0, 8835, 10627, 25359}
var GraceOfAirTotemManaCost = [GraceOfAirTotemRanks + 1]float64{0, 155, 250, 310}
var GraceOfAirTotemLevel = [GraceOfAirTotemRanks + 1]int{0, 42, 56, 60}

func (shaman *Shaman) registerGraceOfAirTotemSpell() {
	shaman.GraceOfAirTotem = make([]*core.Spell, GraceOfAirTotemRanks+1)

	for rank := 1; rank <= GraceOfAirTotemRanks; rank++ {
		config := shaman.newGraceOfAirTotemSpellConfig(rank)

		if config.RequiredLevel <= int(shaman.Level) {
			shaman.GraceOfAirTotem[rank] = shaman.RegisterSpell(config)
		}
	}

	shaman.AirTotems = append(
		shaman.AirTotems,
		core.FilterSlice(shaman.GraceOfAirTotem, func(spell *core.Spell) bool { return spell != nil })...,
	)
}

func (shaman *Shaman) newGraceOfAirTotemSpellConfig(rank int) core.SpellConfig {
	spellId := GraceOfAirTotemSpellId[rank]
	manaCost := GraceOfAirTotemManaCost[rank]
	level := GraceOfAirTotemLevel[rank]

	// FOREVER: Enhancing Totems is not in the client's trees.
	// multiplier := []float64{1, 1.08, 1.15}[shaman.Talents.EnhancingTotems]
	multiplier := 1.0

	buffAura := core.GraceOfAirTotemAuraAt(&shaman.Unit, core.GraceOfAirRanks[rank-1], multiplier)

	lifetime := shaman.registerBuffTotemLifetime("Grace of Air Totem", rank, buffAura)

	spell := shaman.newTotemSpellConfig(manaCost, spellId)
	spell.RequiredLevel = level
	spell.Rank = rank
	spell.RelatedSelfBuff = lifetime
	spell.ApplyEffects = func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
		shaman.dropStandingTotem(sim, AirTotem, spell, lifetime)
	}
	return spell
}

const WindwallTotemRanks = 3

var WindwallTotemSpellId = [WindwallTotemRanks + 1]int32{0, 15107, 15111, 15112}
var WindwallTotemManaCost = [WindwallTotemRanks + 1]float64{0, 115, 170, 225}
var WindwallTotemLevel = [WindwallTotemRanks + 1]int{0, 36, 46, 56}

func (shaman *Shaman) registerWindwallTotemSpell() {
	shaman.WindwallTotem = make([]*core.Spell, WindwallTotemRanks+1)

	for rank := 1; rank <= WindwallTotemRanks; rank++ {
		config := shaman.newWindwallTotemSpellConfig(rank)

		if config.RequiredLevel <= int(shaman.Level) {
			shaman.WindwallTotem[rank] = shaman.RegisterSpell(config)
		}
	}

	shaman.AirTotems = append(
		shaman.AirTotems,
		core.FilterSlice(shaman.WindwallTotem, func(spell *core.Spell) bool { return spell != nil })...,
	)
}

func (shaman *Shaman) newWindwallTotemSpellConfig(rank int) core.SpellConfig {
	spellId := WindwallTotemSpellId[rank]
	manaCost := WindwallTotemManaCost[rank]
	level := WindwallTotemLevel[rank]

	lifetime := shaman.RegisterAura(newTotemLifetimeConfig("Windwall Totem", rank))

	spell := shaman.newTotemSpellConfig(manaCost, spellId)
	spell.RequiredLevel = level
	spell.Rank = rank
	spell.RelatedSelfBuff = lifetime
	spell.ApplyEffects = func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
		shaman.dropStandingTotem(sim, AirTotem, spell, lifetime)
	}
	return spell
}
