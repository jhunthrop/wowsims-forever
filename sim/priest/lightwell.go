package priest

import (
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

// Lightwell, read from the 1.60.1.70009 client (rows quoted in
// sim/priest/healing/lightwell_test.go): the cast (spells 724 / 27870 /
// 27871, levels 40 / 50 / 60, 225 / 295 / 365 mana, a 1.5 s cast and GCD, a
// 10 minute category cooldown) leaves a well that stands 180 s or for 5
// charges. A raid member clicks it to receive its renew (spells 7001 /
// 27873 / 27874), 160 / 233 / 320 health every 2 s for 10 s with no
// spell-power coefficient, and "being attacked cancels the effect".
//
// Modelled in the healing sim's fake raid: a fake member (not the tank,
// who is attacked every swing) whose health falls below
// lightwellClickBelowHealth when the damage model hurts it clicks at once
// if a charge is left, and any damage the model deals a member cancels
// that member's renew. Assumptions: the click threshold, and that a click
// takes no time.

const (
	lightwellCharges  = 5
	lightwellDuration = 180 * time.Second
	lightwellCooldown = 10 * time.Minute
	lightwellCastTime = 1500 * time.Millisecond

	lightwellRenewTicks      = 5
	lightwellRenewTickLength = 2 * time.Second

	// lightwellClickBelowHealth is the health share under which a hurt
	// member clicks the well. Assumption: no client row states how hurt a
	// member must be before it walks over.
	lightwellClickBelowHealth = 0.9
)

type lightwellRank struct {
	castID   int32
	renewID  int32
	level    int
	manaCost float64
	// tick is the renew's amount a tick (EffectBasePointsF).
	tick float64
}

var lightwellRanks = []lightwellRank{
	{castID: 724, renewID: 7001, level: 40, manaCost: 225, tick: 160},
	{castID: 27870, renewID: 27873, level: 50, manaCost: 295, tick: 233},
	{castID: 27871, renewID: 27874, level: 60, manaCost: 365, tick: 320},
}

// lightwellState is the one well a priest keeps up and the renew its
// clicks give.
type lightwellState struct {
	// renew is the renew of the rank last cast; kept after the well is
	// spent so a renew still ticking can be cancelled.
	renew *core.Spell
}

// registerLightwell registers the cast and its renew at every rank the
// priest has reached. Only the healing specs call it (RegisterHealingSpells).
func (priest *Priest) registerLightwell() {
	priest.Lightwell = make([]*core.Spell, len(lightwellRanks)+1)
	var topRank *lightwellRank
	for i := range lightwellRanks {
		if lightwellRanks[i].level <= int(priest.Level) {
			topRank = &lightwellRanks[i]
		}
	}
	if topRank == nil {
		return
	}

	priest.LightwellAura = priest.RegisterAura(core.Aura{
		Label:     "Lightwell",
		ActionID:  core.ActionID{SpellID: topRank.castID},
		Duration:  lightwellDuration,
		MaxStacks: lightwellCharges,
	})
	cooldown := core.Cooldown{Timer: priest.NewTimer(), Duration: lightwellCooldown}

	for i, rank := range lightwellRanks {
		if rank.level > int(priest.Level) {
			continue
		}
		renew := priest.registerLightwellRenew(i+1, rank)
		priest.Lightwell[i+1] = priest.RegisterSpell(priest.newLightwellCast(i+1, rank, renew, cooldown))
	}
	priest.Env.Raid.OnRaidDamage(priest.onRaidDamageForLightwell)
}

func (priest *Priest) registerLightwellRenew(rankNumber int, rank lightwellRank) *core.Spell {
	tick := clientdamage.Effect{Amount: rank.tick}
	return priest.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: rank.renewID},
		SpellSchool: core.SpellSchoolHoly,
		DefenseType: core.DefenseTypeMagic,
		ProcMask:    core.ProcMaskSpellHealing,
		Flags:       core.SpellFlagHelpful | core.SpellFlagNoOnCastComplete,

		RequiredLevel: rank.level,
		Rank:          rankNumber,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		ClientBaseDamage: tick.Range(int(priest.Level)),

		Hot: core.DotConfig{
			Aura: core.Aura{
				Label: "Lightwell Renew",
			},
			NumberOfTicks: lightwellRenewTicks,
			TickLength:    lightwellRenewTickLength,

			OnSnapshot: func(_ *core.Simulation, target *core.Unit, dot *core.Dot, _ bool) {
				snapshotHeal(dot, target, rank.tick)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotHealing(sim, target, dot.Spell.OutcomeHealing)
			},
		},
		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.Hot(target).Apply(sim)
		},
	})
}

func (priest *Priest) newLightwellCast(rankNumber int, rank lightwellRank, renew *core.Spell, cooldown core.Cooldown) core.SpellConfig {
	return core.SpellConfig{
		ActionID:       core.ActionID{SpellID: rank.castID},
		SpellCode:      SpellCode_PriestLightwell,
		ClassSpellMask: PriestSpellMaskLightwell,
		SpellSchool:    core.SpellSchoolHoly,
		DefenseType:    core.DefenseTypeMagic,
		ProcMask:       core.ProcMaskEmpty,
		Flags:          priestHealFlags,

		RequiredLevel: rank.level,
		Rank:          rankNumber,

		// The aura is the well: one for every rank, 180 s as the client's
		// duration index 26 states for each.
		RelatedSelfBuff: priest.LightwellAura,

		ManaCost: core.ManaCostOptions{FlatCost: rank.manaCost},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD:      core.GCDDefault,
				CastTime: lightwellCastTime,
			},
			CD: cooldown,
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			priest.lightwell.renew = renew
			priest.LightwellAura.Activate(sim)
			priest.LightwellAura.SetStacks(sim, lightwellCharges)
		},
	}
}

// LightwellCharges is how many charges the well standing now has left.
func (priest *Priest) LightwellCharges() int32 {
	if priest.LightwellAura == nil || !priest.LightwellAura.IsActive() {
		return 0
	}
	return priest.LightwellAura.GetStacks()
}

// onRaidDamageForLightwell hears the damage model hurt a fake member:
// that cancels the member's renew, and a hurt member who is not the tank
// then clicks the well if it has a charge.
func (priest *Priest) onRaidDamageForLightwell(sim *core.Simulation, unit *core.Unit, _ float64, isTank bool) {
	renew := priest.lightwell.renew
	if renew == nil {
		return
	}
	if hot := renew.Hot(unit); hot != nil && hot.IsActive() {
		hot.Cancel(sim)
	}
	if isTank || priest.LightwellCharges() == 0 || unit.CurrentHealthPercent() >= lightwellClickBelowHealth {
		return
	}
	priest.LightwellAura.RemoveStack(sim)
	renew.Hot(unit).Apply(sim)
}
