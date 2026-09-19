package warrior

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Piercing Howl is a Fury talent in the client's own trait table (node
// 105935, tier 2 column 1, one rank), not a baseline ability: an earlier
// draft of this spec had Forever making it baseline, and the client
// disagrees, so it is gated on the talent like every other talent-granted
// spell in this package.
//
// "Causes all nearby enemies to be Dazed, reducing movement speed by 50%
// for 6 sec." It deals no damage, so it changes nothing on a
// single-target fight; it is registered so the APL validator knows the
// character has it and so a movement-aware encounter profile can use it.
//
// Every number comes from the generated constants
// (sim/warrior/constants_auto_gen.go, build 1.60.1.69893) rather than a
// literal, except the snare's own duration and magnitude, which the
// generated arrays do not carry - those are quoted from the client's
// rank description above and are what the row's BaseDamage of -50 says
// as well.
const (
	// piercingHowlRank is the index into the generated rank arrays. The
	// client gives Piercing Howl a single, unnumbered rank, which the
	// generator emits at label 0.
	piercingHowlRank = 0

	// piercingHowlSnareDuration is the "for 6 sec" of the rank
	// description; the generated arrays carry cast time, cooldown and
	// cost but not aura duration.
	piercingHowlSnareDuration = time.Second * 6
)

// piercingHowlRageCost converts the client's cost column, which stores
// rage in tenths, into the rage the engine charges.
func piercingHowlRageCost() float64 {
	return PiercingHowlManaCost[piercingHowlRank] / 10
}

func (warrior *Warrior) registerPiercingHowlSpell() {
	if !warrior.Talents.PiercingHowl {
		return
	}

	warrior.PiercingHowl = warrior.RegisterSpell(AnyStance, core.SpellConfig{
		ActionID:       core.ActionID{SpellID: PiercingHowlSpellId[piercingHowlRank]},
		ClassSpellMask: WarriorSpellMaskPiercingHowl,
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskEmpty,
		Flags:          core.SpellFlagAPL | core.SpellFlagNoOnCastComplete,

		RageCost: core.RageCostOptions{
			Cost: piercingHowlRageCost(),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			IgnoreHaste: true,
			// The generated cooldown for this rank is 0, so no timer is
			// registered: an always-ready Cooldown still costs a timer,
			// and sim/core/cooldown.go panics past a hundred of them.
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			for _, aoeTarget := range sim.Encounter.TargetUnits {
				spell.CalcAndDealOutcome(sim, aoeTarget, spell.OutcomeAlwaysHit)
			}
		},
	})
}
