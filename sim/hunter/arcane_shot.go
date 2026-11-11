package hunter

import (
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// arcaneShotBaseManaCostPercent is Blizzard's 10 November hotfix
// (t/2403040): "Arcane Shot mana cost changed to 11% of base Mana for all
// ranks of Arcane Shot." Client 1.60.1.70888's SpellPower rows for 3044 and
// 14281-14287 carry PowerCostPct 11 and no flat cost (25 to 190 before).
// Summon Hawk shares it ("same as Arcane Shot").
const arcaneShotBaseManaCostPercent = 0.11

func (hunter *Hunter) getArcaneShotConfig(rank int, timer *core.Timer) core.SpellConfig {
	spellId := [9]int32{0, 3044, 14281, 14282, 14283, 14284, 14285, 14286, 14287}[rank]
	damage := ArcaneShotDamage[rank]
	casterLevel := int(hunter.Level)
	level := [9]int{0, 6, 12, 20, 28, 36, 44, 52, 60}[rank]

	return core.SpellConfig{
		SpellCode:     SpellCode_HunterArcaneShot,
		ActionID:      core.ActionID{SpellID: spellId},
		SpellSchool:   core.SpellSchoolArcane,
		DefenseType:   core.DefenseTypeRanged,
		ProcMask:      core.ProcMaskRangedSpecial,
		Flags:         core.SpellFlagMeleeMetrics | core.SpellFlagAPL | SpellFlagShot,
		CastType:      proto.CastType_CastTypeRanged,
		Rank:          rank,
		RequiredLevel: level,
		MissileSpeed:  arcaneShotMissileSpeed,

		ManaCost: core.ManaCostOptions{
			BaseCost: arcaneShotBaseManaCostPercent,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer: timer,
				// Base cooldown is spellconst's category_cooldown_ms 6000 for
				// every rank of 3044/1428x/14287. Improved Arcane Shot
				// (talents/hunter.json node 105006) is max_rank 5 in Forever,
				// not Classic's 2, and each rank's own description reduces
				// the cooldown by 0.3s ("0.3/0.6/0.9/1.2/1.5 sec"), not
				// Classic's 0.2s/rank -- the old formula (200ms * rank,
				// capped at 2 ranks by the old UI) undershot the Forever
				// talent's real value at every rank.
				Duration: time.Second*6 - time.Millisecond*300*time.Duration(hunter.Talents.ImprovedArcaneShot),
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return hunter.DistanceFromTarget >= core.MinRangedAttackDistance
		},

		CritDamageBonus: hunter.mortalShots(),

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		// SpellEffect's EffectBonusCoefficient is 0 on every rank: spell
		// power adds nothing (Season of Discovery's reissues scale, these do not).
		BonusCoefficient: 0,
		ClientBaseDamage: damage.Range(casterLevel),

		// The server adds 15% of the hunter's ranged attack power to the
		// flat amount (attack_power_scaling.go).
		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := damage.Roll(sim, casterLevel) + arcaneShotAPCoeff*spell.RangedAttackPower(target, true)
			result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeRangedHitAndCrit)

			spell.WaitTravelTime(sim, func(sim *core.Simulation) {
				spell.DealDamage(sim, result)
			})
		},
	}
}

func (hunter *Hunter) registerArcaneShotSpell(timer *core.Timer) {
	maxRank := 8

	for i := 1; i <= maxRank; i++ {
		config := hunter.getArcaneShotConfig(i, timer)

		if config.RequiredLevel <= int(hunter.Level) {
			hunter.ArcaneShot = hunter.GetOrRegisterSpell(config)
		}
	}
}
