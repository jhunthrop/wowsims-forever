package hunter

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

type PetAbilityType int

// Pet AI doesn't use abilities immediately, so model this with a 1.6s GCD.
const PetGCD = time.Millisecond * 1600

const (
	Unknown PetAbilityType = iota
	Bite
	Claw
	Screech
	FuriousHowl
	LightningBreath
	ScorpidPoison
	SavageRend
	TendonRip
	Web
)

func (hp *HunterPet) NewPetAbility(abilityType PetAbilityType, isPrimary bool) *core.Spell {
	switch abilityType {
	case Bite:
		return hp.newBite()
	case Claw:
		return hp.newClaw()
	case Screech:
		return hp.newScreech()
	// case FuriousHowl:
	// 	return hp.newFuriousHowl()
	case LightningBreath:
		return hp.newLightningBreath()
	case ScorpidPoison:
		return hp.newScorpidPoison()
	case SavageRend:
		return hp.newPeriodicAbility(savageRendAbility)
	case TendonRip:
		return hp.newPeriodicAbility(tendonRipAbility)
	case Web:
		return hp.newPeriodicAbility(webAbility)
	// case Swipe:
	// 	return hp.newSwipe()
	case Unknown:
		return nil
	default:
		panic("Invalid pet ability type")
	}
}

// petClawLearnLevels are Cat/Pet Claw's eight rank learn levels; source:
// 1.60.1.70009 client spell data ("Claw", ranks 1-8).
var petClawLearnLevels = []int{1, 8, 16, 24, 32, 40, 48, 56}

// petClawSpellID is Claw's rank -> spell id, index 0 unused.
var petClawSpellID = [9]int32{0, 16827, 16828, 16829, 16830, 16831, 16832, 3010, 3009}

// petClawBaseDamageMin/Max are Claw's rank -> damage roll bounds, index 0
// unused. Only ranks 4, 6, 7 and 8 have a tuned value in this file; ranks
// 1-3 and 5 carry the nearest known rank's numbers forward until real ones
// are sourced.
var petClawBaseDamageMin = [9]float64{0, 16, 16, 16, 16, 16, 26, 35, 43}
var petClawBaseDamageMax = [9]float64{0, 22, 22, 22, 22, 22, 36, 49, 59}

func (hp *HunterPet) newClaw() *core.Spell {
	rank := core.HighestRankAtLevel(petClawLearnLevels, hp.Owner.Level)
	if rank == 0 {
		return nil
	}

	baseDamageMin := petClawBaseDamageMin[rank]
	baseDamageMax := petClawBaseDamageMax[rank]
	spellID := petClawSpellID[rank]

	return hp.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: spellID},
		SpellCode:   SpellCode_HunterPetClaw,
		SpellSchool: core.SpellSchoolPhysical,
		DefenseType: core.DefenseTypeMelee,
		ProcMask:    core.ProcMaskMeleeMHSpecial,
		Flags:       core.SpellFlagMeleeMetrics,

		FocusCost: core.FocusCostOptions{
			Cost: 25,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: PetGCD,
			},
			IgnoreHaste: true,
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := sim.Roll(baseDamageMin, baseDamageMax)
			spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeSpecialHitAndCrit)
		},
	})
}

// petBiteLearnLevels are Pet Bite's eight rank learn levels; source:
// 1.60.1.70009 client spell data ("Bite", ranks 1-8).
var petBiteLearnLevels = []int{1, 8, 16, 24, 32, 40, 48, 56}

// petBiteSpellID is Bite's rank -> spell id, index 0 unused.
var petBiteSpellID = [9]int32{0, 17253, 17255, 17256, 17257, 17258, 17259, 17260, 17261}

// petBiteBaseDamageMin/Max are Bite's rank -> damage roll bounds, index 0
// unused. Only ranks 4, 6, 7 and 8 have a tuned value in this file; ranks
// 1-3 and 5 carry the nearest known rank's numbers forward until real ones
// are sourced.
var petBiteBaseDamageMin = [9]float64{0, 31, 31, 31, 31, 31, 49, 66, 81}
var petBiteBaseDamageMax = [9]float64{0, 37, 37, 37, 37, 37, 59, 80, 91}

