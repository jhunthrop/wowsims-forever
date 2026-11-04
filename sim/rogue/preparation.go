package rogue

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// preparationWorthItAfter is how much of a long DPS cooldown has to be left
// before spending Preparation on it: less than this and waiting costs less
// than the 10 minute reset.
const preparationWorthItAfter = 30 * time.Second

// Preparation (14185): "When activated, this ability immediately finishes
// the cooldown on your other Rogue abilities." Its one effect is a server
// dummy (effect 3), so the rank text is the mechanic: every rogue cooldown
// but its own.
func (rogue *Rogue) registerPreparationCD() {
	if !rogue.Talents.Preparation {
		return
	}

	rogue.Preparation = rogue.RegisterSpell(core.SpellConfig{
		ActionID: core.ActionID{SpellID: 14185},
		Flags:    core.SpellFlagAPL,
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: time.Second,
			},
			CD: core.Cooldown{
				Timer:    rogue.NewTimer(),
				Duration: time.Minute * 10,
			},
			IgnoreHaste: true,
		},
		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
			for _, affected := range rogue.preparationResets() {
				affected.CD.Reset()
			}
		},
	})

	rogue.AddMajorCooldown(core.MajorCooldown{
		Spell:    rogue.Preparation,
		Type:     core.CooldownTypeDPS,
		Priority: core.CooldownPriorityDefault,
		ShouldActivate: func(sim *core.Simulation, character *core.Character) bool {
			if rogue.IsStealthed() || rogue.deathMarkRunning(sim) || rogue.ComboPoints() > premeditationMaxComboPointsBefore {
				return false
			}
			for _, short := range []*core.Spell{rogue.Shank, rogue.GhostlyStrike} {
				if short != nil && short.CD.IsReady(sim) {
					return false
				}
			}
			for _, long := range []*core.Spell{rogue.Premeditation, rogue.ColdBlood, rogue.AdrenalineRush, rogue.BladeFlurry} {
				if long != nil && long.CD.TimeToReady(sim) > preparationWorthItAfter {
					return true
				}
			}
			return false
		},
	})
}

// preparationResets is every registered rogue spell with a cooldown other
// than Preparation itself.
func (rogue *Rogue) preparationResets() []*core.Spell {
	var out []*core.Spell
	for _, spell := range []*core.Spell{
		rogue.AdrenalineRush, rogue.BladeFlurry, rogue.ColdBlood, rogue.DeathMark, rogue.Evasion,
		rogue.GhostlyStrike, rogue.Gouge, rogue.Kick, rogue.Premeditation, rogue.Shadowstep,
		rogue.Shank, rogue.Vanish,
	} {
		if spell != nil && spell.CD.Timer != nil {
			out = append(out, spell)
		}
	}
	return out
}

// deathMarkRunning reports whether a Death Mark cast less than its 6 sec
// ago may still be storing: resetting the cooldown then would let a second
// cast overwrite the first mark.
func (rogue *Rogue) deathMarkRunning(sim *core.Simulation) bool {
	return rogue.DeathMark != nil && rogue.DeathMark.CD.TimeToReady(sim) > deathMarkCooldown-deathMarkDuration
}
