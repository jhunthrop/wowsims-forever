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
// Improved Slam's per-point numbers: the cast time and GCD each lose
// 0.25 s a point (the talent's own rank text, see registerSlamSpell),
// the cooldown loses 3 s a point (Blizzard's 1 October 2026 notes,
// "Improved Slam 3 s off the cooldown per rank"; the client text still
// says 1.5 s and the note is the live state).
const (
	improvedSlamCastReductionPerPoint     = 250 * time.Millisecond
	improvedSlamCooldownReductionPerPoint = 3 * time.Second
)

// improvedSlamReductions is the talent's effect at a given point count,
// as the three durations it takes off Slam's cast time, global cooldown
// and cooldown; kept as a pure function so the per-rank numbers are
// checkable without building a warrior.
func improvedSlamReductions(points int32) (cast, gcd, cooldown time.Duration) {
	p := time.Duration(points)
	return improvedSlamCastReductionPerPoint * p, improvedSlamCastReductionPerPoint * p, improvedSlamCooldownReductionPerPoint * p
}

// improvedSlamKeepsTheSwing is the talent's third clause: with any point
// spent, Slam no longer stops the swing timer for its cast.
func improvedSlamKeepsTheSwing(points int32) bool {
	return points > 0
}

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

	// Improved Slam, per the client's own rank text (build 1.60.1.70009,
	// Arms tree, spell 12862): "Reduces the global cooldown and cast time
	// of your Slam ability by 0.25/0.5 sec. In addition, Slam no longer
	// interrupts or delays your melee swing and Slam's cooldown is
	// reduced by 1.5/3 sec." (3/6 s live, see
	// improvedSlamCooldownReductionPerPoint.) Before 2026-10-07 this file took 0.1 s a
	// point off the cast time only, kept the full GCD and cooldown, and
	// stopped the swing timer for every Slam regardless of the talent.
	castReduction, gcdReduction, cooldownReduction := improvedSlamReductions(warrior.Talents.ImprovedSlam)
	keepsTheSwing := improvedSlamKeepsTheSwing(warrior.Talents.ImprovedSlam)
	castConfig := core.CastConfig{
		DefaultCast: core.Cast{
			GCD:      core.GCDDefault - gcdReduction,
			CastTime: time.Millisecond*time.Duration(slamRankCastTimeMS[rank]) - castReduction,
		},
		ModifyCast: func(sim *core.Simulation, spell *core.Spell, cast *core.Cast) {
			// Untalented Slam still resets the swing: the cast holds
			// the weapon until it lands. With any point in Improved
			// Slam the swing timer runs on underneath the cast.
			if spell.CastTime() > 0 && !keepsTheSwing {
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
			Duration: time.Duration(cooldownMS)*time.Millisecond - cooldownReduction,
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
