package druid

import (
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

// Primal Bite (talent node 104956; spells 407995, 1238069, 1238070 and
// 1238073): "Bite the target, dealing 100% normal damage plus 42 and
// generating a high amount of threat." Every rank costs 20 rage (cost 200,
// less Ferocity's 1 a rank), is on the global cooldown and has a 6 second
// cooldown of its own. The client states a weapon-damage effect (effect
// 58, the flat amount) and a 100% weapon-percent effect (effect 31), so the
// hit is the paw's swing plus the flat amount, which "normal damage" is
// read as.
//
// Client 1.60.1.70707's hotfix made Primal Bite the Feral capstone (tier 6,
// so learned at 40) and raised its low ranks. The client tables still carry
// the pre-hotfix ranks (PrimalBiteBaseDamage: 26, 38, 59 and 77 at 25, 36,
// 48 and 60); the live tooltips (Wowhead's Forever spell tooltips, read
// 2026-10-31) state 42 at 40 (407995), 62 at 50 (1238069) and 77 at 60
// (1238070 and 1238073 alike), so primalBiteRanks is the live ladder and
// the client table (PrimalBiteFlatDamage) is kept only for the
// conformance comparison.
//
// unconfirmed: the threat multiplier (HighThreatMultiplier).
const (
	primalBiteRageCost = 20.0
	// primalBiteCooldown is the client's category cooldown on every rank.
	primalBiteCooldown   = 6 * time.Second
	primalBiteMissRefund = 0.8
)

// PrimalBiteFlatDamage is the client table's weapon-damage flat per rank.
var PrimalBiteFlatDamage = clientdamage.FromTable(PrimalBiteBaseDamage[:], PrimalBitePointsPerLevel[:], PrimalBiteLevel[:], PrimalBiteMaxLevel[:])

// primalBiteRank is one live rank of Primal Bite.
type primalBiteRank struct {
	rank    int
	spellID int32
	level   int
	flat    float64
}

// primalBiteRanks is the live ladder, lowest first. Rank 3 (1238070) is
// stated at level 60 with the same 77 as rank 4, so it is never the
// highest rank a druid knows and is left out.
var primalBiteRanks = []primalBiteRank{
	{rank: 1, spellID: 407995, level: 40, flat: 42},
	{rank: 2, spellID: 1238069, level: 50, flat: 62},
	{rank: 4, spellID: 1238073, level: 60, flat: 77},
}

// primalBiteRankAt is the highest live rank a druid of the level knows, and
// false below the first.
func primalBiteRankAt(level int32) (primalBiteRank, bool) {
	var known primalBiteRank
	for _, rank := range primalBiteRanks {
		if int32(rank.level) <= level {
			known = rank
		}
	}
	return known, known.spellID != 0
}

func (druid *Druid) registerPrimalBiteSpell() {
	if !druid.Talents.PrimalBite {
		return
	}
	rank, ok := primalBiteRankAt(druid.Level)
	if !ok {
		return
	}

	druid.PrimalBite = druid.RegisterSpell(Bear, core.SpellConfig{
		SpellCode:      SpellCode_DruidPrimalBite,
		ClassSpellMask: DruidSpellMaskPrimalBite,
		ActionID:       core.ActionID{SpellID: rank.spellID},
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          SpellFlagOmen | core.SpellFlagMeleeMetrics | core.SpellFlagAPL,

		Rank:          rank.rank,
		RequiredLevel: rank.level,

		RageCost: core.RageCostOptions{
			Cost:   primalBiteRageCost - ferocityRageDiscount(druid.Talents.Ferocity),
			Refund: primalBiteMissRefund,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			CD: core.Cooldown{
				Timer:    druid.NewTimer(),
				Duration: primalBiteCooldown,
			},
			IgnoreHaste: true,
		},

		DamageMultiplier: 1,
		ThreatMultiplier: HighThreatMultiplier,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := rank.flat + spell.Unit.MHWeaponDamage(sim, spell.MeleeAttackPower(target))
			baseDamage *= druid.RendAndTearMultiplier(target)
			result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)
			if !result.Landed() {
				spell.IssueRefund(sim)
			}
			spell.DealDamage(sim, result)
		},
	})
}
