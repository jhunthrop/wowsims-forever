package warlock

import (
	"time"

	googleProto "google.golang.org/protobuf/proto"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// summonDemon is one of the four summons: its spell, the level the trainer
// teaches it (trainables 688, 697, 712, 691), the option that names it and
// the pet it brings out.
type summonDemon struct {
	spellID int32
	level   int32
	option  proto.WarlockOptions_Summon
	pet     func(warlock *Warlock) *WarlockPet
}

// summonDemons keeps the engine's registration order.
var summonDemons = []summonDemon{
	{691, 30, proto.WarlockOptions_Felhunter, func(w *Warlock) *WarlockPet { return w.Felhunter }},
	{688, 1, proto.WarlockOptions_Imp, func(w *Warlock) *WarlockPet { return w.Imp }},
	{712, 20, proto.WarlockOptions_Succubus, func(w *Warlock) *WarlockPet { return w.Succubus }},
	{697, 10, proto.WarlockOptions_Voidwalker, func(w *Warlock) *WarlockPet { return w.Voidwalker }},
}

// learnedSummon is summon, or for a default demon the warlock has not learned: the
// Voidwalker from its class quest at 10 (demonology.md's leveling advice),
// the Imp before that.
func learnedSummon(summon proto.WarlockOptions_Summon, level int32) proto.WarlockOptions_Summon {
	for _, demon := range summonDemons {
		if demon.option == summon && level < demon.level {
			if level >= 10 {
				return proto.WarlockOptions_Voidwalker
			}
			return proto.WarlockOptions_Imp
		}
	}
	return summon
}

// withLearnedSummon returns options whose default demon is one a warlock of
// this level has learned; the caller's options are not changed.
func withLearnedSummon(options *proto.WarlockOptions, level int32) *proto.WarlockOptions {
	summon := learnedSummon(options.Summon, level)
	if summon == options.Summon {
		return options
	}
	learned := googleProto.Clone(options).(*proto.WarlockOptions)
	learned.Summon = summon
	return learned
}

func (warlock *Warlock) registerSummonDemon() {
	// All have a default cast time of 10s and the active pet is dismissed when the cast starts
	cast := core.CastConfig{
		DefaultCast: core.Cast{
			GCD:      core.GCDDefault,
			CastTime: time.Second * 10,
		},
		ModifyCast: func(sim *core.Simulation, spell *core.Spell, cast *core.Cast) {
			warlock.changeActivePet(sim, nil, false)
		},
	}

	for _, demon := range summonDemons {
		// Nothing checks RequiredLevel at cast time, so a summon the trainer
		// has not taught yet is not registered at all.
		if warlock.Level < demon.level {
			continue
		}
		demon := demon
		// The client's SpellPower.csv gives Summon Imp PowerCostPct 80 where
		// the other three summons are 100.
		manaCost := core.TernaryFloat64(demon.option == proto.WarlockOptions_Imp, 0.8, 1) * warlock.BaseMana
		warlock.SummonDemonSpells = append(warlock.SummonDemonSpells, warlock.RegisterSpell(core.SpellConfig{
			ActionID:      core.ActionID{SpellID: demon.spellID},
			SpellSchool:   core.SpellSchoolShadow,
			ProcMask:      core.ProcMaskEmpty,
			Flags:         core.SpellFlagAPL,
			RequiredLevel: int(demon.level),

			ManaCost: core.ManaCostOptions{FlatCost: manaCost},
			Cast:     cast,

			ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				warlock.changeActivePet(sim, demon.pet(warlock), false)
			},
		}))
	}
}
