package hunter

import (
	"fmt"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

type HunterPet struct {
	core.Pet

	config PetConfig

	hunterOwner *Hunter

	specialAbility *core.Spell
	focusDump      *core.Spell
	familyAbility  *core.Spell

	uptimePercent    float64
	hasOwnerCooldown bool
}

func (hunter *Hunter) NewHunterPet() *HunterPet {
	if hunter.Options.PetType == proto.Hunter_Options_PetNone {
		return nil
	}
	if hunter.Options.PetUptime <= 0 {
		return nil
	}
	petConfig := PetConfigs[hunter.Options.PetType]

	hunterPetBaseStats := stats.Stats{}

	baseMinDamage := 0.0
	baseMaxDamage := 0.0
	attackSpeed := 2.0

	switch hunter.Options.PetAttackSpeed {
	case proto.Hunter_Options_One:
		attackSpeed = 1.0
	case proto.Hunter_Options_OneTwo:
		attackSpeed = 1.2
	case proto.Hunter_Options_OneThree:
		attackSpeed = 1.3
	case proto.Hunter_Options_OneFour:
		attackSpeed = 1.4
	case proto.Hunter_Options_OneFive:
		attackSpeed = 1.5
	case proto.Hunter_Options_OneSix:
		attackSpeed = 1.6
	case proto.Hunter_Options_OneSeven:
		attackSpeed = 1.7
	case proto.Hunter_Options_Two:
		attackSpeed = 2
	case proto.Hunter_Options_TwoFour:
		attackSpeed = 2.4
	case proto.Hunter_Options_TwoFive:
		attackSpeed = 2.5
	}

	levelScale := hunterPetDamageLevelScale(hunter.Level)
	baseMinDamage = 18.17 * attackSpeed * levelScale
	baseMaxDamage = 27.66 * attackSpeed * levelScale

	hunterPetBaseStats = hunterPetBaseStatsAtLevel(hunter.Level)

	hp := &HunterPet{
		Pet:         core.NewPet(petConfig.Name, &hunter.Character, hunterPetBaseStats, hunter.makeStatInheritance(), true, false),
		config:      petConfig,
		hunterOwner: hunter,

		hasOwnerCooldown: petConfig.SpecialAbility == FuriousHowl,
	}

	hp.Pet.MobType = petConfig.MobType

	hp.EnableAutoAttacks(hp, core.AutoAttackOptions{
		MainHand: core.Weapon{
			BaseDamageMin: baseMinDamage,
			BaseDamageMax: baseMaxDamage,
			SwingSpeed:    attackSpeed,
		},
		AutoSwingMelee: true,
	})

	// Happiness
	hp.PseudoStats.DamageDealtMultiplier *= 1.25

	// Family scalars
	hp.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexPhysical] *= hp.config.Damage
	hp.PseudoStats.ArmorMultiplier *= hp.config.Armor
	hp.MultiplyStat(stats.Health, hp.config.Health)

	hp.AddStatDependency(stats.Strength, stats.AttackPower, 2)

	// No Agility or Intellect crit of its own: a Forever pet's crit chance
	// is the hunter's (see makeStatInheritance), and it takes no player
	// buffs.
	hp.RefusePlayerBuffs()

	core.ApplyPetConsumeEffects(&hp.Character, hunter.Consumes)

	hunter.AddPet(hp)

	return hp
}

func (hp *HunterPet) GetPet() *core.Pet {
	return &hp.Pet
}

func (hp *HunterPet) Initialize() {
	hp.specialAbility = hp.NewPetAbility(hp.config.SpecialAbility, true)
	hp.focusDump = hp.NewPetAbility(hp.config.FocusDump, false)
	hp.familyAbility = hp.NewPetAbility(hp.config.FamilyAbility, false)

	hp.EnableFocusBar(1, func(sim *core.Simulation) {
		if hp.GCD.IsReady(sim) {
			hp.OnGCDReady(sim)
		}
	})
}

func (hp *HunterPet) Reset(_ *core.Simulation) {
	hp.uptimePercent = min(1, max(0, hp.hunterOwner.Options.PetUptime))
	hp.hunterOwner.petAttackPowerSource = higherAttackPower(hp.hunterOwner.GetStats())
}

