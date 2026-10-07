package priest

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

func (priest *Priest) registerVampiricEmbraceSpell() {
	if !priest.Talents.VampiricEmbrace {
		return
	}

	actionID := core.ActionID{SpellID: 15286}
	manaCost := 40.0
	// 15286 is the cast-on-self spell the client's own duplicate-id
	// cleanup dropped in favor of 15290 when generating
	// constants_auto_gen.go (15290's own columns are all zero - see that
	// file's "Vampiric Embrace rank 0: kept id 15290 ... dropped 15286"
	// comment) - but 15286 is the entry that actually carries this
	// ability's real numbers (cost 40, cooldown 60s, duration 30s,
	// required level 30), so this file reads those straight off the
	// client's per-id spell data instead of the misleadingly-canonical
	// generated constant.
	duration := time.Second * 30
	cooldown := time.Minute * 1

	partyPlayers := priest.Env.Raid.GetPlayerParty(&priest.Unit).Players
	healthMetrics := priest.NewHealthMetrics(actionID)
	// FOREVER: Improved Vampiric Embrace is not in the client's trees.
	// healthReturnedMultuplier := 0.05 + 0.05*float64(priest.Talents.ImprovedVampiricEmbrace)
	healthReturnedMultuplier := 0.05

	priest.VampiricEmbraceAuras = priest.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
		return target.GetOrRegisterAura(core.Aura{
			ActionID: actionID,
			Label:    "Vampiric Embrace (Health) - " + target.Label,
			Duration: duration,
			OnSpellHitTaken: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
				if result.Landed() && spell.SpellSchool.Matches(core.SpellSchoolShadow) {
					healthGained := result.Damage * healthReturnedMultuplier
					for _, player := range partyPlayers {
						player.GetCharacter().GainHealth(sim, healthGained, healthMetrics)
					}
				}
			},
		})
	})

	priest.VampiricEmbrace = priest.RegisterSpell(core.SpellConfig{
		ActionID:      actionID,
		SpellSchool:   core.SpellSchoolShadow,
		DefenseType:   core.DefenseTypeMagic,
		ProcMask:      core.ProcMaskEmpty,
		Flags:         SpellFlagPriest | core.SpellFlagAPL,
		RequiredLevel: 30,

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			CD: core.Cooldown{
				Timer:    priest.NewTimer(),
				Duration: cooldown,
			},
		},

		// The debuff this applies lives on the target, not the caster, so
		// RelatedSelfBuff (documented as the caster's own aura) is used
		// here only so compare.go's conformance report can read its
		// Duration without running a sim - see engineDuration's doc
		// comment in sim/conformance/compare.go.
		RelatedSelfBuff: priest.VampiricEmbraceAuras.Get(priest.CurrentTarget),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcOutcome(sim, target, spell.OutcomeMagicHit)
			if result.Landed() {
				priest.VampiricEmbraceAuras.Get(target).Activate(sim)
			}
			spell.DealOutcome(sim, result)
		},
	})
}
