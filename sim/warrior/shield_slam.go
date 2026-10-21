package warrior

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Shield Slam: "causing 421 to 439 damage, increased by your Block Value
// ... Causes a very high amount of threat" (Protection node 105959, rank 1
// text). The client's row has ap_coefficient 0, so there is no attack-power
// term, and "increased by your Block Value" is one times it (the earlier
// body added twice the block value and 15% of attack power, both
// SoD-shaped).
//
// shieldSlamFlatThreat is UNCONFIRMED: "very high" is a text, not a
// number, and the client's table carries no threat effect for the spell
// (Sunder Armor's is the only warrior effect 63). 508 is the figure the
// fork already used (254 doubled by the 2x flat-threat convention of the
// other warrior specials); the real value is whatever a Forever combat log
// shows. Defensive Stance and Defiance multiply it as they do every threat.
const shieldSlamFlatThreat = 508.0

func (warrior *Warrior) registerShieldSlamSpell() {
	if !warrior.Talents.ShieldSlam {
		return
	}

	rank := rankAtLevel(ShieldSlamLevel[:], warrior.Level)
	spellID := ShieldSlamSpellId[rank]
	// The client's spell 23925 carries one school-damage amount, 655,
	// where vanilla's rank 4 rolled 342-358. The generated table is the
	// authority, so the roll is gone rather than re-centred: a single
	// amount is what the client states and it carries no die width.
	damage := ShieldSlamDamage[rank]
	casterLevel := int(warrior.Level)
	castConfig := core.CastConfig{
		DefaultCast: core.Cast{
			GCD: core.GCDDefault,
		},
		IgnoreHaste: true,
	}
	// Same guard as Slam (slam.go): rank 0 of the generated table (the
	// entry a character below ShieldSlamLevel[1]=40 would resolve to, if
	// the talent were somehow already spent) carries a zero cooldown, and
	// a Cooldown with a Timer but no Duration panics in RegisterSpell.
	if cooldownMS := ShieldSlamCooldownMS[rank]; cooldownMS > 0 {
		castConfig.CD = core.Cooldown{
			Timer:    warrior.NewTimer(),
			Duration: time.Duration(cooldownMS) * time.Millisecond,
		}
	}

	warrior.ShieldSlam = warrior.RegisterSpell(AnyStance, core.SpellConfig{
		SpellCode:      SpellCode_WarriorShieldSlam,
		ClassSpellMask: WarriorSpellMaskShieldSlam,
		ActionID:       core.ActionID{SpellID: spellID},
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskMeleeMHSpecial, // TODO really?
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagAPL | SpellFlagOffensive,

		RequiredLevel: ShieldSlamLevel[rank],
		Rank:          rank,

		RageCost: core.RageCostOptions{
			Cost:   rageCost(ShieldSlamManaCost[rank]),
			Refund: 0.8,
		},
		Cast: castConfig,
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return warrior.PseudoStats.CanBlock
		},

		CritDamageBonus: warrior.impale(),

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		FlatThreatBonus:  shieldSlamFlatThreat,
		BonusCoefficient: 1,
		ClientBaseDamage: damage.Range(casterLevel),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			total := damage.Roll(sim, casterLevel) + warrior.BlockValue()
			result := spell.CalcAndDealDamage(sim, target, total, spell.OutcomeMeleeSpecialHitAndCrit)

			if !result.Landed() {
				spell.IssueRefund(sim)
			}
		},
	})
}