func (hp *HunterPet) ExecuteCustomRotation(sim *core.Simulation) {
	percentRemaining := sim.GetRemainingDurationPercent()
	if percentRemaining < 1.0-hp.uptimePercent { // once fight is % completed, disable pet.
		hp.Disable(sim)
		return
	}

	if hp.hasOwnerCooldown && hp.CurrentFocus() < 50 {
		// When a major ability (Furious Howl or Savage Rend) is ready, pool enough
		// energy to use on-demand.
		return
	}

	target := hp.CurrentTarget

	// using Cast() directly is very expensive, since cast failures are logged, involving string operations
	tryCast := func(spell *core.Spell) bool {
		if !spell.CanCast(sim, target) {
			return false
		}
		if !spell.Cast(sim, target) {
			panic(fmt.Sprintf("Cast failed after CanCast() for spell %d", spell.SpellID))
		}
		return true
	}

	if hp.familyAbility != nil && hp.familyAbility.IsReady(sim) {
		// Pool focus for the ability rather than spending it on the dump.
		if !tryCast(hp.familyAbility) && hp.GCD.IsReady(sim) {
			hp.WaitUntil(sim, sim.CurrentTime+time.Millisecond*500)
		}
		return
	}

	if hp.focusDump == nil {
		if !tryCast(hp.specialAbility) && hp.GCD.IsReady(sim) {
			hp.WaitUntil(sim, sim.CurrentTime+time.Millisecond*500)
		}
		return
	}
	if hp.specialAbility == nil {
		if !tryCast(hp.focusDump) && hp.GCD.IsReady(sim) {
			hp.WaitUntil(sim, sim.CurrentTime+time.Millisecond*500)
		}
		return
	}

	if hp.config.CustomRotation != nil {
		hp.config.CustomRotation(sim, hp, tryCast)
	} else {
		if hp.specialAbility.IsReady(sim) {
			if !tryCast(hp.specialAbility) && hp.GCD.IsReady(sim) {
				hp.WaitUntil(sim, sim.CurrentTime+time.Millisecond*500)
			}
		} else if hp.focusDump.IsReady(sim) {
			if !tryCast(hp.focusDump) && hp.GCD.IsReady(sim) {
				hp.WaitUntil(sim, sim.CurrentTime+time.Millisecond*500)
			}
		}
	}
}

// petAttackPowerShare is the part of the hunter's attack power a Forever
// pet inherits.
const petAttackPowerShare = 0.10

// makeStatInheritance is Forever's pet rework: the pet inherits 10% of the
// hunter's attack power, melee or ranged, whichever is higher, and all of
// the hunter's crit and hit chance. The client's Hunter Pet Scaling
// (415429) carries those channels (aura 99, aura 52, aura 54/55) with
// server-computed amounts; the 10% and 100% crit are from the Warcraft
// Tavern Forever hunter and pet guides and MMOGAH. Hit follows wowsims/sod
// on the same client spell and this fork's demons (Warlock Pet Scaling
// 416189 has the same hit channels). The inheritance is linear in the
// chosen stat, so the same function maps the owner's dynamic stat changes.
// 415429 has two crit channels, aura 52 (melee and ranged) and aura 57
// (spell): the pet takes the hunter's every-table crit and each table's
// own crit onto the same table. The beta logs agree: paired per hunter, a
// pet's magic abilities crit about as often as its melee (11.7% of 1,920
// against 11.0% of 12,163, levels 20 to 35).
func (hunter *Hunter) makeStatInheritance() core.PetStatInheritance {
	return func(ownerStats stats.Stats) stats.Stats {
		return stats.Stats{
			stats.AttackPower: ownerStats[hunter.petAttackPowerSource] * petAttackPowerShare,
			stats.Crit:        ownerStats[stats.Crit],
			stats.MeleeCrit:   ownerStats[stats.MeleeCrit],
			stats.SpellCrit:   ownerStats[stats.SpellCrit],
			stats.Hit:         ownerStats[stats.Hit],
		}
	}
}

// higherAttackPower names the hunter's larger attack power stat, the one
// the pet inherits from. It is chosen once per iteration from the
// hunter's stats at the pet's reset.
func higherAttackPower(ownerStats stats.Stats) stats.Stat {
	if ownerStats[stats.AttackPower] > ownerStats[stats.RangedAttackPower] {
		return stats.AttackPower
	}
	return stats.RangedAttackPower
}

type PetConfig struct {
	Name    string
	MobType proto.MobType

	SpecialAbility PetAbilityType
	FocusDump      PetAbilityType
	// FamilyAbility is the family's trainable periodic-damage ability,
	// cast on its cooldown ahead of the focus dump.
	FamilyAbility PetAbilityType

	Health float64
	Armor  float64
	Damage float64

	CustomRotation func(*core.Simulation, *HunterPet, func(*core.Spell) bool)
}

