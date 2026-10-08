package priest

import (
	"fmt"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

// Shadowfiend (client spell 401977) summons a pet for 15 s (SpellDuration
// 15000) on a five minute cooldown (SpellCooldowns 300000) and a 1.5 s
// global cooldown, at no mana cost. Its Mana Leech (spell 401988, effect
// 137, energize percent, 5) gives the caster 5 percent of its maximum mana
// whenever the fiend attacks.
//
// The pet's swing timer is not in the client tables the site reads (the
// creature, NPC 202079 from the summon effect, is not among them), so the
// engine does not summon a pet: it returns the leech on a stated swing of
// shadowfiendSwing, one return per swing, every swing landing. The swing is
// an ASSUMPTION; the summon duration and the five percent are the client's.
const (
	shadowfiendSpellID       int32 = 401977
	shadowfiendLevel               = 1 // SkillLineAbility: learned at level 1
	shadowfiendDuration            = 15 * time.Second
	shadowfiendCooldown            = 5 * time.Minute
	shadowfiendSwing               = 1500 * time.Millisecond // assumption, see above
	shadowfiendManaLeechFrac       = 0.05
	shadowfiendSwings              = int32(shadowfiendDuration / shadowfiendSwing)
)

// Dark Sacrifice (client spells 1277324 to 1277328, ranks at levels 20 to
// 60) is a 15 s aura on the caster with a ten minute cooldown (the
// category recovery, 600000) and a 1.5 s global cooldown. Every 3 s
// (EffectAuraPeriod 3000) it takes the rank's amount of health (aura 226,
// base 80 to 320) and gives the same amount of mana (aura 24); the spell's
// text adds the caster's Spirit to the mana total (${$o2+$SPI}), which the
// engine spreads over the five ticks.
const (
	darkSacrificeDuration   = 15 * time.Second
	darkSacrificeTickLength = 3 * time.Second
	darkSacrificeTicks      = int32(darkSacrificeDuration / darkSacrificeTickLength)
)

func (priest *Priest) registerShadowfiend() {
	actionID := core.ActionID{SpellID: shadowfiendSpellID}
	manaMetrics := priest.NewManaMetrics(actionID)

	priest.Shadowfiend = priest.RegisterSpell(core.SpellConfig{
		ActionID:      actionID,
		Flags:         core.SpellFlagNoOnCastComplete | core.SpellFlagAPL | core.SpellFlagHelpful,
		RequiredLevel: shadowfiendLevel,

		Cast: core.CastConfig{
			DefaultCast: core.Cast{GCD: core.GCDDefault},
			CD:          core.Cooldown{Timer: priest.NewTimer(), Duration: shadowfiendCooldown},
		},

		Hot: core.DotConfig{
			SelfOnly:      true,
			Aura:          core.Aura{Label: "Shadowfiend"},
			NumberOfTicks: shadowfiendSwings,
			TickLength:    shadowfiendSwing,
			OnTick: func(sim *core.Simulation, target *core.Unit, _ *core.Dot) {
				target.AddMana(sim, target.MaxMana()*shadowfiendManaLeechFrac, manaMetrics)
			},
		},
		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.SelfHot().Apply(sim)
		},
	})
}

func (priest *Priest) registerDarkSacrifice() {
	priest.DarkSacrifice = make([]*core.Spell, DarkSacrificeRanks+1)
	for rank := 1; rank <= DarkSacrificeRanks; rank++ {
		if DarkSacrificeLevel[rank] > int(priest.Level) {
			continue
		}
		priest.DarkSacrifice[rank] = priest.GetOrRegisterSpell(priest.darkSacrificeConfig(rank))
	}
}

// darkSacrificeManaPerTick is one tick's mana: the rank's amount plus an even
// share of the caster's Spirit, so the whole cast gives the client's
// "${$o2+$SPI}" (the health taken plus 100% of Spirit, once).
func darkSacrificeManaPerTick(tickAmount, spirit float64) float64 {
	return tickAmount + spirit/float64(darkSacrificeTicks)
}

func (priest *Priest) darkSacrificeConfig(rank int) core.SpellConfig {
	actionID := core.ActionID{SpellID: DarkSacrificeSpellId[rank]}
	manaMetrics := priest.NewManaMetrics(actionID)
	tick := DarkSacrificeBaseDamage[rank][0]
	cooldown := time.Duration(DarkSacrificeCooldownMS[rank]) * time.Millisecond

	return core.SpellConfig{
		ActionID:      actionID,
		Flags:         core.SpellFlagNoOnCastComplete | core.SpellFlagAPL | core.SpellFlagHelpful,
		RequiredLevel: DarkSacrificeLevel[rank],
		Rank:          rank,

		Cast: core.CastConfig{
			DefaultCast: core.Cast{GCD: core.GCDDefault},
			CD:          core.Cooldown{Timer: priest.NewTimer(), Duration: cooldown},
		},

		Hot: core.DotConfig{
			SelfOnly:      true,
			Aura:          core.Aura{Label: fmt.Sprintf("Dark Sacrifice (Rank %d)", rank)},
			NumberOfTicks: darkSacrificeTicks,
			TickLength:    darkSacrificeTickLength,
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				target.RemoveHealth(sim, tick)
				target.AddMana(sim, darkSacrificeManaPerTick(tick, target.GetStat(stats.Spirit)), manaMetrics)
			},
		},
		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.SelfHot().Apply(sim)
		},
	}
}
