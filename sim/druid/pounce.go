package druid

import (
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

// PounceTickDamage is the periodic amount of each rank's "Pounce Bleed"
// spell (9007, 9824, 9826): the client keeps the bleed apart from the
// Pounce cast (9005, 9823, 9827), whose own effects are the stun, the
// trigger of that spell and one combo point, so neither the generated
// PounceBaseDamage row (all zero) nor the vendored spellconst file carry
// it. Source: 1.60.1.70009 SpellEffect.csv, effect 0 (aura 3, every 3 s)
// of those three spells. TestPounceTickDamage pins the values.
var PounceTickDamage = [PounceRanks + 1]clientdamage.Effect{
	{},
	{Amount: 15, SpellLevel: 36},
	{Amount: 20, SpellLevel: 46},
	{Amount: 25, SpellLevel: 56},
}

const (
	// pounceBleedTicks and pounceBleedTickLength: Pounce Bleed lasts 18 s
	// in 3 s periods on every rank (1.60.1.70009 SpellDuration for 9007,
	// 9824, 9826).
	pounceBleedTicks      int32 = 6
	pounceBleedTickLength       = 3 * time.Second
	// pounceComboPoints is effect 2 (energize, misc 4: combo points) of
	// every rank. The 2 s stun (effect 0, aura 12) is not modelled: no
	// encounter target can be stunned.
	pounceComboPoints int32 = 1
)

func (druid *Druid) registerPounceSpell() {
	for rank := PounceRanks; rank >= 1; rank-- {
		if druid.Level >= int32(PounceLevel[rank]) {
			druid.Pounce = druid.RegisterSpell(Cat, druid.newPounceSpellConfig(rank))
			return
		}
	}
}

func (druid *Druid) newPounceSpellConfig(rank int) core.SpellConfig {
	tickDamage := PounceTickDamage[rank]
	casterLevel := int(druid.Level)

	return core.SpellConfig{
		SpellCode:      SpellCode_DruidPounce,
		ClassSpellMask: DruidSpellMaskPounce,
		ActionID:       core.ActionID{SpellID: PounceSpellId[rank]},
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagIgnoreResists | core.SpellFlagBinary | core.SpellFlagAPL | SpellFlagOmen | SpellFlagBuilder,

		RequiredLevel: PounceLevel[rank],
		Rank:          rank,

		EnergyCost: core.EnergyCostOptions{
			Cost:   PounceManaCost[rank], // the client's cost column; the energy this spell spends
			Refund: 0.8,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: time.Second,
			},
			IgnoreHaste: true,
		},
		// "Must be prowling and behind the target": scripted requirements
		// the client data carries no flag for, mirrored from Ravage and the
		// rogue's Garrote.
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return druid.IsProwling() && !druid.PseudoStats.InFrontOfTarget
		},

		DamageMultiplierAdditive: druid.savageFuryDamageMultiplier(),
		DamageMultiplier:         1,
		ThreatMultiplier:         1,

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "Pounce",
			},
			NumberOfTicks: pounceBleedTicks,
			TickLength:    pounceBleedTickLength,
			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
				dot.Snapshot(target, tickDamage.Center(casterLevel), isRollover)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			druid.BreakProwl(sim)

			result := spell.CalcOutcome(sim, target, spell.OutcomeMeleeSpecialHitNoHitCounter)
			if result.Landed() {
				druid.AddComboPoints(sim, pounceComboPoints, target, spell.ComboPointMetrics())
				spell.Dot(target).Apply(sim)
			} else {
				spell.IssueRefund(sim)
			}
			spell.DealOutcome(sim, result)
		},

		ExpectedTickDamage: func(sim *core.Simulation, target *core.Unit, spell *core.Spell, _ bool) *core.SpellResult {
			return spell.CalcPeriodicDamage(sim, target, tickDamage.Center(casterLevel), spell.OutcomeExpectedMagicAlwaysHit)
		},
	}
}
