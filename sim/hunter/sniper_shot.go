package hunter

import (
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// Sniper Shot (talents/hunter.json node 104997, Marksmanship, bool,
// spell 1310687): "A steady snipe that increases ranged damage by 160."
// The talent grants the spell itself - it has no existing cast in this
// package to modify - the same shape as mage's Ice Lance
// (registerIceLanceSpell), so it is registered gated on the talent
// rather than unconditionally.
//
// spellconst/hunter.json's own body for 1310687: cast_time_ms 4000,
// gcd_ms 1500, cooldown_ms 15000, cost 365 (mana), one effect (index 0,
// effect code 121 - "Normalized Weapon Damage", the same effect code
// Mongoose Bite's ApplyEffects comment names and Aimed Shot's own
// ApplyEffects reads through CalculateNormalizedWeaponDamage - amount
// 160, sp_coefficient 1.0, ap_coefficient 0.0). ap_coefficient 0 does
// not mean "no Ranged Attack Power scaling": like Aimed Shot, the RAP
// scaling lives in the normalized-weapon-damage effect itself, not in a
// separate coefficient column, so this spell's ApplyEffects is Aimed
// Shot's shape with Aimed Shot's own per-rank baseDamage array replaced
// by the rank's flat amount (160/225/295 at levels 40/48/58, read from
// SniperShotDamage; the cost, cast time and cooldown are the same at all
// three ranks, so only the id and the damage follow the rank).
const sniperShotManaCost = 365.0
const sniperShotCastTime = time.Millisecond * 4000
const sniperShotCooldown = time.Second * 15

func (hunter *Hunter) getSniperShotConfig(rank int) core.SpellConfig {
	flatDamage := SniperShotDamage[rank]
	casterLevel := int(hunter.Level)
	return core.SpellConfig{
		SpellCode:     SpellCode_HunterSniperShot,
		ActionID:      core.ActionID{SpellID: SniperShotSpellId[rank]},
		SpellSchool:   core.SpellSchoolPhysical,
		DefenseType:   core.DefenseTypeRanged,
		ProcMask:      core.ProcMaskRangedSpecial,
		Flags:         core.SpellFlagMeleeMetrics | core.SpellFlagAPL | SpellFlagShot,
		CastType:      proto.CastType_CastTypeRanged,
		Rank:          rank,
		RequiredLevel: SniperShotLevel[rank],
		MissileSpeed:  24,

		ManaCost: core.ManaCostOptions{
			FlatCost: sniperShotManaCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD:      core.GCDDefault,
				CastTime: sniperShotCastTime,
			},
			CD: core.Cooldown{
				Timer:    hunter.NewTimer(),
				Duration: sniperShotCooldown,
			},
			ModifyCast: func(sim *core.Simulation, spell *core.Spell, cast *core.Cast) {
				cast.CastTime = spell.CastTime()
				hunter.Unit.AutoAttacks.CancelAutoSwing(sim)
			},
			IgnoreHaste: true, // Hunter GCD is locked at 1.5s
			CastTime: func(spell *core.Spell) time.Duration {
				return time.Duration(float64(spell.DefaultCast.CastTime) / hunter.RangedSwingSpeed())
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return hunter.DistanceFromTarget >= core.MinRangedAttackDistance
		},

		CritDamageBonus: hunter.mortalShots(),

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: 1,
		ClientBaseDamage: flatDamage.Range(casterLevel),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := hunter.AutoAttacks.Ranged().CalculateNormalizedWeaponDamage(sim, spell.RangedAttackPower(target, false)) +
				hunter.AmmoDamageBonus +
				flatDamage.Roll(sim, casterLevel)

			result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeRangedHitAndCrit)
			hunter.Unit.AutoAttacks.EnableAutoSwing(sim)
			spell.WaitTravelTime(sim, func(s *core.Simulation) {
				spell.DealDamage(sim, result)
			})
		},
	}
}

func (hunter *Hunter) registerSniperShotSpell() {
	if !hunter.Talents.SniperShot {
		return
	}
	rank := core.HighestRankAtLevel(SniperShotLevel[1:], hunter.Level)
	if rank == 0 {
		return
	}

	hunter.SniperShot = hunter.GetOrRegisterSpell(hunter.getSniperShotConfig(rank))
}
