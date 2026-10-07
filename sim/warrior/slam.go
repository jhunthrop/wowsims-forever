package warrior

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// slamRankLevel, slamRankSpellID, slamRankCastTimeMS, slamRankCooldownMS
// and slamRankManaCost correct the generator's dedup for ranks 1-4: each
// rank's winning row (SlamSpellId/Level/CastTime/CooldownMS/ManaCost
// above, 462893-462897) is a same-rank, same-ish-spell_level duplicate
// with NO cost, cast time or cooldown at all - a stub, not the real
// castable ability - that out-ranked the real id on the generator's
// spell_level tiebreak (the stub's spell_level reads one tier high).
// Rank 5 needs no correction: its own duplicate pair (11605/1310200)
// both carry the real numbers. The real per-rank ids and their shared
// 1500ms cast time / 18000ms cooldown / 150-tenths cost are confirmed
// against both data/builds/1.60.1.70009/spellconst/warrior.json (ids
// 1240193, 1464, 8820, 11604, 11605) and spellranks.json, which never
// lists 462893-462897 at all. 11605 is also the id the UI and the
// preset rotations name for rank 5.
var (
	slamRankLevel      = [SlamRanks + 1]int{0, 20, 30, 38, 46, 54}
	slamRankSpellID    = [SlamRanks + 1]int32{0, 1240193, 1464, 8820, 11604, 11605}
	slamRankCastTimeMS = [SlamRanks + 1]int32{0, 1500, 1500, 1500, 1500, 1500}
	slamRankCooldownMS = [SlamRanks + 1]int32{0, 18000, 18000, 18000, 18000, 18000}
	slamRankManaCost   = [SlamRanks + 1]float64{0, 150, 150, 150, 150, 150}
)

func (warrior *Warrior) registerSlamSpell() {
	rank := rankAtLevel(slamRankLevel[:], warrior.Level)
	requiredLevel := slamRankLevel[rank]
	spellID := slamRankSpellID[rank]
	flatDamageBonus := SlamBaseDamage[rank][0]

	castConfig := core.CastConfig{
		DefaultCast: core.Cast{
			GCD:      core.GCDDefault,
			CastTime: time.Millisecond*time.Duration(slamRankCastTimeMS[rank]) - time.Millisecond*100*time.Duration(warrior.Talents.ImprovedSlam),
		},
		ModifyCast: func(sim *core.Simulation, spell *core.Spell, cast *core.Cast) {
			if spell.CastTime() > 0 {
				warrior.AutoAttacks.StopMeleeUntil(sim, sim.CurrentTime+cast.CastTime, true)
			}
		},
	}
	// The client gives Slam an 18 s category cooldown at every learned
	// rank (slamRankCooldownMS); rank 0 (unlearned) genuinely has none: a
	// Cooldown with a Timer and a zero Duration panics in RegisterSpell
	// ("Cast.CD w/o Duration"), so the CD is only attached when the
	// rank's corrected duration is real.
	if cooldownMS := slamRankCooldownMS[rank]; cooldownMS > 0 {
		castConfig.CD = core.Cooldown{
			Timer:    warrior.NewTimer(),
			Duration: time.Duration(cooldownMS) * time.Millisecond,
		}
	}

	warrior.Slam = warrior.RegisterSpell(AnyStance, core.SpellConfig{
		SpellCode:      SpellCode_WarriorSlam,
		ClassSpellMask: WarriorSpellMaskSlam,
		ActionID:       core.ActionID{SpellID: spellID},
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagAPL | SpellFlagOffensive,

		RequiredLevel: requiredLevel,
		Rank:          rank,

		RageCost: core.RageCostOptions{
			Cost:   rageCost(slamRankManaCost[rank]),
			Refund: 0.8,
		},
		Cast: castConfig,

		CritDamageBonus: warrior.impale(),

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		FlatThreatBonus:  140, // Should this be 54 or the old 140 value from before SoD?
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := flatDamageBonus + spell.Unit.MHWeaponDamage(sim, spell.MeleeAttackPower(target))

			result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)
			if !result.Landed() {
				spell.IssueRefund(sim)
			}
		},
	})
}
