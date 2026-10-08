package shaman

import (
	"fmt"
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

const SearingTotemRanks = 6

var SearingTotemSpellId = [SearingTotemRanks + 1]int32{0, 3599, 6363, 6364, 6365, 10437, 10438}
var SearingTotemAttackSpellId = [SearingTotemRanks + 1]int32{0, 3606, 6350, 6351, 6352, 10435, 10436}

// The Searing Totem's bolt is the client's "Attack" spell (ids above):
// its own roll per rank (rank 6 rolls 40-54 at level 60, a centre of 47)
// and a spell-power coefficient of 0.017, a fifth of the 0.083 the
// Classic port carried, so a geared shaman's totem was hitting for
// several times its real damage. The Classic min-max roll the table
// replaces averaged within a point of the client's.
var SearingTotemDamage = [SearingTotemRanks + 1]clientdamage.Effect{
	{},
	{Amount: 10, Variance: 0.2, SpellLevel: 10},
	{Amount: 15, Variance: 0.266667, SpellLevel: 20},
	{Amount: 22, Variance: 0.272727, SpellLevel: 30},
	{Amount: 30, Variance: 0.266667, SpellLevel: 40},
	{Amount: 39, Variance: 0.307692, SpellLevel: 50},
	{Amount: 47, Variance: 0.297872, SpellLevel: 60},
}
var SearingTotemSpellCoef = [SearingTotemRanks + 1]float64{0, .017, .017, .017, .017, .017, .017}
var SearingTotemManaCost = [SearingTotemRanks + 1]float64{0, 25, 45, 75, 110, 145, 170}
var SearingTotemDuration = [SearingTotemRanks + 1]int{0, 30, 35, 40, 45, 50, 55}
var SearingTotemLevel = [SearingTotemRanks + 1]int{0, 10, 20, 30, 40, 50, 60}

func (shaman *Shaman) registerSearingTotemSpell() {
	shaman.SearingTotem = make([]*core.Spell, SearingTotemRanks+1)

	for rank := 1; rank <= SearingTotemRanks; rank++ {
		config := shaman.newSearingTotemSpellConfig(rank)

		if config.RequiredLevel <= int(shaman.Level) {
			shaman.SearingTotem[rank] = shaman.RegisterSpell(config)
		}
	}

	shaman.FireTotems = append(
		shaman.FireTotems,
		core.FilterSlice(shaman.SearingTotem, func(spell *core.Spell) bool { return spell != nil })...,
	)
}

func (shaman *Shaman) newSearingTotemSpellConfig(rank int) core.SpellConfig {
	totemSpellId := SearingTotemSpellId[rank]
	damage := SearingTotemDamage[rank]
	casterLevel := int(shaman.Level)
	spellCoeff := SearingTotemSpellCoef[rank]
	manaCost := SearingTotemManaCost[rank]
	duration := time.Second * time.Duration(SearingTotemDuration[rank])
	level := SearingTotemLevel[rank]

	attackInterval := time.Millisecond * 2500

	attackSpell := shaman.RegisterSpell(core.SpellConfig{
		SpellCode:   SpellCode_ShamanSearingTotemAttack,
		ActionID:    core.ActionID{SpellID: SearingTotemAttackSpellId[rank]},
		SpellSchool: core.SpellSchoolFire,
		DefenseType: core.DefenseTypeMagic,
		ProcMask:    core.ProcMaskEmpty,

		DamageMultiplier: shaman.callOfFlameMultiplier(),
		BonusCoefficient: spellCoeff,
		ClientBaseDamage: damage.Range(casterLevel),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.CalcAndDealDamage(sim, target, damage.Roll(sim, casterLevel), spell.OutcomeMagicHitAndCrit)
		},
	})

	spell := core.SpellConfig{
		SpellCode:   SpellCode_ShamanSearingTotem,
		ActionID:    core.ActionID{SpellID: totemSpellId},
		SpellSchool: core.SpellSchoolFire,
		DefenseType: core.DefenseTypeMagic,
		ProcMask:    core.ProcMaskEmpty,
		Flags:       SpellFlagTotem | core.SpellFlagAPL,

		RequiredLevel: level,
		Rank:          rank,

		ManaCost: core.ManaCostOptions{
			FlatCost:   manaCost,
			Multiplier: shaman.totemManaMultiplier(),
		},

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: totemGCD,
			},
			IgnoreHaste: true,
		},

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: fmt.Sprintf("Searing Totem (Rank %d)", rank),
			},
			// These are the real tick values, but searing totem doesn't start its next
			// cast until the previous missile hits the target. We don't have an option
			// for target distance yet so just pretend the tick rate is lower.
			// https://wotlk.wowhead.com/spell=25530/attack
			//TickLength:           time.Second * 2.2,
			NumberOfTicks: int32(duration / attackInterval),
			TickLength:    attackInterval,
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				attackSpell.Cast(sim, target)
			},
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
			shaman.endStandingFireTotem(sim)
			spell.Dot(sim.GetTargetUnit(0)).Apply(sim)
			// +1 needed because of rounding issues with totem tick time.
			shaman.TotemExpirations[FireTotem] = sim.CurrentTime + duration + 1
			shaman.ActiveTotems[FireTotem] = spell
		},
	}

	return spell
}

