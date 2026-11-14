package warrior

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// rendTicks is how many three-second ticks each rank's bleed runs for.
// The generated arrays carry no duration column, so it is read by hand
// from the client's own duration_ms / period_ms
// (data/builds/1.60.1.69893/spellconst/warrior.json): 9/12/15/18/21/21/21
// seconds over a 3 s period. Listed for the data lane as the one column
// this ability still cannot read.
var rendTicks = [RendRanks + 1]int32{0, 3, 4, 5, 6, 7, 7, 7}

// rendAttackPowerPerTick is the share of attack power each Rend tick adds
// to the rank's flat amount, snapshot at the cast. No client column holds
// it (the rows carry only the flat 5 to 21), so it is a hand row from our
// logs: every logged tick runs above the flat amount and grows with the
// warrior, and the one warrior whose talents and stance a log names (the
// 8 November advanced report, COMBATANT_INFO without Improved Rend,
// Defensive Stance before every Rend) ticks 16 to 20 over 45 applications
// at a cast-time attack power of 599 to 769: 0.9 x (11 + 0.0129 x attack
// power), bootstrap 0.0126 to 0.0132. The other 34 warriors in our reports
// at levels 1 to 44 and ranks 1 to 7 fit the same share under their stance
// and Improved Rend multipliers. The Thunder Clap precedent.
const rendAttackPowerPerTick = 0.0129

func (warrior *Warrior) registerRendSpell() {
	rank := rankAtLevel(RendLevel[:], warrior.Level)
	damage := RendDamage[rank]
	casterLevel := int(warrior.Level)

	// Improved Rend: "Increases the Bleed damage done by your Rend
	// ability by 12%" at rank 1, "by 23%" at rank 2 and "by 35%" at
	// rank 3 (Arms node 105956). Vanilla's 15/25/35 is what stood here;
	// only the top rank happens to agree.
	damageMultiplier := improvedRendDamageMultiplier[rankIndex(warrior.Talents.ImprovedRend, improvedRendDamageMultiplier[:])]

	warrior.Rend = warrior.RegisterSpell(BattleStance|DefensiveStance, core.SpellConfig{
		SpellCode:      SpellCode_WarriorRend,
		ClassSpellMask: WarriorSpellMaskRend,
		ActionID:       core.ActionID{SpellID: RendSpellId[rank]},
		SpellSchool:    core.SpellSchoolPhysical,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagAPL | core.SpellFlagNoOnCastComplete | SpellFlagOffensive,

		RequiredLevel: RendLevel[rank],
		Rank:          rank,

		RageCost: core.RageCostOptions{
			Cost:   rageCost(RendManaCost[rank]),
			Refund: 0.8,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
		},

		CritDamageBonus:  warrior.rendCritDamageBonus(),
		DamageMultiplier: damageMultiplier,
		ThreatMultiplier: 1,
		ClientBaseDamage: damage.Range(casterLevel),

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "Rend",
				Tag:   "Rend",
			},
			NumberOfTicks: rendTicks[rank],
			TickLength:    time.Second * 3,
			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
				tick := damage.Roll(sim, casterLevel) + rendAttackPowerPerTick*dot.Spell.MeleeAttackPower(target)
				dot.Snapshot(target, tick, isRollover)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcOutcome(sim, target, spell.OutcomeMeleeSpecialHitNoHitCounter)
			if result.Landed() {
				spell.Dot(target).Apply(sim)
			} else {
				spell.IssueRefund(sim)
			}

			spell.DealOutcome(sim, result)
		},
	})

}