func (hp *HunterPet) newBite() *core.Spell {
	rank := core.HighestRankAtLevel(petBiteLearnLevels, hp.Owner.Level)
	if rank == 0 {
		return nil
	}

	baseDamageMin := petBiteBaseDamageMin[rank]
	baseDamageMax := petBiteBaseDamageMax[rank]
	spellID := petBiteSpellID[rank]

	return hp.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: spellID},
		SpellCode:   SpellCode_HunterPetBite,
		SpellSchool: core.SpellSchoolPhysical,
		DefenseType: core.DefenseTypeMelee,
		ProcMask:    core.ProcMaskMeleeMHSpecial,
		Flags:       core.SpellFlagMeleeMetrics,

		FocusCost: core.FocusCostOptions{
			Cost: 35,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: PetGCD,
			},
			CD: core.Cooldown{
				Timer:    hp.NewTimer(),
				Duration: 10 * time.Second,
			},
			IgnoreHaste: true,
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := sim.Roll(baseDamageMin, baseDamageMax)
			spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeSpecialHitAndCrit)
		},
	})
}

// petLightningBreathLearnLevels are Lightning Breath's six rank learn
// levels; source: 1.60.1.70009 client spell data ("Lightning Breath",
// ranks 1-6).
var petLightningBreathLearnLevels = []int{1, 12, 24, 36, 48, 60}

// petLightningBreathSpellID is Lightning Breath's rank -> spell id, index 0
// unused. The old code deliberately reused rank 3's id for rank 4 with a
// "not available in SoD Phase 2" comment; SoD's phases are all released in
// this client now, so rank 4 gets its own real id (25010).
var petLightningBreathSpellID = [7]int32{0, 24844, 25008, 25009, 25010, 25011, 25012}

// petLightningBreathBaseDamageMin/Max are Lightning Breath's rank ->
// damage roll bounds, index 0 unused. Only ranks 3, 5 and 6 have a tuned
// value in this file; ranks 1-2 carry rank 3's numbers backward, and rank 4
// carries rank 3's numbers forward (matching the old code's own rank-3/4
// reuse), until real ones are sourced.
var petLightningBreathBaseDamageMin = [7]float64{0, 36, 36, 36, 36, 78, 99}
var petLightningBreathBaseDamageMax = [7]float64{0, 41, 41, 41, 41, 91, 113}

func (hp *HunterPet) newLightningBreath() *core.Spell {
	rank := core.HighestRankAtLevel(petLightningBreathLearnLevels, hp.Owner.Level)
	if rank == 0 {
		return nil
	}

	baseDamageMin := petLightningBreathBaseDamageMin[rank]
	baseDamageMax := petLightningBreathBaseDamageMax[rank]
	spellID := petLightningBreathSpellID[rank]

	return hp.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: spellID},
		SpellCode:   SpellCode_HunterPetLightningBreath,
		SpellSchool: core.SpellSchoolNature,
		DefenseType: core.DefenseTypeMagic,
		ProcMask:    core.ProcMaskSpellDamage,

		FocusCost: core.FocusCostOptions{
			Cost: 50,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: PetGCD,
			},
			IgnoreHaste: true,
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := sim.Roll(baseDamageMin, baseDamageMax)

			spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)
		},
	})
}

func (hp *HunterPet) newScreech() *core.Spell {
	baseDamageMin := map[int32]float64{
		25: 12,
		40: 12,
		50: 19,
		60: 26,
	}[hp.Owner.Level]

	baseDamageMax := map[int32]float64{
		25: 16,
		40: 16,
		50: 25,
		60: 46,
	}[hp.Owner.Level]

	spellID := map[int32]int32{
		15: 24580,
		40: 24580,
		50: 24581,
		60: 24582,
	}[hp.Owner.Level]

	return hp.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: spellID},
		SpellCode:   SpellCode_HunterPetScreech,
		SpellSchool: core.SpellSchoolPhysical,
		DefenseType: core.DefenseTypeMelee,
		ProcMask:    core.ProcMaskMeleeSpecial,
		Flags:       core.SpellFlagMeleeMetrics,

		FocusCost: core.FocusCostOptions{
			Cost: 20,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: PetGCD,
			},
			IgnoreHaste: true,
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := sim.Roll(baseDamageMin, baseDamageMax)
			// This ability also applies a melee attack power reduction similar to demoralizing shout - left it out for now
			spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeSpecialHitAndCrit)
		},
	})
}

// func (hp *HunterPet) newFuriousHowl() *core.Spell {
// 	actionID := core.ActionID{SpellID: 64495}

