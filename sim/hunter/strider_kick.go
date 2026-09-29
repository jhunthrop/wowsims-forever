package hunter

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// striderKickLevel is Strider Kick's single learn level; source:
// 1.60.1.70009 client spell data ("Strider Kick", spell 1317257).
const striderKickLevel = 30

// Strider Kick is a 1-point, single-rank Survival talent (max_rank 1):
// "A powerful kick that deals 100% melee weapon damage and increases
// movement speed by 30% for 3 sec." The client's effect 0 is a code-121
// "Normalized Weapon Damage" baseline of 0 (see mongoose_bite.go and
// counterattack.go's comments on that effect code, cross-checked against
// aimed_shot.go's own CalculateNormalizedWeaponDamage use), and effect 1
// is a code-31 weapon-damage-percent coefficient of 100 on that same
// baseline -- 0 + 100% of normalized weapon damage, matching "100%
// melee weapon damage" exactly. The 3s speed buff (effect index 2) isn't
// modeled -- this engine's sim doesn't move a static-distance target, so
// a self-speed buff has no mechanical effect here, the same
// simplification Wing Clip's snare and Counterattack's root already
// make.
func (hunter *Hunter) registerStriderKickSpell() {
	if !hunter.Talents.StriderKick {
		return
	}
	if hunter.Level < striderKickLevel {
		return
	}

	hunter.StriderKick = hunter.RegisterSpell(core.SpellConfig{
		SpellCode:     SpellCode_HunterStriderKick,
		ActionID:      core.ActionID{SpellID: 1317257},
		SpellSchool:   core.SpellSchoolPhysical,
		DefenseType:   core.DefenseTypeMelee,
		ProcMask:      core.ProcMaskMeleeMHSpecial,
		Flags:         core.SpellFlagMeleeMetrics | core.SpellFlagAPL,
		Rank:          1, // client spellconst rank for 1317257 is 1, not 0 (a genuinely unranked ability, like Bloodrage/Whirlwind, reads 0).
		RequiredLevel: striderKickLevel,

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			CD: core.Cooldown{
				Timer:    hunter.NewTimer(),
				Duration: time.Second * 8,
			},
		},

		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return hunter.DistanceFromTarget <= core.MaxMeleeAttackDistance
		},

		CritDamageBonus:  hunter.mortalShots() + hunter.predatorsEdgeCritDamage(),
		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			damage := hunter.MHNormalizedWeaponDamage(sim, spell.MeleeAttackPower(target))
			spell.CalcAndDealDamage(sim, target, damage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)
		},
	})
}