// Abilities reference: https://wotlk.wowhead.com/hunter-pets
// https://wotlk.wowhead.com/guides/hunter-dps-best-pets-taming-loyalty-burning-crusade-classic
var PetConfigs = map[proto.Hunter_Options_PetType]PetConfig{
	proto.Hunter_Options_Cat: {
		Name:    "Cat",
		MobType: proto.MobType_MobTypeBeast,

		SpecialAbility: Bite,
		FocusDump:      Claw,

		Health: 0.98,
		Armor:  1.00,
		Damage: 1.10,

		// Bite whenever it is ready (pooling for its 35 focus), Claw
		// otherwise: Bite deals more damage a focus. The old gate, focus
		// regen above Claw's cost / 1.6 (15.6 a second), never opened at
		// a pet's 5 to 6 a second.
		CustomRotation: func(sim *core.Simulation, hp *HunterPet, tryCast func(*core.Spell) bool) {
			if hp.specialAbility.CD.IsReady(sim) {
				if !tryCast(hp.specialAbility) && hp.GCD.IsReady(sim) {
					hp.WaitUntil(sim, sim.CurrentTime+time.Millisecond*500)
				}
			} else {
				if !tryCast(hp.focusDump) && hp.GCD.IsReady(sim) {
					hp.WaitUntil(sim, sim.CurrentTime+time.Millisecond*500)
				}
			}
		},
	},
	proto.Hunter_Options_WindSerpent: {
		Name:    "Wind Serpent",
		MobType: proto.MobType_MobTypeBeast,

		SpecialAbility: Bite,
		FocusDump:      LightningBreath,

		Health: 1.00,
		Armor:  1.00,
		Damage: 1.07,
	},
	proto.Hunter_Options_Bat: {
		Name:    "Bat",
		MobType: proto.MobType_MobTypeBeast,

		SpecialAbility: Bite,
		FocusDump:      Screech,

		Health: 1.00,
		Armor:  1.00,
		Damage: 1.07,

		CustomRotation: func(sim *core.Simulation, hp *HunterPet, tryCast func(*core.Spell) bool) {
			if hp.specialAbility.CD.IsReady(sim) {
				if !tryCast(hp.specialAbility) && hp.GCD.IsReady(sim) {
					hp.WaitUntil(sim, sim.CurrentTime+time.Millisecond*500)
				}
			} else {
				if !tryCast(hp.focusDump) && hp.GCD.IsReady(sim) {
					hp.WaitUntil(sim, sim.CurrentTime+time.Millisecond*500)
				}
			}
		},
	},
	proto.Hunter_Options_Bear: {
		Name:    "Bear",
		MobType: proto.MobType_MobTypeBeast,

		SpecialAbility: Bite,
		FocusDump:      Claw,

		Health: 1.08,
		Armor:  1.05,
		Damage: 0.91,
	},
	proto.Hunter_Options_Boar: {
		Name:    "Boar",
		MobType: proto.MobType_MobTypeBeast,

		//SpecialAbility: Gore,
		FocusDump: Bite,

		Health: 1.04,
		Armor:  1.09,
		Damage: 0.90,
	},
	proto.Hunter_Options_CarrionBird: {
		Name:    "Carrion Bird",
		MobType: proto.MobType_MobTypeBeast,

		SpecialAbility: Bite, // Screech
		FocusDump:      Claw,

		Health: 1.00,
		Armor:  1.05,
		Damage: 1.00,
	},
	proto.Hunter_Options_Owl: {
		Name:    "Owl",
		MobType: proto.MobType_MobTypeBeast,

		//SpecialAbility: Screech,
		FocusDump: Claw,

		Health: 1.00,
		Armor:  1.00,
		Damage: 1.07,
	},
	proto.Hunter_Options_Crab: {
		Name:    "Crab",
		MobType: proto.MobType_MobTypeBeast,

		FamilyAbility: Pinch,
		FocusDump:     Claw,

		Health: 0.96,
		Armor:  1.13,
		Damage: 0.95,
	},
	proto.Hunter_Options_Crocolisk: {
		Name:    "Crocolisk",
		MobType: proto.MobType_MobTypeBeast,

		FamilyAbility: Dismember,
		FocusDump:     Bite,

		Health: 0.95,
		Armor:  1.10,
		Damage: 1.00,
	},
	proto.Hunter_Options_Gorilla: {
		Name:    "Gorilla",
		MobType: proto.MobType_MobTypeBeast,

		// SpecialAbility: Thunderstomp,
		FocusDump: Bite,

		Health: 1.04,
		Armor:  1.00,
		Damage: 1.02,
	},
	proto.Hunter_Options_Hyena: {
		Name:    "Hyena",
		MobType: proto.MobType_MobTypeBeast,

		FamilyAbility: TendonRip,
		FocusDump:     Bite,

		Health: 1.00,
		Armor:  1.05,
		Damage: 1.00,
	},
	proto.Hunter_Options_Raptor: {
		Name:    "Raptor",
		MobType: proto.MobType_MobTypeBeast,

		SpecialAbility: Bite,
		FamilyAbility:  SavageRend,
		FocusDump:      Claw,

		Health: 0.95,
		Armor:  1.03,
		Damage: 1.10,
	},
	proto.Hunter_Options_Scorpid: {
		Name:    "Scorpid",
		MobType: proto.MobType_MobTypeBeast,

		SpecialAbility: ScorpidPoison,
		FocusDump:      Claw,

		Health: 1.00,
		Armor:  1.10,
		Damage: 0.94,

		CustomRotation: func(sim *core.Simulation, hp *HunterPet, tryCast func(*core.Spell) bool) {
			target := hp.CurrentTarget

			if (hp.specialAbility.Dot(target).GetStacks() < hp.specialAbility.Dot(target).MaxStacks || hp.specialAbility.Dot(target).RemainingDuration(sim) < time.Second*3) && hp.CurrentFocus() < 90 {
				if !tryCast(hp.specialAbility) && hp.GCD.IsReady(sim) {
					hp.WaitUntil(sim, sim.CurrentTime+time.Millisecond*500)
				}
			} else {
				if !tryCast(hp.focusDump) && hp.GCD.IsReady(sim) {
					hp.WaitUntil(sim, sim.CurrentTime+time.Millisecond*500)
				}
			}
		},
	},
	proto.Hunter_Options_Spider: {
		Name:    "Spider",
		MobType: proto.MobType_MobTypeBeast,

		FamilyAbility: Web,
		FocusDump:     Bite,

		Health: 1.00,
		Armor:  1.00,
		Damage: 1.07,
	},
	proto.Hunter_Options_Tallstrider: {
		Name:    "Tallstrider",
		MobType: proto.MobType_MobTypeBeast,

		FamilyAbility: DustCloud,
		FocusDump:     Bite,

		Health: 1.05,
		Armor:  1.00,
		Damage: 1.00,
	},
	proto.Hunter_Options_Turtle: {
		Name:    "Turtle",
		MobType: proto.MobType_MobTypeBeast,

		// SpecialAbility: ShellShield,
		FocusDump: Bite,

		Health: 1.00,
		Armor:  1.13,
		Damage: 0.90,
	},
	proto.Hunter_Options_Wolf: {
		Name:    "Wolf",
		MobType: proto.MobType_MobTypeBeast,

		// SpecialAbility: FuriousHowl,
		FocusDump: Bite,

		Health: 1.00,
		Armor:  1.05,
		Damage: 1.00,
	},
}

