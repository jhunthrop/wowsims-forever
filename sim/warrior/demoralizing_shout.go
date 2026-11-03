package warrior

import (
	"strconv"
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Demoralizing Shout (client spells 1160, 6190, 11554, 11555, 11556):
// "-49/-77/-98/-147/-196" attack power (aura 99) at the rank's own level,
// a further 1.4 per caster level up to the rank's max level
// (DemoralizingShoutPointsPerLevel, DemoralizingShoutMaxLevel), for 45000
// ms, at 100 (10 rage). The vanilla 45/56/76/111/146 and 30 seconds that
// core.DemoralizingShoutAura still carries for the raid debuff are gone
// from the warrior's own cast.
//
// The generated tables describe the free 27579 variant at rank 5 (cost
// 0, 30 s, 1.0 a level), so the cost, the amounts and the 1.4 a level are
// typed here; TestDemoralizingShoutReductionMatchesClient checks them
// against the client rows.
const (
	demoralizingShoutDuration            = 45 * time.Second
	demoralizingShoutRageCost            = 10.0
	demoralizingShoutAttackPowerPerLevel = 1.4
)

var demoralizingShoutBaseAttackPower = [DemoralizingShoutRanks + 1]float64{0, 49, 77, 98, 147, 196}

// demoralizingShoutAttackPowerReduction is the attack power a rank takes
// off the target at the caster's level.
func demoralizingShoutAttackPowerReduction(rank int, level int32) float64 {
	levelsAbove := max(0, min(int(level), DemoralizingShoutMaxLevel[rank])-DemoralizingShoutLevel[rank])
	return demoralizingShoutBaseAttackPower[rank] + demoralizingShoutAttackPowerPerLevel*float64(levelsAbove)
}

// demoralizingShoutBaseThreat and demoralizingShoutThreatPerLevel are each
// rank's effect 63 (flat threat on every target hit): 11, 19, 27, 35 and 43
// at the rank's learn level, plus 0.8 a level to its cap (SpellEffect
// 1361983 to 1361987).
var demoralizingShoutBaseThreat = [DemoralizingShoutRanks + 1]float64{0, 11, 19, 27, 35, 43}

const demoralizingShoutThreatPerLevel = 0.8

func demoralizingShoutThreat(rank int, level int32) float64 {
	levelsAbove := max(0, min(int(level), DemoralizingShoutMaxLevel[rank])-DemoralizingShoutLevel[rank])
	return demoralizingShoutBaseThreat[rank] + demoralizingShoutThreatPerLevel*float64(levelsAbove)
}

func (warrior *Warrior) registerDemoralizingShoutSpell() {
	rank := max(1, rankAtLevel(core.DemoralizingShoutLevel[:], warrior.Level))
	actionID := core.ActionID{SpellID: core.DemoralizingShoutSpellId[rank]}
	apReduction := demoralizingShoutAttackPowerReduction(rank, warrior.Level)

	warrior.DemoralizingShoutAuras = warrior.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
		return core.DemoralizingShoutAuraFor(target, "DemoralizingShout-"+strconv.Itoa(int(apReduction)), actionID.SpellID, apReduction, demoralizingShoutDuration)
	})

	warrior.DemoralizingShout = warrior.RegisterSpell(AnyStance, core.SpellConfig{
		ActionID:       actionID,
		ClassSpellMask: WarriorSpellMaskDemoralizingShout,
		SpellSchool:    core.SpellSchoolPhysical,
		ProcMask:       core.ProcMaskEmpty,
		Flags:          core.SpellFlagAPL | SpellFlagOffensive,

		RequiredLevel: core.DemoralizingShoutLevel[rank],
		Rank:          rank,

		RageCost: core.RageCostOptions{
			Cost: demoralizingShoutRageCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			IgnoreHaste: true,
		},

		// The shout deals no damage, so ThreatMultiplier touches nothing;
		// its threat is effect 63's flat amount on each target hit
		// (demoralizingShoutThreat).
		ThreatMultiplier: 0.4,
		FlatThreatBonus:  demoralizingShoutThreat(rank, warrior.Level),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			for _, aoeTarget := range sim.Encounter.TargetUnits {
				result := spell.CalcAndDealOutcome(sim, aoeTarget, spell.OutcomeMagicHit)
				if result.Landed() {
					warrior.DemoralizingShoutAuras.Get(aoeTarget).Activate(sim)
				}
			}
		},

		RelatedAuras:    []core.AuraArray{warrior.DemoralizingShoutAuras},
		RelatedSelfBuff: warrior.DemoralizingShoutAuras.Get(warrior.CurrentTarget),
	})
}
