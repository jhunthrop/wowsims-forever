package hunter

import (
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// multiShotLevel is Multi-Shot's single learn level; source: 1.60.1.70009
// client spell data. Classic's five-rank progression (2643/14288/14289/
// 14290/25294, levels 18/30/42/54/60, flat mana 100-230, flat bonus
// damage 0-150) does not exist in Forever's spellconst/hunter.json:
// spell 2643 is the *only* Multi-Shot entry the client carries, still
// learned at 18, with cost_type 0 (mana) but cost 0 -- see
// multiShotBaseManaCostPercent below for why the flat "cost" column is
// not the real number here -- category_cooldown_ms 6000 (6s, not
// Classic's 10s) and effect amount 0 (code 121, "weapon damage + flat
// amount": pure weapon/ranged-AP damage, no per-rank flat bonus). The
// engine used to keep Classic's 4 extra rank ids and their mana/damage
// numbers; none of those ids resolve in the pinned client, so a level-30+
// hunter was overpaying (140-230 mana, still on a 10s clock) for a shot
// Forever prices at a fraction of that on a 6s clock.
const multiShotLevel = 18

// multiShotBaseManaCostPercent is Multi-Shot's cost as a fraction of base
// mana. spellconst's flat "cost" field is 0 for spell 2643 (the pipeline
// only carries the client's flat-mana column, not its percent-of-base-
// mana column), so this is corroborated directly against Wowhead's
// Forever tooltip for spell 2643 ("13.9% of base mana") instead --
// scaling with level the same way Classic's percent-cost spells always
// did, just via one spell id instead of Classic's five.
const multiShotBaseManaCostPercent = 0.139

func (hunter *Hunter) getMultiShotConfig(timer *core.Timer) core.SpellConfig {
	const spellId = 2643

	// Pool-sized ceiling, live-bounded loop; see APLActionMultidot and
	// warrior registerWhirlwindSpell.
	maxHits := min(3, len(hunter.Env.Encounter.AllTargetUnits))
	results := make([]*core.SpellResult, maxHits)

	return core.SpellConfig{
		SpellCode:      SpellCode_HunterMultiShot,
		ClassSpellMask: HunterSpellMaskMultiShot,
		ActionID:       core.ActionID{SpellID: spellId},
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeRanged,
		ProcMask:       core.ProcMaskRangedSpecial,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagAPL | SpellFlagShot,
		CastType:       proto.CastType_CastTypeRanged,
		RequiredLevel:  multiShotLevel,
		MissileSpeed:   24,

		ManaCost: core.ManaCostOptions{
			BaseCost: multiShotBaseManaCostPercent,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD:      core.GCDDefault,
				CastTime: time.Millisecond * 500,
			},
			ModifyCast: func(sim *core.Simulation, spell *core.Spell, cast *core.Cast) {
				cast.CastTime = spell.CastTime()
				hunter.Unit.AutoAttacks.CancelAutoSwing(sim)
			},
			IgnoreHaste: true, // Hunter GCD is locked at 1.5s
			CD: core.Cooldown{
				Timer:    timer,
				Duration: time.Second * 6,
			},
			CastTime: func(spell *core.Spell) time.Duration {
				return time.Duration(float64(spell.DefaultCast.CastTime) / hunter.RangedSwingSpeed())
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return hunter.DistanceFromTarget >= core.MinRangedAttackDistance
		},

		CritDamageBonus: hunter.mortalShots(),

		DamageMultiplier: 1 + .05*float64(hunter.Talents.Barrage),
		ThreatMultiplier: 1,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			curTarget := target

			// Fixed for this cast so the landing loop below deals
			// exactly the results the hit loop calculated.
			numHits := min(maxHits, len(sim.Encounter.TargetUnits))
			for hitIndex := 0; hitIndex < numHits; hitIndex++ {
				// spellconst's effect amount is 0 (no flat bonus): every
				// target takes pure normalized weapon + ammo damage.
				baseDamage := hunter.AutoAttacks.Ranged().CalculateNormalizedWeaponDamage(sim, spell.RangedAttackPower(target, false)) +
					hunter.AmmoDamageBonus

				results[hitIndex] = spell.CalcDamage(sim, curTarget, baseDamage, spell.OutcomeRangedHitAndCrit)

				curTarget = sim.Environment.NextTargetUnit(curTarget)
			}
			hunter.Unit.AutoAttacks.EnableAutoSwing(sim)
			spell.WaitTravelTime(sim, func(s *core.Simulation) {
				for hitIndex := 0; hitIndex < numHits; hitIndex++ {
					spell.DealDamage(sim, results[hitIndex])

					curTarget = sim.Environment.NextTargetUnit(curTarget)
				}
			})
		},
	}
}

func (hunter *Hunter) registerMultiShotSpell(timer *core.Timer) {
	if int(hunter.Level) < multiShotLevel {
		return
	}
	hunter.MultiShot = hunter.GetOrRegisterSpell(hunter.getMultiShotConfig(timer))
}
