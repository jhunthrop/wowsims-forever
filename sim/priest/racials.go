package priest

import (
	"fmt"
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// The Forever priest racial heals, from the 1.60.1.70707 client (rows
// quoted in sim/priest/healing/racials_test.go). Both are free, on the
// GCD, and share nothing with another cooldown: a 10 minute category
// cooldown of their own. Neither rotation names them, so each is a major
// cooldown the autocast line fires on the healer's target.
const (
	racialHealCooldown = 10 * time.Minute

	// divineGraceBelowHealth is Divine Grace's gate (effect 1, 50).
	divineGraceBelowHealth  = 0.5
	divineGraceCoefficient  = 0.429
	contingencyBelowHealth  = 0.35
	contingencyWardDuration = 30 * time.Second
	contingencyHotTicks     = 5
	contingencyHotPeriod    = 3 * time.Second
	contingencyCoefficient  = 0.2
)

type divineGraceRank struct {
	spellID int32
	heal    clientdamage.Effect
}

// divineGraceRanks are the human priest's ranks: SkillLineAbility skill
// line 56, RaceMasks 1.
var divineGraceRanks = []divineGraceRank{
	{1277370, clientdamage.Effect{Amount: 149, Variance: 0.23684211, PerLevel: 2.4, SpellLevel: 10, MaxLevel: 16}},
	{1277371, clientdamage.Effect{Amount: 276, Variance: 0.21088400, PerLevel: 3.4, SpellLevel: 18, MaxLevel: 24}},
	{1277372, clientdamage.Effect{Amount: 458, Variance: 0.19393939, PerLevel: 4.5, SpellLevel: 26, MaxLevel: 32}},
	{1277374, clientdamage.Effect{Amount: 595, Variance: 0.18518500, PerLevel: 5.3, SpellLevel: 34, MaxLevel: 40}},
	{1277376, clientdamage.Effect{Amount: 856, Variance: 0.17505500, PerLevel: 6.4, SpellLevel: 42, MaxLevel: 48}},
	{1277377, clientdamage.Effect{Amount: 1143, Variance: 0.16957600, PerLevel: 7.4, SpellLevel: 50, MaxLevel: 56}},
	{1277378, clientdamage.Effect{Amount: 1383, Variance: 0.16493399, PerLevel: 8.2, SpellLevel: 58, MaxLevel: 64}},
}

type contingencyRank struct {
	wardID, shieldID, hotID int32
	level                   int
	absorb, tick            float64
}

// contingencyRanks are the gnome priest's ranks: skill line 613, RaceMasks
// 64. The ward names its absorb and its heal over time by id.
var contingencyRanks = []contingencyRank{
	{1277462, 1277463, 1277456, 20, 155, 25},
	{1277634, 1277464, 1277457, 30, 290, 40},
	{1277638, 1277465, 1277458, 40, 467, 56},
	{1277639, 1277467, 1277459, 50, 596, 102},
	{1277640, 1277466, 1277460, 60, 926, 134},
}

// registerRacialHeals registers the race's heal, if it has one.
func (priest *Priest) registerRacialHeals() {
	switch priest.Race {
	case proto.Race_RaceHuman:
		priest.registerDivineGrace()
	case proto.Race_RaceGnome:
		priest.registerContingencyPlan()
	}
}

// racialHealTarget is who the racial heals go on: the healer's current
// target (the tank in the healing sim), never the priest. A major cooldown
// casts a helpful spell on its caster, so a cast on the priest herself is
// that autocast and goes to the current target; nil when there is none.
func (priest *Priest) racialHealTarget(target *core.Unit) *core.Unit {
	if target == &priest.Unit {
		target = priest.CurrentTarget
	}
	if target != nil && target != &priest.Unit && !target.IsOpponent(&priest.Unit) && target.HasHealthBar() {
		return target
	}
	return nil
}

func (priest *Priest) racialHealSpell(id int32, level int) core.SpellConfig {
	return core.SpellConfig{
		ActionID:    core.ActionID{SpellID: id},
		SpellSchool: core.SpellSchoolHoly,
		DefenseType: core.DefenseTypeMagic,
		ProcMask:    core.ProcMaskSpellHealing,
		Flags:       core.SpellFlagHelpful,

		RequiredLevel: level,

		Cast: core.CastConfig{
			DefaultCast: core.Cast{GCD: core.GCDDefault},
			CD:          core.Cooldown{Timer: priest.NewTimer(), Duration: racialHealCooldown},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
	}
}

// registerDivineGrace: "Instantly heals a friendly target below 50% Health
// and removes Weakened Soul from that target. Cannot be cast on self."
func (priest *Priest) registerDivineGrace() {
	var rank *divineGraceRank
	for i := range divineGraceRanks {
		if divineGraceRanks[i].heal.SpellLevel <= int(priest.Level) {
			rank = &divineGraceRanks[i]
		}
	}
	if rank == nil {
		return
	}
	config := priest.racialHealSpell(rank.spellID, rank.heal.SpellLevel)
	config.BonusCoefficient = divineGraceCoefficient
	config.ClientBaseDamage = rank.heal.Range(int(priest.Level))
	config.ExtraCastCondition = func(_ *core.Simulation, target *core.Unit) bool {
		target = priest.racialHealTarget(target)
		return target != nil && target.CurrentHealthPercent() < divineGraceBelowHealth
	}
	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		target = priest.racialHealTarget(target)
		spell.CalcAndDealHealing(sim, target, rank.heal.Roll(sim, int(priest.Level)), spell.OutcomeHealingCrit)
		if weakened := priest.weakenedSoul(target); weakened != nil {
			weakened.Deactivate(sim)
		}
	}
	priest.DivineGrace = priest.RegisterSpell(config)
	priest.AddMajorCooldown(core.MajorCooldown{
		Spell: priest.DivineGrace,
		Type:  core.CooldownTypeUnknown,
		ShouldActivate: func(sim *core.Simulation, _ *core.Character) bool {
			return priest.DivineGrace.CanCast(sim, &priest.Unit)
		},
	})
}

// registerContingencyPlan: a ward for 30 s; the next damage that drops its
// holder below 35% health gives a shield and a heal over time. One ward a
// target.
func (priest *Priest) registerContingencyPlan() {
	var rank *contingencyRank
	for i := range contingencyRanks {
		if contingencyRanks[i].level <= int(priest.Level) {
			rank = &contingencyRanks[i]
		}
	}
	if rank == nil {
		return
	}

	priest.ContingencyShield = priest.RegisterSpell(core.SpellConfig{
		ActionID:         core.ActionID{SpellID: rank.shieldID},
		SpellSchool:      core.SpellSchoolHoly,
		DefenseType:      core.DefenseTypeMagic,
		ProcMask:         core.ProcMaskSpellHealing,
		Flags:            core.SpellFlagHelpful | core.SpellFlagNoOnCastComplete,
		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		Shield: core.ShieldConfig{
			Aura: core.Aura{Label: "Contingency Plan Shield", Duration: contingencyHotTicks * contingencyHotPeriod},
		},
	})
	priest.ContingencyHot = priest.RegisterSpell(core.SpellConfig{
		ActionID:         core.ActionID{SpellID: rank.hotID},
		SpellSchool:      core.SpellSchoolHoly,
		DefenseType:      core.DefenseTypeMagic,
		ProcMask:         core.ProcMaskSpellHealing,
		Flags:            core.SpellFlagHelpful | core.SpellFlagNoOnCastComplete,
		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		Hot: core.DotConfig{
			Aura:             core.Aura{Label: "Contingency Plan Heal"},
			NumberOfTicks:    contingencyHotTicks,
			TickLength:       contingencyHotPeriod,
			BonusCoefficient: contingencyCoefficient,
			DynamicSnapshot:  true,
			OnSnapshot: func(_ *core.Simulation, target *core.Unit, dot *core.Dot, _ bool) {
				snapshotHeal(dot, target, rank.tick)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotHealing(sim, target, dot.OutcomeTick)
			},
		},
		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.Hot(target).Apply(sim)
		},
	})
	priest.ContingencyWards = priest.NewRaidAuraArray(func(target *core.Unit) *core.Aura {
		return target.GetOrRegisterAura(core.Aura{
			Label:    fmt.Sprintf("Contingency Plan Ward-%d", priest.UnitIndex),
			ActionID: core.ActionID{SpellID: rank.wardID},
			Duration: contingencyWardDuration,
		})
	})

	config := priest.racialHealSpell(rank.wardID, rank.level)
	config.Flags |= core.SpellFlagNoOnCastComplete
	config.ExtraCastCondition = func(_ *core.Simulation, target *core.Unit) bool {
		target = priest.racialHealTarget(target)
		return target != nil && int(target.UnitIndex) < len(priest.ContingencyWards) && !priest.ContingencyWards.Get(target).IsActive()
	}
	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, _ *core.Spell) {
		priest.ContingencyWards.Get(priest.racialHealTarget(target)).Activate(sim)
	}
	priest.ContingencyPlan = priest.RegisterSpell(config)
	priest.AddMajorCooldown(core.MajorCooldown{
		Spell: priest.ContingencyPlan,
		Type:  core.CooldownTypeUnknown,
		ShouldActivate: func(sim *core.Simulation, _ *core.Character) bool {
			return priest.ContingencyPlan.CanCast(sim, &priest.Unit)
		},
	})

	priest.Env.Raid.OnRaidDamage(func(sim *core.Simulation, unit *core.Unit, _ float64, _ bool) {
		if int(unit.UnitIndex) >= len(priest.ContingencyWards) {
			return
		}
		ward := priest.ContingencyWards.Get(unit)
		if ward == nil || !ward.IsActive() || unit.CurrentHealthPercent() >= contingencyBelowHealth {
			return
		}
		ward.Deactivate(sim)
		priest.ContingencyShield.Shield(unit).Apply(sim, rank.absorb)
		priest.ContingencyHot.Cast(sim, unit)
	})
}
