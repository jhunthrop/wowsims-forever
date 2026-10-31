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
// The client tables still carry the trained ranks (414644, 1235826 and
// 1235827 at 42, 50 and 58, ticking 10, 12 and 15 every 3 seconds for 15
// seconds: LacerateTickDamage). The hotfix added two low ranks the tables do
// not have; the live tooltips (Wowhead's Forever spell tooltips, read
// 2026-10-31) state 20 over 15 sec at 25 (1328617, the talent's own spell)
// and 30 at 34 (1328616), and the same 50, 60 and 75 for the trained ranks.
// The hidden companion spell 414647 used to be a flat 20% weapon damage hit;
// the live text makes the hit 10% weapon damage for each stack already on
// the target, so the first application lands no weapon damage.
//
// unconfirmed: the stack count the client tables do not hold (five, as the
// text says) with each stack adding the tick again; and the threat
// multiplier ("high amount of threat", the Primal Bite tier).
const (
	lacerateTicks      = 5
	lacerateTickLength = 3 * time.Second
	lacerateMaxStacks  = 5
	// lacerateWeaponDamagePerStack is the hit's 10% weapon damage for each
	// stack already on the target.
	lacerateWeaponDamagePerStack = 0.10
	lacerateRageCost             = 15.0
	lacerateMissRefund           = 0.8

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

// lacerateRank is one live rank of Lacerate.
type lacerateRank struct {
	spellID int32
	tick    clientdamage.Effect
}

// lacerateRanks is the live ladder, lowest first: the hotfix's two low
// ranks (a total over five ticks), then the client's trained ranks.
var lacerateRanks = []lacerateRank{
	{spellID: 1328617, tick: clientdamage.Effect{Amount: 20.0 / lacerateTicks, SpellLevel: 25}},
	{spellID: 1328616, tick: clientdamage.Effect{Amount: 30.0 / lacerateTicks, SpellLevel: 34}},
	{spellID: LacerateSpellId[1], tick: LacerateTickDamage[1]},
	{spellID: LacerateSpellId[2], tick: LacerateTickDamage[2]},
	{spellID: LacerateSpellId[3], tick: LacerateTickDamage[3]},
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
				dot.Snapshot(target, tick.Center(casterLevel)*float64(dot.GetStacks()), false)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := lacerateHitDamage(spell.Dot(target).GetStacks(), spell.Unit.MHWeaponDamage(sim, spell.MeleeAttackPower(target)))
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

// lacerateHitDamage is the hit's weapon damage with existingStacks already
// on the target.
func lacerateHitDamage(existingStacks int32, weaponDamage float64) float64 {
	return lacerateWeaponDamagePerStack * float64(existingStacks) * weaponDamage
}

// stackLacerate adds a stack and restarts the bleed's clock.
func (druid *Druid) stackLacerate(sim *core.Simulation, dot *core.Dot) {
	dot.ApplyOrReset(sim)
	dot.SetStacks(sim, min(dot.GetStacks()+1, lacerateMaxStacks))
	dot.TakeSnapshot(sim, false)
}
