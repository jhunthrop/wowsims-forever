package priest

import (
	"fmt"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

// innerFireDuration is the client's duration for every rank.
const innerFireDuration = 10 * time.Minute

// improvedInnerFireArmorPerRank is the talent's "+15% armor" per rank. Its
// extra charges are not modelled: they limit how many hits the buff
// survives, and the healer is not a target of the raid damage model.
const improvedInnerFireArmorPerRank = 0.15

// registerInnerFire registers every rank of Inner Fire, the priest's armor
// self buff.
func (priest *Priest) registerInnerFire() {
	priest.InnerFire = priest.registerHealRanks(innerFireRanks, func(rank int, entry healRank) core.SpellConfig {
		armor := entry.effect.Amount * (1 + improvedInnerFireArmorPerRank*float64(rankOf("improved_inner_fire", priest.Talents.ImprovedInnerFire)))
		aura := priest.RegisterAura(core.Aura{
			Label:    fmt.Sprintf("Inner Fire (Rank %d)", rank),
			ActionID: core.ActionID{SpellID: entry.spellID},
			Duration: innerFireDuration,
			OnGain: func(_ *core.Aura, sim *core.Simulation) {
				priest.AddStatDynamic(sim, stats.Armor, armor)
			},
			OnExpire: func(_ *core.Aura, sim *core.Simulation) {
				priest.AddStatDynamic(sim, stats.Armor, -armor)
			},
		})
		return core.SpellConfig{
			ActionID:    core.ActionID{SpellID: entry.spellID},
			SpellCode:   SpellCode_PriestInnerFire,
			SpellSchool: core.SpellSchoolHoly,
			Flags:       SpellFlagPriest | core.SpellFlagAPL,

			RequiredLevel: entry.level,
			Rank:          rank,

			ManaCost: core.ManaCostOptions{FlatCost: entry.manaCost},
			Cast: core.CastConfig{
				DefaultCast: core.Cast{GCD: core.GCDDefault},
			},

			RelatedSelfBuff: aura,
			ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
				aura.Activate(sim)
			},
		}
	})
}