// hunterPetLevel60Stamina is the engine's level-60 pet Stamina. Era's
// pet_levelstats carries 256; the engine's level-60 sheet (wowsims) is
// kept, and lower levels scale it by the table's Stamina curve.
const hunterPetLevel60Stamina = 274.0

// hunterPetBaseStatsAtLevel is a pet's base sheet at the hunter's level (a
// hunter's pet is its own level): Era's pet_levelstats (hunterPetLevelStats),
// with Stamina on the engine's level-60 value, and the -20 attack power the
// pet's Strength x 2 is offset by.
func hunterPetBaseStatsAtLevel(level int32) stats.Stats {
	row := hunterPetLevelStats[hunterPetStatsIndex(level)]
	top := hunterPetLevelStats[len(hunterPetLevelStats)-1]
	return stats.Stats{
		stats.Strength:  row[0],
		stats.Agility:   row[1],
		stats.Stamina:   hunterPetLevel60Stamina * row[2] / top[2],
		stats.Intellect: row[3],
		stats.Spirit:    row[4],

		stats.AttackPower: -20,
	}
}

// hunterPetStatsIndex is level's row in hunterPetLevelStats, clamped to 1..60.
func hunterPetStatsIndex(level int32) int {
	return int(min(max(level, 1), int32(len(hunterPetLevelStats)))) - 1
}

// hunterPetDamageLevelScale scales the level-60 weapon roll to the pet's
// level: Era's cores make a hunter pet's weapon damage proportional to its
// level (vmangos Pet::InitStatsForLevel: petlevel x 1.2075 to 1.5225 a 2 s
// swing; cmangos linear in petlevel).
func hunterPetDamageLevelScale(level int32) float64 {
	return float64(hunterPetStatsIndex(level)+1) / float64(len(hunterPetLevelStats))
}
