package warrior

import (
	"strconv"
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Thunder Clap is the CLASSIC-id family the trainer teaches: the client's
// SkillLineAbility rows carry 6343, 8198, 8204, 8205, 11580 and 11581 and
// none of the 4618xx ids the generator's higher-id tie-break keeps (the
// generated ThunderClapSpellId, which sim/conformance and the damage
// tests still read). Those 4618xx rows are a Season of Discovery
// reissue - 4 s cooldown, 10% slow - while the learnable family has the
// 6 s cooldown and the 20% slow the Forever Deep Dive describes ("rank 4
// slows attacks 20% for 22 s", research/06-since-announcement.md); the
// damage and the 200 rage cost are identical in both families, so
// ThunderClapDamage and ThunderClapManaCost are read for either.
var thunderClapRankSpellID = [ThunderClapRanks + 1]int32{0, 6343, 8198, 8204, 8205, 11580, 11581}

// thunderClapRankDuration is the client's duration_ms for each rank of the
// learnable family (10, 14, 18, 22, 26 and 30 seconds).
var thunderClapRankDuration = [ThunderClapRanks + 1]time.Duration{
	0, 10 * time.Second, 14 * time.Second, 18 * time.Second, 22 * time.Second, 26 * time.Second, 30 * time.Second,
}

const (
	// thunderClapCooldown is the learnable family's category cooldown
	// (6000 ms); its slow is core.ThunderClapSlowPercent.
	thunderClapCooldown               = 6 * time.Second
	conquerorsBattlegear5pcSlowDamage = 1.5

	// thunderClapAttackPowerCoefficient is the attack power each hit adds
	// to the rank's amount. Blizzard's 30 September 2026 deep dive:
	// "Thunder Clap: Now gains significant damage scaling from Attack
	// Power ... Threat rebalanced (reduced) to account for being usable
	// in Defensive Stance." The client rows carry no coefficient (a
	// server value), so this is measured: eleven Protection warriors at
	// 15 to 33 in the public Forever logs of 5 to 12 November
	// (foreverlogs.gg), each one's Thunder Clap hits read against its
	// own Revenge hits (whose 0.25 x attack power the client's
	// description variable 1024 states) to take the target's armor out,
	// give 0.05 to 0.07 of attack power with a median of 0.0625
	// (design/reviews/2026-11-12-sim-accuracy/protection-warrior.md).
	// The threat multiplier below is vanilla's 2.5 and UNCONFIRMED for
	// Forever: Blizzard says it was reduced and names no number.
	thunderClapAttackPowerCoefficient = 0.0625
)

// thunderClapBaseDamage is one Thunder Clap hit before mitigation: the
// rank's roll plus the attack power share.
func thunderClapBaseDamage(roll, attackPower float64) float64 {
	return roll + thunderClapAttackPowerCoefficient*attackPower
}

func (warrior *Warrior) registerThunderClapSpell() {
	rank := max(1, rankAtLevel(ThunderClapLevel[:], warrior.Level))
	spellID := thunderClapRankSpellID[rank]
	damage := ThunderClapDamage[rank]
	casterLevel := int(warrior.Level)
	// Conqueror's Battlegear 5-piece: "Increase the Slow effect and damage
	// of Thunder Clap by 50%".
	has5pcConq := warrior.HasSetBonus(ItemSetConquerorsBattleGear, 5)
	attackSpeedReduction := core.ThunderClapSlowPercent
	if has5pcConq {
		attackSpeedReduction = core.ThunderClapConquerorSlowPercent
	}
	// Forever lets a Defensive Stance warrior clap ("Thunder Clap works in
	// Battle or Defensive Stance", research/06-since-announcement.md,
	// single-source).
	stanceMask := BattleStance | DefensiveStance

	warrior.ThunderClapAuras = warrior.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
		return thunderClapAura(target, spellID, attackSpeedReduction, thunderClapRankDuration[rank])
	})

	// Pool-sized ceiling, live-bounded loop; see registerWhirlwindSpell.
	results := make([]*core.SpellResult, min(4, len(warrior.Env.Encounter.AllTargetUnits)))

	castConfig := core.CastConfig{
		DefaultCast: core.Cast{
			GCD: core.GCDDefault,
		},
		IgnoreHaste: true,
		CD: core.Cooldown{
			Timer:    warrior.NewTimer(),
			Duration: thunderClapCooldown,
		},
	}

	warrior.ThunderClap = warrior.RegisterSpell(stanceMask, core.SpellConfig{
		ActionID:       core.ActionID{SpellID: spellID},
		ClassSpellMask: WarriorSpellMaskThunderClap,
		RequiredLevel:  ThunderClapLevel[rank],
		Rank:           rank,
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMagic,
		ProcMask:       core.ProcMaskSpellDamage,
		Flags:          core.SpellFlagAPL | SpellFlagOffensive,

		RageCost: core.RageCostOptions{
			// Improved Thunder Clap's discount is a SpellMod in
			// talents.go, so this is the client's undiscounted cost.
			Cost: rageCost(ThunderClapManaCost[rank]),
		},
		Cast: castConfig,

		CritDamageBonus: warrior.impale(),

		DamageMultiplier: core.TernaryFloat64(has5pcConq, conquerorsBattlegear5pcSlowDamage, 1),
		ThreatMultiplier: 2.5,
		ClientBaseDamage: damage.Range(casterLevel),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			numHits := min(len(results), len(sim.Encounter.TargetUnits))
			for idx := 0; idx < numHits; idx++ {
				results[idx] = spell.CalcDamage(sim, target, thunderClapBaseDamage(damage.Roll(sim, casterLevel), spell.MeleeAttackPower(target)), spell.OutcomeMagicHitAndCrit)
				target = sim.Environment.NextTargetUnit(target)
			}

			for _, result := range results[:numHits] {
				spell.DealDamage(sim, result)
				if result.Landed() {
					warrior.ThunderClapAuras.Get(result.Target).Activate(sim)
				}
			}
		},

		RelatedAuras:    []core.AuraArray{warrior.ThunderClapAuras},
		RelatedSelfBuff: warrior.ThunderClapAuras.Get(warrior.CurrentTarget),
	})
}

// thunderClapAura is the slow Thunder Clap puts on a target: the shared
// attack-speed-reduction effect for the rank's own duration. core's
// ThunderClapAura is the raid-debuff one with a fixed 30 s.
func thunderClapAura(target *core.Unit, spellID int32, slowPercent int32, duration time.Duration) *core.Aura {
	aura := target.GetOrRegisterAura(core.Aura{
		Label:    "ThunderClap-" + strconv.Itoa(int(slowPercent)),
		ActionID: core.ActionID{SpellID: spellID},
		Duration: duration,
	})
	core.AtkSpeedReductionEffect(aura, 1+0.01*float64(slowPercent))
	return aura
}
