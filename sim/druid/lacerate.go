package druid

import (
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

// Lacerate (spells 414644, 1235826 and 1235827, learned at 42, 50 and 58;
// build 1.60.1.70009): a Bear Form bleed for 15 rage (cost 150) on the
// global cooldown. Each rank's periodic-damage aura (effect 6, aura 3)
// pays 10, 12 and 15 every 3 seconds for 15 seconds, and the client's
// hidden companion spell 414647 ("Lacerate", effect 31) is a 20% weapon
// damage hit. Shredding Attacks takes 1 rage a rank off it.
//
// unconfirmed: that 414647 is Lacerate's direct hit; that the bleed stacks
// (the client tables hold no stack count: five is the Season of Discovery
// and Wrath figure) with each stack adding the tick again; the dummy effect
// (amount 10) on every rank's own row, which is read as nothing; and the
// threat multiplier ("high amount of threat" in the Primal Bite text, taken
// to be the same tier here).
const (
	lacerateTicks              = 5
	lacerateTickLength         = 3 * time.Second
	lacerateMaxStacks          = 5
	lacerateDirectWeaponDamage = 0.20
	lacerateRageCost           = 15.0
	lacerateMissRefund         = 0.8

	// shreddingAttacksLacerateRagePerRank is Shredding Attacks' "reduces
	// the Rage cost of your Lacerate ability by 1" a rank (node 104945).
	shreddingAttacksLacerateRagePerRank = 1.0
	shreddingAttacksMaxRank             = 3
)

// LacerateTickDamage is the bleed's per-tick effect for ranks 1 to 3 (rank
// 0 of the generated table is the hidden companion spell).
var LacerateTickDamage = lacerateTickEffects()

func lacerateTickEffects() [len(LacerateSpellId)]clientdamage.Effect {
	var effects [len(LacerateSpellId)]clientdamage.Effect
	for rank := 1; rank < len(LacerateSpellId); rank++ {
		effects[rank] = clientdamage.Effect{Amount: LacerateBaseDamage[rank][0], SpellLevel: LacerateLevel[rank]}
	}
	return effects
}

func (druid *Druid) registerLacerateSpell() {
	rank := core.HighestRankAtLevel(LacerateLevel[1:], druid.Level)
	if rank == 0 {
		return
	}
	tick := LacerateTickDamage[rank]
	casterLevel := int(druid.Level)
	rageCost := lacerateRageCost -
		shreddingAttacksLacerateRagePerRank*float64(clampRank(druid.Talents.ShreddingAttacks, shreddingAttacksMaxRank))

	druid.Lacerate = druid.RegisterSpell(Bear, core.SpellConfig{
		SpellCode:      SpellCode_DruidLacerate,
		ClassSpellMask: DruidSpellMaskLacerate,
		ActionID:       core.ActionID{SpellID: LacerateSpellId[rank]},
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagIgnoreResists | core.SpellFlagAPL | SpellFlagOmen,

		Rank:          rank,
		RequiredLevel: LacerateLevel[rank],

		RageCost: core.RageCostOptions{
			Cost:   rageCost,
			Refund: lacerateMissRefund,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			IgnoreHaste: true,
		},

		DamageMultiplier: 1,
		ThreatMultiplier: HighThreatMultiplier,
		BonusCoefficient: 1,
		ClientBaseDamage: tick.Range(casterLevel),

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label:     "Lacerate",
				MaxStacks: lacerateMaxStacks,
			},
			NumberOfTicks: lacerateTicks,
			TickLength:    lacerateTickLength,
			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, _ bool) {
				// A new stack changes the tick, so every cast re-snapshots.
				dot.Snapshot(target, tick.Center(casterLevel)*float64(dot.GetStacks()), false)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := lacerateDirectWeaponDamage * spell.Unit.MHWeaponDamage(sim, spell.MeleeAttackPower(target))
			baseDamage *= druid.RendAndTearMultiplier(target)

			result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)
			if result.Landed() {
				druid.stackLacerate(sim, spell.Dot(target))
			} else {
				spell.IssueRefund(sim)
			}
			spell.DealDamage(sim, result)
		},
	})
}

// stackLacerate adds a stack and restarts the bleed's clock.
func (druid *Druid) stackLacerate(sim *core.Simulation, dot *core.Dot) {
	dot.ApplyOrReset(sim)
	dot.SetStacks(sim, min(dot.GetStacks()+1, lacerateMaxStacks))
	dot.TakeSnapshot(sim, false)
}
