package druid

import (
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

// Lacerate is a Feral talent since client 1.60.1.70707's hotfix (node
// 104949, tier 3, behind Savage Fury 2): "Lacerates the enemy target,
// making them bleed for 20 damage over 15 sec plus 10% weapon damage per
// existing application of Lacerate on the target. Causes a high amount of
// threat. This effect stacks up to 5 times on the same target." Before it,
// every druid trained it at 42. A Bear Form bleed for 15 rage (cost 150) on
// the global cooldown; Shredding Attacks takes 1 rage a rank off it.
//
// The client tables carry every rank since the site's data pipeline layers
// the server's hotfix push into them (round 6): the hotfix's two low ranks
// 1328617 at 25 and 1328616 at 34 (4 and 6 a tick), then the trained ranks
// 414644, 1235826 and 1235827 at 42, 50 and 58 (10, 12 and 15 a tick),
// every 3 seconds for 15 seconds (LacerateTickDamage). The live tooltips
// (Wowhead's Forever spell tooltips, read 2026-10-31) state the same 20,
// 30, 50, 60 and 75 over 15 sec. The hidden companion spell 414647 used to
// be a flat 20% weapon damage hit; the live text makes the hit 10% weapon
// damage for each stack. The first Forever bear logs (foreverlogs.gg, 5 to
// 11 November 2026; sim-accuracy 2026-11-12 feral) show the hit counting the
// stack it adds: a level 43 bear's hits against its paw's white hit read
// 0.106, 0.206, 0.307, 0.44, 0.56 and 0.62 with 0 to 5 stacks already up, so
// the first application lands 10% weapon damage and the sixth 60%
// (lacerateHitDamage). The same logs show the bleed's ticks 15% larger from
// the second application on, the bear's Rend and Tear 5/5: the server puts
// the talent on the bleed it snapshots when the target already bleeds
// (bleedRendAndTearMultiplier).
//
// unconfirmed: the stack count the client tables do not hold (five, as the
// text says) with each stack adding the tick again; and the threat
// multiplier ("high amount of threat", the Primal Bite tier).
const (
	lacerateTicks      = 5
	lacerateTickLength = 3 * time.Second
	lacerateMaxStacks  = 5
	// lacerateWeaponDamagePerStack is the hit's 10% weapon damage for each
	// stack on the target once it lands, the one it adds included.
	lacerateWeaponDamagePerStack = 0.10
	lacerateRageCost             = 15.0
	lacerateMissRefund           = 0.8

	// shreddingAttacksLacerateRagePerRank is Shredding Attacks' "reduces
	// the Rage cost of your Lacerate ability by 1" a rank (node 104945).
	shreddingAttacksLacerateRagePerRank = 1.0
	shreddingAttacksMaxRank             = 3
)

// LacerateTickDamage is the bleed's per-tick effect for every rank (rank 0
// of the generated table is the hidden companion spell).
var LacerateTickDamage = lacerateTickEffects()

func lacerateTickEffects() [len(LacerateSpellId)]clientdamage.Effect {
	var effects [len(LacerateSpellId)]clientdamage.Effect
	for rank := 1; rank < len(LacerateSpellId); rank++ {
		effects[rank] = clientdamage.Effect{Amount: LacerateBaseDamage[rank][0], SpellLevel: LacerateLevel[rank]}
	}
	return effects
}

// lacerateRank is one live rank of Lacerate.
type lacerateRank struct {
	spellID int32
	tick    clientdamage.Effect
}

// lacerateRanks is the client's ladder, lowest first.
var lacerateRanks = lacerateClientRanks()

func lacerateClientRanks() []lacerateRank {
	ranks := make([]lacerateRank, 0, LacerateRanks)
	for rank := 1; rank <= LacerateRanks; rank++ {
		ranks = append(ranks, lacerateRank{spellID: LacerateSpellId[rank], tick: LacerateTickDamage[rank]})
	}
	return ranks
}

// lacerateRankAt is the highest live rank a druid of the level knows, its
// 1-based number, and 0 below the first.
func lacerateRankAt(level int32) (lacerateRank, int) {
	number := 0
	for index, rank := range lacerateRanks {
		if int32(rank.tick.SpellLevel) <= level {
			number = index + 1
		}
	}
	if number == 0 {
		return lacerateRank{}, 0
	}
	return lacerateRanks[number-1], number
}

func (druid *Druid) registerLacerateSpell() {
	if !druid.Talents.Lacerate {
		return
	}
	live, rank := lacerateRankAt(druid.Level)
	if rank == 0 {
		return
	}
	tick := live.tick
	casterLevel := int(druid.Level)
	rageCost := lacerateRageCost -
		shreddingAttacksLacerateRagePerRank*float64(clampRank(druid.Talents.ShreddingAttacks, shreddingAttacksMaxRank))

	druid.Lacerate = druid.RegisterSpell(Bear, core.SpellConfig{
		SpellCode:      SpellCode_DruidLacerate,
		ClassSpellMask: DruidSpellMaskLacerate,
		ActionID:       core.ActionID{SpellID: live.spellID},
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagIgnoreResists | core.SpellFlagAPL | SpellFlagOmen,

		Rank:          rank,
		RequiredLevel: tick.SpellLevel,

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
				dot.Snapshot(target, tick.Center(casterLevel)*float64(dot.GetStacks())*druid.bleedRendAndTear, false)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			rendAndTear := druid.RendAndTearMultiplier(target)
			baseDamage := lacerateHitDamage(spell.Dot(target).GetStacks(), spell.Unit.MHWeaponDamage(sim, spell.MeleeAttackPower(target)))
			baseDamage *= rendAndTear

			result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)
			if result.Landed() {
				druid.bleedRendAndTear = rendAndTear
				druid.stackLacerate(sim, spell.Dot(target))
			} else {
				spell.IssueRefund(sim)
			}
			spell.DealDamage(sim, result)
		},
	})
}

// lacerateHitDamage is the hit's weapon damage with existingStacks already
// on the target: 10% for each of them and for the stack the hit adds, so
// the hit on a full stack of five lands 60%.
func lacerateHitDamage(existingStacks int32, weaponDamage float64) float64 {
	return lacerateWeaponDamagePerStack * float64(existingStacks+1) * weaponDamage
}

// stackLacerate adds a stack and restarts the bleed's clock.
func (druid *Druid) stackLacerate(sim *core.Simulation, dot *core.Dot) {
	dot.ApplyOrReset(sim)
	dot.SetStacks(sim, min(dot.GetStacks()+1, lacerateMaxStacks))
	dot.TakeSnapshot(sim, false)
}
