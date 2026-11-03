package priest

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// vampiricEmbraceHealShare is the share of the priest's Shadow spell damage
// that heals each party member: 15286's effect 0 (SpellEffect 694156, aura 4)
// states 20, and the server text reads "healed for 20% of any Shadow spell
// damage you deal". Season of Discovery's 5% (the upstream number) is not
// Forever's. Improved Vampiric Embrace is not in Forever's trees.
const vampiricEmbraceHealShare = 0.20

// vampiricEmbraceHealing is the health each party member gains from one
// damage event on the embraced target: the share of the priest's own landed
// Shadow damage, direct or periodic. Another player's damage heals nobody.
func vampiricEmbraceHealing(owner *core.Unit, spell *core.Spell, result *core.SpellResult) float64 {
	if spell.Unit != owner || !result.Landed() || !spell.SpellSchool.Matches(core.SpellSchoolShadow) {
		return 0
	}
	return result.Damage * vampiricEmbraceHealShare
}

// vampiricEmbraceAuraConfig is the target's debuff: it heals the party off
// both Shadow hits and Shadow ticks, as 15286's proc row (SpellAuraOptions
// 128620, ProcTypeMask 0x800A0000: harmful magic spells and periodic damage
// taken) states.
func vampiricEmbraceAuraConfig(owner *core.Unit, target *core.Unit, healParty func(*core.Simulation, float64)) core.Aura {
	onDamage := func(_ *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
		if amount := vampiricEmbraceHealing(owner, spell, result); amount > 0 {
			healParty(sim, amount)
		}
	}
	return core.Aura{
		Label:                 "Vampiric Embrace (Health) - " + target.Label,
		OnSpellHitTaken:       onDamage,
		OnPeriodicDamageTaken: onDamage,
	}
}

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
	cooldown := time.Second * 30 // client 1.60.1.70425 (1 min before)

	partyPlayers := priest.Env.Raid.GetPlayerParty(&priest.Unit).Players
	healthMetrics := priest.NewHealthMetrics(actionID)

	healParty := func(sim *core.Simulation, amount float64) {
		for _, player := range partyPlayers {
			player.GetCharacter().GainHealth(sim, amount, healthMetrics)
		}
	}
	priest.VampiricEmbraceAuras = priest.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
		config := vampiricEmbraceAuraConfig(&priest.Unit, target, healParty)
		config.ActionID = actionID
		config.Duration = duration
		return target.GetOrRegisterAura(config)
	})

	priest.VampiricEmbrace = priest.RegisterSpell(core.SpellConfig{
		ActionID:       actionID,
		ClassSpellMask: PriestSpellMaskVampiricEmbrace,
		SpellSchool:    core.SpellSchoolShadow,
		DefenseType:    core.DefenseTypeMagic,
		ProcMask:       core.ProcMaskEmpty,
		Flags:          SpellFlagPriest | core.SpellFlagAPL,
		RequiredLevel:  30,

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
