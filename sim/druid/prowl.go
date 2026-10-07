package druid

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// prowlLearnLevels are Prowl's three rank learn levels; source:
// 1.60.1.70009 client spell data ("Prowl", ranks 1-3: levels 20/40/60).
// Every rank has cost 0, cast_time_ms 0 and gcd_ms 0 -- only the stealth
// detection bonus (effect 0, aura 16) and move-speed penalty (effect 1,
// aura 33) change per rank, neither modeled here (mirrors
// sim/rogue/stealth.go's own Stealth ranks).
var prowlLearnLevels = []int{20, 40, 60}

// prowlSpellID is Prowl's rank -> spell id, index 0 unused.
var prowlSpellID = [4]int32{0, 5215, 6783, 9913}

// registerProwlAura registers the Prowl aura unconditionally (mirroring
// sim/rogue/stealth.go's registerStealthAura, called from Initialize
// regardless of level) so BreakProwl is always safe to call from any Cat
// Form ability's ApplyEffects, even below Prowl's own level-20 learn level.
func (druid *Druid) registerProwlAura() {
	druid.ProwlAura = druid.RegisterAura(core.Aura{
		Label:    "Prowl",
		ActionID: core.ActionID{SpellID: prowlSpellID[3]},
		Duration: core.NeverExpires,
	})
}

// registerProwlSpell registers Prowl as a castable APL action so a Feral
// rotation's prepullActions can open a pull from Prowl the way a real Cat
// Form druid does: out of combat only, Cat Form only, and only while not
// already Prowling -- there is no in-combat path back into ProwlAura
// (mirrors sim/rogue/stealth.go's registerStealthSpell; Cat Form has no
// Vanish equivalent, so unlike Stealth this is the only path in at all).
func (druid *Druid) registerProwlSpell() {
	rank := core.HighestRankAtLevel(prowlLearnLevels, druid.Level)
	if rank == 0 {
		return
	}

	druid.Prowl = druid.RegisterSpell(Cat, core.SpellConfig{
		ActionID: core.ActionID{SpellID: prowlSpellID[rank]},
		Flags:    core.SpellFlagAPL,

		RequiredLevel: prowlLearnLevels[rank-1],

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: 0,
			},
			IgnoreHaste: true,
			// Prowl's cooldown_ms column is 0 but category_cooldown_ms
			// is 10000 on every rank (1.60.1.70009 spellconst/
			// druid.json ids 5215/6783/9913); the engine previously had
			// no CD at all configured.
			CD: core.Cooldown{
				Timer:    druid.NewTimer(),
				Duration: time.Second * 10,
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return sim.CurrentTime < 0 && !druid.IsProwling()
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			druid.ProwlAura.Activate(sim)
		},
	})
}
