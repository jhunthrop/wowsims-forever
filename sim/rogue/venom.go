package rogue

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Venom is a Forever-only Assassination talent (tier 6, single rank):
// talent node 105712, spell 1310703, prereq Mutilate rank 1. Client
// tooltip (talents/rogue.json): "Finishing move that increases the
// damage of your Poisons by 30% and your chance to apply Poisons by 10%.
// Lasts longer per combo point: 1 point 9 sec, 2 points 12 sec, 3 points
// 15 sec, 4 points 18 sec, 5 points 21 sec." -- a flat 6 s plus 3 s per
// combo point, which the spellconst duration_ms (6000, the 0-combo-point
// base) corroborates.
//
// spellconst's effects give three real numbers for this: two aura=108
// (amount 30, differing only in misc_value 0 vs 22) and one aura=107
// (amount 10). The tooltip names exactly one poison-damage bonus and one
// poison-proc-chance bonus, so the two aura=108 entries are read as one
// 30% poison-damage bonus (not stacked to 60%) rather than guessed apart
// -- the client's own effect-type numbering isn't decoded anywhere in
// this codebase (sim/core/spellconst never interprets the Effect/Aura
// columns, only carries them), so there is nothing here to disambiguate
// misc_value 0 from 22 against. The third effect (index 0, effect 3, a
// Dummy with amount 0) is the finishing-move template flag Rupture and
// Slice and Dice's own dot/aura effects also carry; nothing to implement
// from it.
const (
	venomPoisonDamageMultiplier = 1.30
	venomPoisonProcChanceBonus  = 0.10
)

func (rogue *Rogue) registerVenomSpell() {
	if !rogue.Talents.Venom {
		return
	}

	actionID := core.ActionID{SpellID: 1310703}

	rogue.venomDurations = [6]time.Duration{
		0,
		time.Second * 9,
		time.Second * 12,
		time.Second * 15,
		time.Second * 18,
		time.Second * 21,
	}

	rogue.VenomAura = rogue.RegisterAura(core.Aura{
		Label:    "Venom",
		ActionID: actionID,
		// Overridden on cast to the combo-point duration; a non-zero
		// default so it doesn't crash when read by an APL prepull, same
		// as SliceAndDiceAura above.
		Duration: rogue.venomDurations[5],
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			rogue.additivePoisonBonusChance += venomPoisonProcChanceBonus
			if rogue.InstantPoison != nil {
				rogue.InstantPoison.DamageMultiplier *= venomPoisonDamageMultiplier
			}
			if rogue.deadlyPoisonTick != nil {
				rogue.deadlyPoisonTick.DamageMultiplier *= venomPoisonDamageMultiplier
			}
			if rogue.WoundPoison != nil {
				rogue.WoundPoison.DamageMultiplier *= venomPoisonDamageMultiplier
			}
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			rogue.additivePoisonBonusChance -= venomPoisonProcChanceBonus
			if rogue.InstantPoison != nil {
				rogue.InstantPoison.DamageMultiplier /= venomPoisonDamageMultiplier
			}
			if rogue.deadlyPoisonTick != nil {
				rogue.deadlyPoisonTick.DamageMultiplier /= venomPoisonDamageMultiplier
			}
			if rogue.WoundPoison != nil {
				rogue.WoundPoison.DamageMultiplier /= venomPoisonDamageMultiplier
			}
		},
	})

	rogue.Venom = rogue.RegisterSpell(core.SpellConfig{
		SpellCode:    SpellCode_RogueVenom,
		ActionID:     actionID,
		Flags:        core.SpellFlagAPL,
		MetricSplits: 6,

		EnergyCost: core.EnergyCostOptions{
			Cost: 25,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: time.Second,
			},
			IgnoreHaste: true,
			ModifyCast: func(sim *core.Simulation, spell *core.Spell, cast *core.Cast) {
				spell.SetMetricsSplit(spell.Unit.ComboPoints())
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return rogue.ComboPoints() > 0
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
			rogue.VenomAura.Duration = rogue.venomDurations[rogue.ComboPoints()]
			rogue.VenomAura.Activate(sim)
			rogue.SpendComboPoints(sim, spell)
		},
	})
	rogue.Finishers = append(rogue.Finishers, rogue.Venom)
}
