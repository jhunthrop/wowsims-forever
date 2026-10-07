package warrior

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

func (warrior *Warrior) registerThunderClapSpell() {
	rank := rankAtLevel(ThunderClapLevel[:], warrior.Level)
	// Every rank of Thunder Clap has both a legacy id and a Forever
	// reissue sharing the same rank label and spell_level; the generator
	// keeps the reissue (ThunderClapSpellId) on the higher-id tiebreak.
	// This file used to hardcode the level-58 legacy id (11581) for
	// EVERY rank, which (a) mislabeled every rank below 6 and (b)
	// compared this spell's numbers against the legacy row's stale
	// 6000ms cooldown and -20% slow instead of the reissue's live 4000ms
	// / -10% that ThunderClapCooldownMS and attackSpeedReduction below
	// already use - a convention-vs-literal bug, not a talent
	// double-count.
	spellID := ThunderClapSpellId[rank]
	baseDamage := ThunderClapBaseDamage[rank][0]
	has5pcConq := warrior.HasSetBonus(ItemSetConquerorsBattleGear, 5)
	attackSpeedReduction := core.TernaryInt32(has5pcConq, 15, 10)
	stanceMask := BattleStance

	warrior.ThunderClapAuras = warrior.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
		return core.ThunderClapAura(target, spellID, attackSpeedReduction)
	})

	// Pool-sized ceiling, live-bounded loop; see registerWhirlwindSpell.
	results := make([]*core.SpellResult, min(4, len(warrior.Env.Encounter.AllTargetUnits)))

	castConfig := core.CastConfig{
		DefaultCast: core.Cast{
			GCD: core.GCDDefault,
		},
		IgnoreHaste: true,
	}
	// Rank 0 (level 1, ThunderClapLevel[0]=1 but ThunderClapCooldownMS[0]
	// is still 0) carries a zero cooldown; guard as slam.go does, since
	// Thunder Clap is unconditionally registered (no talent gate) and so
	// is the one spell in this file every level-1 warrior actually hits.
	if cooldownMS := ThunderClapCooldownMS[rank]; cooldownMS > 0 {
		castConfig.CD = core.Cooldown{
			Timer:    warrior.NewTimer(),
			Duration: time.Duration(cooldownMS) * time.Millisecond,
		}
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

		DamageMultiplier: core.TernaryFloat64(has5pcConq, 1.5, 1),
		ThreatMultiplier: 2.5,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			numHits := min(len(results), len(sim.Encounter.TargetUnits))
			for idx := 0; idx < numHits; idx++ {
				results[idx] = spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)
				target = sim.Environment.NextTargetUnit(target)
			}

			for _, result := range results[:numHits] {
				spell.DealDamage(sim, result)
				if result.Landed() {
					warrior.ThunderClapAuras.Get(result.Target).Activate(sim)
				}
			}
		},

		RelatedAuras: []core.AuraArray{warrior.ThunderClapAuras},
	})
}
