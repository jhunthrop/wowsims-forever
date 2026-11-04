package warrior

import (
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// spearingStrikeRageCost is Spearing Strike's own cost, read from the
// client's SpellPower row for spell 1310222 (150 tenths, data lane's
// raw/SpellPower.csv), not from a rank description: "A brutal attack
// that deals 75% weapon damage" names no cost of its own.
const spearingStrikeRageCost = 15.0

// spearingStrikeWeaponDamagePercent is Spearing Strike's base weapon
// damage, 1310222 e1 (effect 31, weapon percent damage) base points 75
// since 1.60.1.70755 (40 before). spearingStrikeBonusMultiplier is e2's
// base points (1 since 70755, 2 before): the tooltip's "additional
// ${$s2*$s3}% weapon damage against Giants, Dragonkin, and mounted
// targets", so 75% more, 150% in all, against one of those.
// TestSpearingStrikeMatchesTheClient pins both to the client row.
const (
	spearingStrikeWeaponDamagePercent = 0.75
	spearingStrikeBonusMultiplier     = 1.0
)

// spearingStrikeBonusWeaponDamagePercent is the additional weapon damage
// against a Giant, Dragonkin or mounted target.
const spearingStrikeBonusWeaponDamagePercent = spearingStrikeWeaponDamagePercent * spearingStrikeBonusMultiplier

// spearingStrikeDamagePercent is the pure selection Spearing Strike's
// ApplyEffects reads: no RNG, no sim state, just the target's MobType.
// Kept separate from ApplyEffects so the 75%/150% split is checkable
// without rolling weapon damage.
//
// "Mounted" is not modelled: nothing in this engine's target.MobType or
// Unit carries a mount state - there is no mounted-player or -mob
// encounter type - so only the Giant and Dragonkin clause of the bonus
// applies here.
func spearingStrikeDamagePercent(target *core.Unit) float64 {
	if target.MobType == proto.MobType_MobTypeGiant || target.MobType == proto.MobType_MobTypeDragonkin {
		return spearingStrikeWeaponDamagePercent + spearingStrikeBonusWeaponDamagePercent
	}
	return spearingStrikeWeaponDamagePercent
}

// registerSpearingStrikeSpell is Spearing Strike (Arms node 105949, 1
// rank, spell 1310222): a new attack, gated on the talent exactly as the
// mage's Ice Lance is gated on its own talent bool, so an Arms warrior
// that did not spend the point cannot cast it and its .results golden
// cannot move for a spell it never registers.
func (warrior *Warrior) registerSpearingStrikeSpell() {
	if !warrior.Talents.SpearingStrike {
		return
	}

	actionID := core.ActionID{SpellID: TalentSpellIDs["spearing_strike"][0]}

	// SpellShapeshift 58595: ShapeshiftMask 65536, Battle Stance only.
	warrior.SpearingStrike = warrior.RegisterSpell(BattleStance, core.SpellConfig{
		SpellCode:      SpellCode_WarriorSpearingStrike,
		ClassSpellMask: WarriorSpellMaskSpearingStrike,
		ActionID:       actionID,
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagAPL | SpellFlagOffensive,

		RequiredLevel: SpearingStrikeLevel[0],

		RageCost: core.RageCostOptions{
			Cost:   spearingStrikeRageCost,
			Refund: 0.8,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			// 20000ms: SpearingStrikeCooldownMS[0] (constants_auto_gen.go).
			// This cooldown was missing entirely.
			CD: core.Cooldown{
				Timer:    warrior.NewTimer(),
				Duration: time.Duration(SpearingStrikeCooldownMS[0]) * time.Millisecond,
			},
		},

		CritDamageBonus: warrior.impale(),

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := spearingStrikeDamagePercent(target) * spell.Unit.MHNormalizedWeaponDamage(sim, spell.MeleeAttackPower(target))
			result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)

			if !result.Landed() {
				spell.IssueRefund(sim)
			}
		},
	})
}
