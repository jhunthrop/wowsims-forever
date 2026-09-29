package hunter

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// mongooseBiteLearnLevels are Mongoose Bite's four rank learn levels;
// source: 1.60.1.70009 client spell data ("Mongoose Bite", ranks 1-4).
var mongooseBiteLearnLevels = []int{16, 30, 44, 58}

// mongooseBiteBaseDamage is ranks 1-4's flat bonus damage; source:
// 1.60.1.70009 client spell data (spells 1495/14269-14271). Forever's
// numbers are well below vanilla Classic's here too (rank 4 is +57, not
// vanilla's +115); corrected against the client.
var mongooseBiteBaseDamage = [5]float64{0, 15, 22, 37, 57}

func (hunter *Hunter) getMongooseBiteConfig(rank int) core.SpellConfig {
	spellId := [5]int32{0, 1495, 14269, 14270, 14271}[rank]
	baseDamage := mongooseBiteBaseDamage[rank]
	manaCost := [5]float64{0, 30, 40, 50, 65}[rank]
	level := [5]int{0, 16, 30, 44, 58}[rank]

	spellConfig := core.SpellConfig{
		SpellCode:     SpellCode_HunterMongooseBite,
		ActionID:      core.ActionID{SpellID: spellId},
		SpellSchool:   core.SpellSchoolPhysical,
		DefenseType:   core.DefenseTypeMelee,
		ProcMask:      core.ProcMaskMeleeSpecial,
		Flags:         core.SpellFlagMeleeMetrics | core.SpellFlagAPL,
		Rank:          rank,
		RequiredLevel: level,

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			CD: core.Cooldown{
				Timer:    hunter.NewTimer(),
				Duration: time.Second * 5,
			},
		},

		// FOREVER: vanilla Classic's Mongoose Bite could only be cast
		// after the hunter dodged an attack ("Defensive State"). The
		// client's Forever tooltip (wowhead.com/forever/spell=1495) no
		// longer lists that requirement -- only "Requires main hand
		// weapon" and "Cannot be used while shapeshifted" -- so the
		// dodge-gating aura this file used to require is removed; range
		// and the normal cooldown/GCD/mana are the only gates left.
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return hunter.DistanceFromTarget <= core.MaxMeleeAttackDistance
		},

		BonusCritRating:  float64(hunter.Talents.SavageStrikes) * 10 * core.CritRatingPerCritChance,
		CritDamageBonus:  hunter.mortalShots() + hunter.predatorsEdgeCritDamage(),
		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			// Effect code 121 ("Normalized Weapon Damage", wowhead's
			// Forever tooltip literally names it that) is this build's
			// same normalized-weapon-speed effect Aimed Shot's own
			// ApplyEffects reads through CalculateNormalizedWeaponDamage
			// -- Mongoose Bite is an instant special that used to deal
			// only its flat rank amount with no weapon scaling at all,
			// which undersells it relative to the client's own tooltip.
			damage := baseDamage + hunter.MHNormalizedWeaponDamage(sim, spell.MeleeAttackPower(target))
			result := spell.CalcAndDealDamage(sim, target, damage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)
			hunter.tryProcLaceratingStrikes(sim, target, result)
		},
	}

	return spellConfig
}

func (hunter *Hunter) registerMongooseBiteSpell() {
	rank := core.HighestRankAtLevel(mongooseBiteLearnLevels, hunter.Level)
	if rank == 0 {
		return
	}

	config := hunter.getMongooseBiteConfig(rank)
	hunter.MongooseBite = hunter.GetOrRegisterSpell(config)
	hunter.registerLaceratingStrikesDot()
}
