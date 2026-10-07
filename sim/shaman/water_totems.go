package shaman

import (
	"fmt"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

const HealingStreamTotemRanks = 5

var HealingStreamTotemSpellId = [HealingStreamTotemRanks + 1]int32{0, 5394, 6375, 6377, 10462, 10463}
var HealingStreamTotemHealId = [HealingStreamTotemRanks + 1]int32{0, 5672, 6371, 6372, 10460, 10461}
var HealingStreamTotemBaseHealing = [HealingStreamTotemRanks + 1]float64{0, 6, 8, 10, 12, 14}
var HealingStreamTotemSpellCoeff = [HealingStreamTotemRanks + 1]float64{0, .022, .022, .022, .022, .022}
var HealingStreamTotemManaCost = [HealingStreamTotemRanks + 1]float64{0, 40, 50, 60, 70, 80}
var HealingStreamTotemLevel = [HealingStreamTotemRanks + 1]int{0, 20, 30, 40, 50, 60}

func (shaman *Shaman) registerHealingStreamTotemSpell() {
	shaman.HealingStreamTotem = make([]*core.Spell, HealingStreamTotemRanks+1)

	for rank := 1; rank <= HealingStreamTotemRanks; rank++ {
		config := shaman.newHealingStreamTotemSpellConfig(rank)

		if config.RequiredLevel <= int(shaman.Level) {
			shaman.HealingStreamTotem[rank] = shaman.RegisterSpell(config)
		}
	}

	shaman.WaterTotems = append(
		shaman.WaterTotems,
		core.FilterSlice(shaman.HealingStreamTotem, func(spell *core.Spell) bool { return spell != nil })...,
	)
}

func (shaman *Shaman) newHealingStreamTotemSpellConfig(rank int) core.SpellConfig {
	spellId := HealingStreamTotemSpellId[rank]
	healId := HealingStreamTotemHealId[rank]
	baseHealing := HealingStreamTotemBaseHealing[rank]*shaman.purificationHealingModifier() + shaman.restorativeTotemsModifier()
	spellCoeff := HealingStreamTotemSpellCoeff[rank]
	manaCost := HealingStreamTotemManaCost[rank]
	level := HealingStreamTotemLevel[rank]

	duration := time.Second * 60
	healInterval := time.Second * 2

	config := shaman.newTotemSpellConfig(manaCost, spellId)
	config.RequiredLevel = level
	config.Rank = rank

	healSpell := shaman.RegisterSpell(core.SpellConfig{
		ActionID:      core.ActionID{SpellID: healId},
		SpellSchool:   core.SpellSchoolNature,
		ProcMask:      core.ProcMaskSpellHealing,
		Flags:         core.SpellFlagHelpful | core.SpellFlagNoOnCastComplete | core.SpellFlagNoLogs | core.SpellFlagNoMetrics,
		RequiredLevel: level,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: spellCoeff,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.CalcAndDealHealing(sim, target, baseHealing, spell.OutcomeHealing)
		},
	})

	config.Hot = core.DotConfig{
		Aura: core.Aura{
			Label: fmt.Sprintf("Healing Stream HoT (Rank %d)", rank),
		},
		NumberOfTicks: int32(duration / healInterval),
		TickLength:    healInterval,
		OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
			healSpell.Cast(sim, target)
		},
	}

	config.ApplyEffects = func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
		shaman.TotemExpirations[WaterTotem] = sim.CurrentTime + duration
		shaman.ActiveTotems[WaterTotem] = spell

		for _, agent := range shaman.Party.Players {
			spell.Hot(&agent.GetCharacter().Unit).Activate(sim)
		}
	}

	return config
}

const ManaSpringTotemRanks = 4

var ManaSpringTotemSpellId = [ManaSpringTotemRanks + 1]int32{0, 5675, 10495, 10496, 10497}
var ManaSpringTotemManaRestore = [ManaSpringTotemRanks + 1]int32{0, 4, 6, 8, 10}
var ManaSpringTotemManaCost = [ManaSpringTotemRanks + 1]float64{0, 40, 60, 80, 100}
var ManaSpringTotemLevel = [ManaSpringTotemRanks + 1]int{0, 26, 36, 46, 56}

func (shaman *Shaman) registerManaSpringTotemSpell() {
	shaman.ManaSpringTotem = make([]*core.Spell, ManaSpringTotemRanks+1)

	for rank := 1; rank <= ManaSpringTotemRanks; rank++ {
		config := shaman.newManaSpringTotemSpellConfig(rank)

		if config.RequiredLevel <= int(shaman.Level) {
			shaman.ManaSpringTotem[rank] = shaman.RegisterSpell(config)
		}
	}

	shaman.WaterTotems = append(
		shaman.WaterTotems,
		core.FilterSlice(shaman.ManaSpringTotem, func(spell *core.Spell) bool { return spell != nil })...,
	)
}

// manaSpringTotemDuration is the client's duration_ms (300000) on every rank.
const manaSpringTotemDuration = 5 * time.Minute

// manaSpringMP5PerRestore turns "restores N mana every 2 seconds" into the
// engine's mana-per-5-seconds stat: 2.5 ticks every 5 seconds.
const manaSpringMP5PerRestore = 2.5

func (shaman *Shaman) newManaSpringTotemSpellConfig(rank int) core.SpellConfig {
	spellId := ManaSpringTotemSpellId[rank]
	manaCost := ManaSpringTotemManaCost[rank]
	level := ManaSpringTotemLevel[rank]
	bonus := stats.Stats{stats.MP5: float64(ManaSpringTotemManaRestore[rank]) * manaSpringMP5PerRestore}

	// The totem's restore is the shaman's own mana regeneration while it
	// stands; the group-wide raid buff in core/buffs.go stays a separate
	// source for other players' totems.
	buffAura := shaman.RegisterAura(core.Aura{
		Label:    fmt.Sprintf("Mana Spring Totem (Rank %d)", rank),
		ActionID: core.ActionID{SpellID: spellId},
		Duration: manaSpringTotemDuration,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			shaman.AddStatsDynamic(sim, bonus)
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			shaman.AddStatsDynamic(sim, bonus.Multiply(-1))
		},
	})

	spell := shaman.newTotemSpellConfig(manaCost, spellId)
	spell.RequiredLevel = level
	spell.Rank = rank
	spell.ApplyEffects = func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
		shaman.TotemExpirations[WaterTotem] = sim.CurrentTime + manaSpringTotemDuration
		shaman.ActiveTotems[WaterTotem] = spell
		buffAura.Activate(sim)
	}
	return spell
}
