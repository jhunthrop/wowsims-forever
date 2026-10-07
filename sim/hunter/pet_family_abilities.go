package hunter

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// The trainable Forever family abilities that are periodic damage: Savage
// Rend (Raptor), Tendon Rip (Hyena) and Web (Spider). Each is read from
// its generated rank table (id, learn level, focus cost, cooldown and the
// per-tick amount, effect 6 aura 3), and the pet casts it on its cooldown
// ahead of the focus dump: a bleed ignores armor, and each of these
// returns more damage per focus than the family's Claw or Bite.
// Dismember (Crocolisk), Pinch (Crab) and Sonic Blast (Bat) are direct
// hits that return less damage per focus than the dump they would
// replace (see design/reviews/2026-10-07-hunter-hydra-lacerate-pets.md),
// so the pet does not cast them and the engine does not register them.

// petPeriodicAbility is one family ability's client table and shape.
type petPeriodicAbility struct {
	name       string
	spellCode  int32
	school     core.SpellSchool
	spellIDs   []int32
	levels     []int
	focusCosts []float64
	cooldownMS []int32
	damage     [][]float64
	ticks      int32
	tickLength time.Duration
}

var (
	savageRendAbility = petPeriodicAbility{
		name:       "Savage Rend",
		spellCode:  SpellCode_HunterPetSavageRend,
		school:     core.SpellSchoolPhysical,
		spellIDs:   SavageRendSpellId[:],
		levels:     SavageRendLevel[:],
		focusCosts: SavageRendManaCost[:],
		cooldownMS: SavageRendCooldownMS[:],
		damage:     SavageRendBaseDamage[:],
		ticks:      6, // 18 s over 3 s
		tickLength: 3 * time.Second,
	}
	tendonRipAbility = petPeriodicAbility{
		name:       "Tendon Rip",
		spellCode:  SpellCode_HunterPetTendonRip,
		school:     core.SpellSchoolPhysical,
		spellIDs:   TendonRipSpellId[:],
		levels:     TendonRipLevel[:],
		focusCosts: TendonRipManaCost[:],
		cooldownMS: TendonRipCooldownMS[:],
		damage:     TendonRipBaseDamage[:],
		ticks:      3, // 9 s over 3 s
		tickLength: 3 * time.Second,
	}
	webAbility = petPeriodicAbility{
		name:       "Web",
		spellCode:  SpellCode_HunterPetWeb,
		school:     core.SpellSchoolNature,
		spellIDs:   WebSpellId[:],
		levels:     WebLevel[:],
		focusCosts: WebManaCost[:],
		cooldownMS: WebCooldownMS[:],
		damage:     WebBaseDamage[:],
		ticks:      4, // 4 s over 1 s
		tickLength: time.Second,
	}
)

func (hp *HunterPet) newPeriodicAbility(ability petPeriodicAbility) *core.Spell {
	rank := core.HighestRankAtLevel(ability.levels[1:], hp.Owner.Level)
	if rank == 0 {
		return nil
	}
	tickDamage := ability.damage[rank][0]
	outcome := func(spell *core.Spell) core.OutcomeApplier { return spell.OutcomeMeleeSpecialHitNoHitCounter }
	defense := core.DefenseTypeMelee
	if ability.school == core.SpellSchoolNature {
		outcome = func(spell *core.Spell) core.OutcomeApplier { return spell.OutcomeMagicHit }
		defense = core.DefenseTypeMagic
	}

	return hp.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: ability.spellIDs[rank]},
		SpellCode:   ability.spellCode,
		SpellSchool: ability.school,
		DefenseType: defense,
		ProcMask:    core.ProcMaskMeleeMHSpecial,
		Flags:       core.SpellFlagMeleeMetrics | core.SpellFlagNoOnCastComplete,

		FocusCost: core.FocusCostOptions{
			Cost: ability.focusCosts[rank],
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: PetGCD,
			},
			CD: core.Cooldown{
				Timer:    hp.NewTimer(),
				Duration: time.Duration(ability.cooldownMS[rank]) * time.Millisecond,
			},
			IgnoreHaste: true,
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		ClientBaseDamage: [2]float64{tickDamage, tickDamage},

		Dot: core.DotConfig{
			Aura:          core.Aura{Label: ability.name},
			NumberOfTicks: ability.ticks,
			TickLength:    ability.tickLength,
			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
				dot.Snapshot(target, tickDamage, isRollover)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcOutcome(sim, target, outcome(spell))
			if result.Landed() {
				spell.Dot(target).Apply(sim)
			}
			spell.DealOutcome(sim, result)
		},
	})
}
