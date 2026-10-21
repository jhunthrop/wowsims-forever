package druid

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// tigersFuryLearnLevels: source 1.60.1.70009 client spell data. "Tiger's
// Fury" carries only rank-0 entries (id 5217, level 24) -- per the reference
// caveat, an all-rank-0 name is a spell without ranks. The old code's other
// three ids (6793/9845/9846) do not exist anywhere in this client's spell
// data at all; they are a stale four-tier SoD scaling that the client no
// longer models. Tiger's Fury is now a single spell learned at level 24.
var tigersFuryLearnLevels = []int{24}

const tigersFurySpellID = 5217

// tigersFuryDamageBonus is spell 5217's effect 0 (aura 4, mod damage done,
// school physical): base_points/amount 15 (16 on wowhead's Forever tooltip; the
// client stores the float 15.x). Forever made Tiger's Fury free
// (cost 0) on a 30 s cooldown; it no longer spends or requires energy.
const tigersFuryDamageBonus = 16.0

func (druid *Druid) registerTigersFurySpell() {
	rank := core.HighestRankAtLevel(tigersFuryLearnLevels, druid.Level)
	if rank == 0 {
		return
	}

	actionID := core.ActionID{SpellID: tigersFurySpellID}
	dmgBonus := tigersFuryDamageBonus

	druid.TigersFuryAura = druid.RegisterAura(core.Aura{
		Label:    "Tiger's Fury Aura",
		ActionID: actionID,
		Duration: 6 * time.Second,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			druid.PseudoStats.BonusPhysicalDamage += dmgBonus
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			druid.PseudoStats.BonusPhysicalDamage -= dmgBonus
		},
	})

	spell := druid.RegisterSpell(Cat, core.SpellConfig{
		ActionID:       actionID,
		ClassSpellMask: DruidSpellMaskTigersFury,
		Flags:          core.SpellFlagAPL,

		RequiredLevel: tigersFuryLearnLevels[rank-1],

		Cast: core.CastConfig{
			CD: core.Cooldown{
				Timer:    druid.NewTimer(),
				Duration: 30 * time.Second,
			},
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			druid.TigersFuryAura.Activate(sim)
		},
	})

	druid.TigersFury = spell
}
