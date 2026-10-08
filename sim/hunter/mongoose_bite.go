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

func (hunter *Hunter) getMongooseBiteConfig(rank int) core.SpellConfig {
	spellId := [5]int32{0, 1495, 14269, 14270, 14271}[rank]
	damage := MongooseBiteDamage[rank]
	casterLevel := int(hunter.Level)
	manaCost := [5]float64{0, 30, 40, 50, 65}[rank]
	level := [5]int{0, 16, 30, 44, 58}[rank]

	spellConfig := core.SpellConfig{
		SpellCode:     SpellCode_HunterMongooseBite,
		ActionID:      core.ActionID{SpellID: spellId},
		SpellSchool:   core.SpellSchoolPhysical,
		DefenseType:   core.DefenseTypeMelee,
		ProcMask:      core.ProcMaskMeleeMHSpecial,
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

		// Every rank's client text is "Can only be performed after you
		// dodge" (spells 1495/14269-14271): the cast is gated on the
		// window a dodge or Expose Prey opens (MongooseBiteWindowAura).
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return hunter.DistanceFromTarget <= core.MaxMeleeAttackDistance && hunter.MongooseBiteWindowAura.IsActive()
		},
		RelatedSelfBuff: hunter.MongooseBiteWindowAura,

		BonusCritRating:  float64(hunter.Talents.SavageStrikes) * 10 * core.CritRatingPerCritChance,
		CritDamageBonus:  hunter.mortalShots() + hunter.predatorsEdgeCritDamage(),
		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		ClientBaseDamage: damage.Range(casterLevel),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			// The window's single charge is spent before the hit lands,
			// so an Expose Prey proc off this very hit opens a fresh one.
			hunter.MongooseBiteWindowAura.Deactivate(sim)
			// Effect code 121 ("Normalized Weapon Damage", wowhead's
			// Forever tooltip literally names it that) is this build's
			// same normalized-weapon-speed effect Aimed Shot's own
			// ApplyEffects reads through CalculateNormalizedWeaponDamage
			// -- Mongoose Bite is an instant special that used to deal
			// only its flat rank amount with no weapon scaling at all,
			// which undersells it relative to the client's own tooltip.
			total := damage.Roll(sim, casterLevel) + hunter.MHNormalizedWeaponDamage(sim, spell.MeleeAttackPower(target))
			result := spell.CalcAndDealDamage(sim, target, total, spell.OutcomeMeleeWeaponSpecialHitAndCrit)
			hunter.tryProcLaceratingStrikes(sim, target, result)
		},
	}

	return spellConfig
}

const (
	// mongooseBiteWindowSpellId is Defensive State (client spell 5302):
	// 5 sec, one charge a melee spell consumes. Expose Prey's own
	// "Mongoose Bite activated" (1310726) carries identical rows, so both
	// sources share this one aura.
	mongooseBiteWindowSpellId int32 = 5302
	mongooseBiteWindowLength        = 5 * time.Second
	// exposePreyWindowLength is Expose Prey's "Mongoose Bite activated"
	// (client spell 1310726): its duration index went from 5 sec to 10 sec
	// in build 1.60.1.70291.
	exposePreyWindowLength = 10 * time.Second
)

// openMongooseBiteWindowFor opens the window for at least length, never
// cutting short a longer one already open.
func (hunter *Hunter) openMongooseBiteWindowFor(sim *core.Simulation, length time.Duration) {
	window := hunter.MongooseBiteWindowAura
	expires := sim.CurrentTime + length
	if window.IsActive() {
		expires = max(expires, window.ExpiresAt())
	}
	window.Activate(sim)
	window.UpdateExpires(sim, expires)
}

// registerMongooseBiteWindow registers the aura the APL gates Mongoose
// Bite on, opened by a dodge (client text) or by Expose Prey.
func (hunter *Hunter) registerMongooseBiteWindow() {
	hunter.MongooseBiteWindowAura = hunter.RegisterAura(core.Aura{
		Label:    "Mongoose Bite Ready",
		ActionID: core.ActionID{SpellID: mongooseBiteWindowSpellId},
		Duration: mongooseBiteWindowLength,
	})

	hunter.RegisterAura(core.Aura{
		Label:    "Mongoose Bite Dodge Trigger",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitTaken: func(_ *core.Aura, sim *core.Simulation, _ *core.Spell, result *core.SpellResult) {
			if result.DidDodge() {
				hunter.openMongooseBiteWindowFor(sim, mongooseBiteWindowLength)
			}
		},
	})
}

func (hunter *Hunter) registerMongooseBiteSpell() {
	rank := core.HighestRankAtLevel(mongooseBiteLearnLevels, hunter.Level)
	if rank == 0 {
		return
	}

	hunter.registerMongooseBiteWindow()

	config := hunter.getMongooseBiteConfig(rank)
	hunter.MongooseBite = hunter.GetOrRegisterSpell(config)
	hunter.registerLaceratingStrikesDot()
}
