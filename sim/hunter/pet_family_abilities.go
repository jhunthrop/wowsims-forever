package hunter

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// The trainable Forever family abilities the client teaches (each has a
// SkillLineAbility row on its pet family's skill line): Savage Rend
// (Raptor), Tendon Rip (Hyena) and Web (Spider), which are periodic
// damage; Dismember (Crocolisk) and Pinch (Crab), which are direct hits;
// and Dust Cloud (Tallstrider), an armor reduction. Each is read from its
// generated rank table (id, learn level, focus cost, cooldown and the
// amount), and the pet casts it as soon as it is ready, ahead of the
// focus dump.
//
// Sonic Blast (Bat) is not here: none of its five ranks has a
// SkillLineAbility row, so the client does not teach it (it is in the same
// position as Hydra Shot, design/reviews/2026-10-07-hunter-hydra-lacerate-pets.md).
// The earlier file left Dismember and Pinch out because a hit returns less
// damage per focus than the family's Claw or Bite; a pet's autocast uses
// an ability whenever it is ready, so they are cast now.

// petFamilyTable is the generated rank table every family ability shares.
type petFamilyTable struct {
	name       string
	spellCode  int32
	spellIDs   []int32
	levels     []int
	focusCosts []float64
	cooldownMS []int32
}

// petPeriodicAbility is a family ability that deals periodic damage.
type petPeriodicAbility struct {
	petFamilyTable
	school     core.SpellSchool
	damage     [][]float64
	ticks      int32
	tickLength time.Duration
}

// petDirectAbility is a family ability that deals one physical hit; the
// client's row carries no attack power coefficient, so the roll is flat.
type petDirectAbility struct {
	petFamilyTable
	damage [][]float64
}

var (
	savageRendAbility = petPeriodicAbility{
		petFamilyTable: petFamilyTable{
			name:       "Savage Rend",
			spellCode:  SpellCode_HunterPetSavageRend,
			spellIDs:   SavageRendSpellId[:],
			levels:     SavageRendLevel[:],
			focusCosts: SavageRendManaCost[:],
			cooldownMS: SavageRendCooldownMS[:],
		},
		school:     core.SpellSchoolPhysical,
		damage:     SavageRendBaseDamage[:],
		ticks:      6, // 18 s over 3 s
		tickLength: 3 * time.Second,
	}
	tendonRipAbility = petPeriodicAbility{
		petFamilyTable: petFamilyTable{
			name:       "Tendon Rip",
			spellCode:  SpellCode_HunterPetTendonRip,
			spellIDs:   TendonRipSpellId[:],
			levels:     TendonRipLevel[:],
			focusCosts: TendonRipManaCost[:],
			cooldownMS: TendonRipCooldownMS[:],
		},
		school:     core.SpellSchoolPhysical,
		damage:     TendonRipBaseDamage[:],
		ticks:      3, // 9 s over 3 s
		tickLength: 3 * time.Second,
	}
	webAbility = petPeriodicAbility{
		petFamilyTable: petFamilyTable{
			name:       "Web",
			spellCode:  SpellCode_HunterPetWeb,
			spellIDs:   WebSpellId[:],
			levels:     WebLevel[:],
			focusCosts: WebManaCost[:],
			cooldownMS: WebCooldownMS[:],
		},
		school:     core.SpellSchoolNature,
		damage:     WebBaseDamage[:],
		ticks:      4, // 4 s over 1 s
		tickLength: time.Second,
	}
	dismemberAbility = petDirectAbility{
		petFamilyTable: petFamilyTable{
			name:       "Dismember",
			spellCode:  SpellCode_HunterPetDismember,
			spellIDs:   DismemberSpellId[:],
			levels:     DismemberLevel[:],
			focusCosts: DismemberManaCost[:],
			cooldownMS: DismemberCooldownMS[:],
		},
		damage: DismemberBaseDamage[:],
	}
	pinchAbility = petDirectAbility{
		petFamilyTable: petFamilyTable{
			name:       "Pinch",
			spellCode:  SpellCode_HunterPetPinch,
			spellIDs:   PinchSpellId[:],
			levels:     PinchLevel[:],
			focusCosts: PinchManaCost[:],
			cooldownMS: PinchCooldownMS[:],
		},
		damage: PinchBaseDamage[:],
	}
)