const MagmaTotemRanks = 4

var MagmaTotemSpellId = [MagmaTotemRanks + 1]int32{0, 8190, 10585, 10586, 10587}
var MagmaTotemAoeSpellId = [MagmaTotemRanks + 1]int32{0, 8187, 10579, 10580, 10581}
var MagmaTotemDamage = [MagmaTotemRanks + 1]clientdamage.Effect{
	{},
	{Amount: 20, SpellLevel: 26},
	{Amount: 35, SpellLevel: 36},
	{Amount: 52, SpellLevel: 46},
	{Amount: 73, SpellLevel: 56},
}
var MagmaTotemSpellCoeff = [MagmaTotemRanks + 1]float64{0, .033, .033, .033, .033}
var MagmaTotemManaCost = [MagmaTotemRanks + 1]float64{0, 230, 360, 500, 650}
var MagmaTotemLevel = [MagmaTotemRanks + 1]int{0, 26, 36, 46, 56}

func (shaman *Shaman) registerMagmaTotemSpell() {
	shaman.MagmaTotem = make([]*core.Spell, MagmaTotemRanks+1)

	for rank := 1; rank <= MagmaTotemRanks; rank++ {
		config := shaman.newMagmaTotemSpellConfig(rank)

		if config.RequiredLevel <= int(shaman.Level) {
			shaman.MagmaTotem[rank] = shaman.RegisterSpell(config)
		}
	}

	shaman.FireTotems = append(
		shaman.FireTotems,
		core.FilterSlice(shaman.MagmaTotem, func(spell *core.Spell) bool { return spell != nil })...,
	)
}

func (shaman *Shaman) newMagmaTotemSpellConfig(rank int) core.SpellConfig {
	spellId := MagmaTotemSpellId[rank]
	damage := MagmaTotemDamage[rank]
	casterLevel := int(shaman.Level)
	spellCoeff := MagmaTotemSpellCoeff[rank]
	manaCost := MagmaTotemManaCost[rank]
	level := MagmaTotemLevel[rank]

	duration := time.Second * 20
	attackInterval := time.Second * 2

	aoeSpell := shaman.RegisterSpell(core.SpellConfig{
		SpellCode:     SpellCode_ShamanMagmaTotem,
		ActionID:      core.ActionID{SpellID: MagmaTotemAoeSpellId[rank]},
		SpellSchool:   core.SpellSchoolFire,
		DefenseType:   core.DefenseTypeMagic,
		ProcMask:      core.ProcMaskEmpty,
		RequiredLevel: level,

		DamageMultiplier: shaman.callOfFlameMultiplier(),
		BonusCoefficient: spellCoeff,
		ClientBaseDamage: damage.Range(casterLevel),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			for _, aoeTarget := range sim.Encounter.TargetUnits {
				spell.CalcAndDealDamage(sim, aoeTarget, damage.Roll(sim, casterLevel), spell.OutcomeMagicHitAndCrit)
			}
		},
	})

	spell := core.SpellConfig{
		SpellCode:   SpellCode_ShamanMagmaTotem,
		ActionID:    core.ActionID{SpellID: spellId},
		SpellSchool: core.SpellSchoolFire,
		DefenseType: core.DefenseTypeMagic,
		ProcMask:    core.ProcMaskEmpty,
		Flags:       SpellFlagTotem | core.SpellFlagAPL,

		RequiredLevel: level,
		Rank:          rank,

		ManaCost: core.ManaCostOptions{
			FlatCost:   manaCost,
			Multiplier: shaman.totemManaMultiplier(),
		},

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: totemGCD,
			},
			IgnoreHaste: true,
		},

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: fmt.Sprintf("Magma Totem (Rank %d)", rank),
			},
			NumberOfTicks: int32(duration / attackInterval),
			TickLength:    attackInterval,

			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				aoeSpell.Cast(sim, target)
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			shaman.endStandingFireTotem(sim)
			spell.Dot(sim.GetTargetUnit(0)).Apply(sim)
			// +1 needed because of rounding issues with totem tick time.
			shaman.TotemExpirations[FireTotem] = sim.CurrentTime + duration + 1
			shaman.ActiveTotems[FireTotem] = spell
		},
	}

	return spell
}
