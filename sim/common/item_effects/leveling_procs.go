package item_effects

// Leveling-era weapon procs the engine did not implement before this
// lane (2026-09-28 weights-effects; rotation-accuracy program phase 2,
// docs/superpowers/specs/2026-09-28-rotation-accuracy-program-design.md
// on the site). Each item here was either an actual committed BiS pick
// at a leveling band (data/builds/<build>/bis/*.json, bands 10-55) or a
// top-3-scored candidate for some written spec's slot at those bands,
// whose effect_text sim/cmd/leveling-bis/score.go's own doc admits
// score() cannot see at all -- a stat-weight total with no way to
// value a proc.
//
// Every damage/duration number below is this SITE's own
// data/builds/<build>/items/<class>.json effect_text -- Forever's own
// rescaled tooltip, not vanilla Classic's. Blight is the clearest
// case: this build's 100 instant + 360-over-60s is exactly double
// vanilla Classic's 50 + 180 (confirmed against a live wowhead classic
// page for the real "Blight" spell, id 9796) -- the fork lane rule's
// "the client wins over Classic knowledge wherever it states a
// number" is why this file uses Forever's own doubled figures, not
// the vanilla ones a Classic wowhead search returns first.
//
// A proc rate ("PPM") is not carried by this site's item data at all,
// so every one of these follows this file's neighbours' own,
// long-standing convention two doors over (item_effects.go's
// "Chillpike", "Ebon Hand", "Emerald Dragonfang", ...) of assuming 1.0
// PPM absent testing data, flagged the same way they are: "TODO: Proc
// rate assumed and needs testing".
//
// Frost Tiger Blade's and Dark Iron Rifle's own on-hit bolt has no
// confirmed retail spell id the way Blight's does -- each uses
// core.ActionID{ItemID: ...} instead, the same fallback item_effects.go
// itself reaches for whenever an item's own activation needs no
// external spell id (see phase_6.go's "Kiss of the Spider").
//
// Frost Tiger Blade's own tooltip also states a 50% movement slow this
// file does not implement, following item_effects.go's own "Coldrage
// Dagger" (the near-identical "frost bolt + slow" effect already
// registered in this engine build), which carries the same omission --
// a stationary-target DPS sim has no movement to slow.
import (
	"time"

	"github.com/wowsims/classic/sim/common/itemhelpers"
	"github.com/wowsims/classic/sim/core"
)

const (
	FrostTigerBlade = 3854
	Blight          = 7959
	DarkIronRifle   = 16004
)

func init() {
	// https://www.wowhead.com/classic/item=3854/frost-tiger-blade
	// Chance on hit: Launches a bolt of frost at the enemy causing 50
	// Frost damage. TODO: Proc rate assumed and needs testing.
	itemhelpers.CreateWeaponProcSpell(FrostTigerBlade, "Frost Tiger Blade", 1.0, func(character *core.Character) *core.Spell {
		return character.RegisterSpell(core.SpellConfig{
			ActionID:         core.ActionID{ItemID: FrostTigerBlade},
			SpellSchool:      core.SpellSchoolFrost,
			DefenseType:      core.DefenseTypeMagic,
			ProcMask:         core.ProcMaskEmpty,
			Flags:            core.SpellFlagNoOnCastComplete | core.SpellFlagPassiveSpell,
			DamageMultiplier: 1,
			ThreatMultiplier: 1,
			ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				spell.CalcAndDealDamage(sim, target, 50, spell.OutcomeAlwaysHit)
			},
		})
	})

	// https://www.wowhead.com/classic/item=7959/blight
	// Chance on hit: Diseases a target for 100 Nature damage and an
	// additional 360 damage over 1 min (this build's own doubled
	// figures -- see this file's header doc). Real spell:
	// https://www.wowhead.com/classic/spell=9796/blight. The tick
	// schedule (12 ticks, 5 sec apart) is this lane's own assumption --
	// the client states only the total, not the schedule.
	// TODO: Proc rate assumed and needs testing.
	itemhelpers.CreateWeaponProcSpell(Blight, "Blight", 1.0, func(character *core.Character) *core.Spell {
		return character.RegisterSpell(core.SpellConfig{
			ActionID:         core.ActionID{SpellID: 9796},
			SpellSchool:      core.SpellSchoolNature,
			DefenseType:      core.DefenseTypeMagic,
			ProcMask:         core.ProcMaskEmpty,
			Flags:            core.SpellFlagNoOnCastComplete | core.SpellFlagPassiveSpell,
			DamageMultiplier: 1,
			ThreatMultiplier: 1,

			Dot: core.DotConfig{
				Aura: core.Aura{
					Label: "Blight (Blight)",
				},
				TickLength:    time.Second * 5,
				NumberOfTicks: 12,
				OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
					dot.Snapshot(target, 30, isRollover)
				},
				OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
					dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
				},
			},

			ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				spell.CalcAndDealDamage(sim, target, 100, spell.OutcomeAlwaysHit)
				spell.Dot(target).Apply(sim)
			},
		})
	})

	// https://www.wowhead.com/classic/item=16004/dark-iron-rifle
	// Chance on hit: Fires a Shadow Shot at the target for 26 Shadow
	// damage. TODO: Proc rate assumed and needs testing.
	itemhelpers.CreateWeaponProcSpell(DarkIronRifle, "Dark Iron Rifle", 1.0, func(character *core.Character) *core.Spell {
		return character.RegisterSpell(core.SpellConfig{
			ActionID:         core.ActionID{ItemID: DarkIronRifle},
			SpellSchool:      core.SpellSchoolShadow,
			DefenseType:      core.DefenseTypeMagic,
			ProcMask:         core.ProcMaskEmpty,
			Flags:            core.SpellFlagNoOnCastComplete | core.SpellFlagPassiveSpell,
			DamageMultiplier: 1,
			ThreatMultiplier: 1,
			ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				spell.CalcAndDealDamage(sim, target, 26, spell.OutcomeAlwaysHit)
			},
		})
	})
}