// dustCloudArmorReduction is each Dust Cloud rank's armor reduction, the
// client's effect 0 (aura 22, base points -65/-175/-285/-395/-505), and
// dustCloudDuration its 30 second duration. The ability has no cooldown;
// the pet renews it as the debuff lapses, so the engine gives it the
// duration as a cooldown.
var dustCloudArmorReduction = [DustCloudRanks + 1]float64{0, 65, 175, 285, 395, 505}

const dustCloudDuration = 30 * time.Second

// petFamilyConfig is the config every family ability shares: the focus
// cost, the pet GCD and the rank's cooldown.
func (hp *HunterPet) petFamilyConfig(table petFamilyTable, rank int, school core.SpellSchool, defense core.DefenseType) core.SpellConfig {
	return core.SpellConfig{
		ActionID:    core.ActionID{SpellID: table.spellIDs[rank]},
		SpellCode:   table.spellCode,
		SpellSchool: school,
		DefenseType: defense,
		ProcMask:    core.ProcMaskMeleeMHSpecial,
		Flags:       core.SpellFlagMeleeMetrics | core.SpellFlagNoOnCastComplete,

		FocusCost: core.FocusCostOptions{
			Cost: table.focusCosts[rank],
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: PetGCD,
			},
			CD: core.Cooldown{
				Timer:    hp.NewTimer(),
				Duration: time.Duration(table.cooldownMS[rank]) * time.Millisecond,
			},
			IgnoreHaste: true,
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
	}
}

func (hp *HunterPet) newDirectAbility(ability petDirectAbility) *core.Spell {
	rank := core.HighestRankAtLevel(ability.levels[1:], hp.Owner.Level)
	if rank == 0 {
		return nil
	}
	low, high := ability.damage[rank][0], ability.damage[rank][1]

	config := hp.petFamilyConfig(ability.petFamilyTable, rank, core.SpellSchoolPhysical, core.DefenseTypeMelee)
	config.ClientBaseDamage = [2]float64{low, high}
	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		spell.CalcAndDealDamage(sim, target, sim.Roll(low, high), spell.OutcomeMeleeSpecialHitAndCrit)
	}
	return hp.RegisterSpell(config)
}

// newDustCloud is the Tallstrider's armor reduction: a minor armor
// reduction (the category Faerie Fire is in), so it does not stack with
// Faerie Fire. Unconfirmed: whether Forever lets it stack; the client rows
// state the amount and the duration only.
func (hp *HunterPet) newDustCloud() *core.Spell {
	rank := core.HighestRankAtLevel(DustCloudLevel[1:], hp.Owner.Level)
	if rank == 0 {
		return nil
	}
	reduction := dustCloudArmorReduction[rank]
	auras := hp.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
		return core.MinorArmorReductionAura(target, "Dust Cloud", DustCloudSpellId[rank], reduction, dustCloudDuration)
	})

	config := hp.petFamilyConfig(petFamilyTable{
		name:       "Dust Cloud",
		spellCode:  SpellCode_HunterPetDustCloud,
		spellIDs:   DustCloudSpellId[:],
		focusCosts: DustCloudManaCost[:],
		cooldownMS: []int32{0, 0, 0, 0, 0, 0},
	}, rank, core.SpellSchoolPhysical, core.DefenseTypeMelee)
	config.Cast.CD.Duration = dustCloudDuration
	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		spell.CalcAndDealOutcome(sim, target, spell.OutcomeMeleeSpecialHitNoHitCounter)
		auras.Get(target).Activate(sim)
	}
	return hp.RegisterSpell(config)
}

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

	config := hp.petFamilyConfig(ability.petFamilyTable, rank, ability.school, defense)
	config.ClientBaseDamage = [2]float64{tickDamage, tickDamage}
	config.Dot = core.DotConfig{
		Aura:          core.Aura{Label: ability.name},
		NumberOfTicks: ability.ticks,
		TickLength:    ability.tickLength,
		OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
			dot.Snapshot(target, tickDamage, isRollover)
		},
		OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
			dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
		},
	}
	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		result := spell.CalcOutcome(sim, target, outcome(spell))
		if result.Landed() {
			spell.Dot(target).Apply(sim)
		}
		spell.DealOutcome(sim, result)
	}
	return hp.RegisterSpell(config)
}
