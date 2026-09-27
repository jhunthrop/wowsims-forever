package warrior

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

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
	baseDamage := ShieldSlamBaseDamage[rank][0]
	// No known equation for either, and the client's table carries
	// neither a threat column nor an attack-power coefficient for this
	// spell (its ap_coefficient is 0), so both stay typed.
	threat := 254.0
	apCoef := 0.15

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
		FlatThreatBonus:  threat * 2,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			damage := baseDamage + warrior.BlockValue()*2 + apCoef*spell.MeleeAttackPower(target)
			result := spell.CalcAndDealDamage(sim, target, damage, spell.OutcomeMeleeSpecialHitAndCrit)

			if !result.Landed() {
				spell.IssueRefund(sim)
			}
		},
	})
}