// 	petAura := hp.NewTemporaryStatsAura("FuriousHowl", actionID, stats.Stats{stats.AttackPower: 320, stats.RangedAttackPower: 320}, time.Second*20)
// 	ownerAura := hp.hunterOwner.NewTemporaryStatsAura("FuriousHowl", actionID, stats.Stats{stats.AttackPower: 320, stats.RangedAttackPower: 320}, time.Second*20)

// 	howlSpell := hp.RegisterSpell(core.SpellConfig{
// 		ActionID: actionID,

// 		FocusCost: core.FocusCostOptions{
// 			Cost: 20,
// 		},
// 		Cast: core.CastConfig{
// 			CD: core.Cooldown{
// 				Timer:    hp.NewTimer(),
// 				Duration: time.Second * 40,
// 			},
// 		},
// 		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
// 			return hp.IsEnabled()
// 		},
// 		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
// 			petAura.Activate(sim)
// 			ownerAura.Activate(sim)
// 		},
// 	})

// 	hp.hunterOwner.RegisterSpell(core.SpellConfig{
// 		ActionID: actionID,
// 		Flags:    core.SpellFlagAPL | core.SpellFlagMCD,
// 		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
// 			return howlSpell.CanCast(sim, target)
// 		},
// 		ApplyEffects: func(sim *core.Simulation, target *core.Unit, _ *core.Spell) {
// 			howlSpell.Cast(sim, target)
// 		},
// 	})

// 	hp.hunterOwner.AddMajorCooldown(core.MajorCooldown{
// 		Spell: howlSpell,
// 		Type:  core.CooldownTypeDPS,
// 	})

// 	return nil
// }

// petScorpidPoisonLearnLevels are Scorpid Poison's four rank learn levels;
// source: 1.60.1.70009 client spell data ("Scorpid Poison", ranks 1-4).
var petScorpidPoisonLearnLevels = []int{8, 24, 40, 56}

// petScorpidPoisonSpellID is Scorpid Poison's rank -> spell id, index 0
// unused.
var petScorpidPoisonSpellID = [5]int32{0, 24640, 24583, 24586, 24587}

// petScorpidPoisonBaseDamageTick is Scorpid Poison's rank -> tick damage,
// index 0 unused. Rank 1 has no tuned value in this file; it carries rank
// 2's number backward until a real one is sourced.
var petScorpidPoisonBaseDamageTick = [5]float64{0, 3, 3, 6, 8}

func (hp *HunterPet) newScorpidPoison() *core.Spell {
	rank := core.HighestRankAtLevel(petScorpidPoisonLearnLevels, hp.Owner.Level)
	if rank == 0 {
		return nil
	}

	baseDamageTick := petScorpidPoisonBaseDamageTick[rank]
	spellID := petScorpidPoisonSpellID[rank]

	return hp.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: spellID},
		SpellCode:   SpellCode_HunterPetScorpidPoison,
		SpellSchool: core.SpellSchoolNature,
		DefenseType: core.DefenseTypeMelee,
		ProcMask:    core.ProcMaskMeleeMHSpecial,
		Flags:       core.SpellFlagPassiveSpell | core.SpellFlagPoison,

		FocusCost: core.FocusCostOptions{
			Cost: 30,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: PetGCD,
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    hp.NewTimer(),
				Duration: time.Second * 4,
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label:     "ScorpidPoison",
				MaxStacks: 5,
				Duration:  time.Second * 10,
			},
			NumberOfTicks: 5,
			TickLength:    time.Second * 2,

			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, applyStack bool) {
				if !applyStack {
					return
				}

				// only the first stack snapshots the multiplier
				if dot.GetStacks() == 1 {
					attackTable := dot.Spell.Unit.AttackTables[target.UnitIndex][dot.Spell.CastType]
					dot.SnapshotAttackerMultiplier = dot.Spell.AttackerDamageMultiplier(attackTable, true)
					dot.SnapshotBaseDamage = 0
				}

				dot.SnapshotBaseDamage += baseDamageTick
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcAndDealOutcome(sim, target, spell.OutcomeMeleeSpecialHit)
			if !result.Landed() {
				return
			}

			dot := spell.Dot(target)
			dot.ApplyOrRefresh(sim)
			if dot.GetStacks() < dot.MaxStacks {
				dot.AddStack(sim)
				dot.TakeSnapshot(sim, true)
			}
		},
	})
}
