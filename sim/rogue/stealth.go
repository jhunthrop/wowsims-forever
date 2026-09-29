package rogue

import (
	"github.com/wowsims/classic/sim/core"
)

// stealthLearnLevels are Stealth's four rank learn levels; source:
// 1.60.1.70009 client spell data ("Stealth", ranks 1-4: levels 1/20/40/60).
// Every rank has cost 0, cast_time_ms 0 and gcd_ms 0 -- only the detection
// radius and move-speed penalty change per rank, neither modeled here.
var stealthLearnLevels = []int{1, 20, 40, 60}

// stealthSpellID is Stealth's rank -> spell id, index 0 unused.
var stealthSpellID = [5]int32{0, 1784, 1785, 1786, 1787}

func (rogue *Rogue) registerStealthAura() {
	rogue.StealthAura = rogue.RegisterAura(core.Aura{
		Label:    "Stealth",
		ActionID: core.ActionID{SpellID: 1787},
		Duration: core.NeverExpires,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			// Stealth triggered auras
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
		},
		// Stealth breaks on damage taken (if not absorbed)
		// This may be desirable later, but not applicable currently
	})
}

// registerStealthSpell registers Stealth as a castable APL action so a
// rotation's prepullActions can open a pull from Stealth the way a real
// rogue does: out of combat only (Stealth cannot be (re)cast once the pull
// has started -- Vanish is the only in-combat path back into StealthAura,
// registered separately in vanish.go), free, instant, no GCD, matching the
// client's own numbers for every rank.
func (rogue *Rogue) registerStealthSpell() {
	rank := core.HighestRankAtLevel(stealthLearnLevels, rogue.Level)
	if rank == 0 {
		return
	}

	rogue.Stealth = rogue.RegisterSpell(core.SpellConfig{
		SpellCode: SpellCode_RogueStealth,
		ActionID:  core.ActionID{SpellID: stealthSpellID[rank]},
		Flags:     core.SpellFlagAPL,

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: 0,
			},
			IgnoreHaste: true,
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return sim.CurrentTime < 0 && !rogue.IsStealthed()
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			rogue.StealthAura.Activate(sim)
		},
	})
}
