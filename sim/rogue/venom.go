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
// Client rows (SpellEffect.csv, spell 1310703): two aura=108 percent
// modifiers of 30 on distinct poison spell-class masks (8192/8 and 65536,
// misc 0 and 22 -- the same pair Vile Poisons 16513 carries), so one 30%
// poison-damage bonus per poison, and one aura=107 flat modifier of 10 on
// the poison proc chance. Same aura kind, misc value and mask stack
// additively with Vile Poisons, so Venom is added to it, never multiplied.
// Effect index 0 is a Dummy template flag with nothing to implement.
const (
	venomPoisonDamageBonus     = 0.30
	venomPoisonProcChanceBonus = 0.10
)

// venomDamageScale is the factor that takes the poisons' current
// Vile-Poisons multiplier (1 + vile) to (1 + vile + venom). The client adds
// Venom and Vile Poisons: both are aura 108 percent modifiers with the same
// misc value and spell-class masks, which stack additively.
func (rogue *Rogue) venomDamageScale() float64 {
	vile := rogue.getPoisonDamageMultiplier()
	return (vile + venomPoisonDamageBonus) / vile
}

// scalePoisonDamage multiplies the damage multiplier of every poison spell.
func (rogue *Rogue) scalePoisonDamage(factor float64) {
	for _, spell := range []*core.Spell{rogue.InstantPoison, rogue.deadlyPoisonTick, rogue.WoundPoison} {
		if spell != nil {
			spell.DamageMultiplier *= factor
		}
	}
}

// venomLevel is Venom's learn level, the client's spell_level for 1310703.
const venomLevel = 40

func (rogue *Rogue) registerVenomSpell() {
	if !rogue.Talents.Venom {
		return
	}

	actionID := core.ActionID{SpellID: 1310703}

	// Index 0 is the client's base duration (spell 1310703, 6000 ms), the
	// registered default before a cast sets one by combo points, the same
	// shape as Slice and Dice's table; a cast needs a combo point, so
	// it is never the one a cast uses.
	rogue.venomDurations = [6]time.Duration{
		time.Second * 6,
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
		Duration: rogue.venomDurations[0],
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			rogue.additivePoisonBonusChance += venomPoisonProcChanceBonus
			rogue.scalePoisonDamage(rogue.venomDamageScale())
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			rogue.additivePoisonBonusChance -= venomPoisonProcChanceBonus
			rogue.scalePoisonDamage(1 / rogue.venomDamageScale())
		},
	})

	rogue.Venom = rogue.RegisterSpell(core.SpellConfig{
		SpellCode:     SpellCode_RogueVenom,
		ActionID:      actionID,
		Flags:         core.SpellFlagAPL,
		MetricSplits:  6,
		RequiredLevel: venomLevel,

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
