package warlock

import (
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

type OnPetDisable func(sim *core.Simulation, isSacrifice bool)

type WarlockPet struct {
	core.Pet

	OnPetDisable OnPetDisable

	owner *Warlock

	primaryAbility   *core.Spell
	secondaryAbility *core.Spell

	SoulLinkAura *core.Aura

	LifeTapManaMetrics *core.ResourceMetrics

	manaPooling bool
}

type PetConfig struct {
	Name          string
	PowerModifier float64 // GetUnitPowerModifier("pet")
	Stats         stats.Stats
	AutoAttacks   core.AutoAttackOptions
}

func (warlock *Warlock) setDefaultActivePet() {
	switch warlock.Options.Summon {
	case proto.WarlockOptions_Imp:
		warlock.ActivePet = warlock.Imp
	case proto.WarlockOptions_Felhunter:
		warlock.ActivePet = warlock.Felhunter
	case proto.WarlockOptions_Succubus:
		warlock.ActivePet = warlock.Succubus
	case proto.WarlockOptions_Voidwalker:
		warlock.ActivePet = warlock.Voidwalker
	}
}

func (warlock *Warlock) changeActivePet(sim *core.Simulation, newPet *WarlockPet, isSacrifice bool) {
	if warlock.ActivePet != nil {
		warlock.ActivePet.Disable(sim, isSacrifice)

		// Sacrificed pets lose all buffs
		if isSacrifice {
			for _, aura := range warlock.ActivePet.GetAuras() {
				aura.Deactivate(sim)
			}
		}

		if warlock.MasterDemonologistAura != nil {
			warlock.MasterDemonologistAura.Deactivate(sim)
		}
	}

	warlock.ActivePet = newPet

	if newPet != nil {
		newPet.Enable(sim, newPet)
	}
}

func (warlock *Warlock) registerPets() {
	warlock.petSpellPowerSchool = stats.ShadowPower
	warlock.Felhunter = warlock.makeFelhunter()
	warlock.Imp = warlock.makeImp()
	warlock.Succubus = warlock.makeSuccubus()
	warlock.Voidwalker = warlock.makeVoidwalker()

	warlock.BasePets = []*WarlockPet{warlock.Felhunter, warlock.Imp, warlock.Succubus, warlock.Voidwalker}
}

func (warlock *Warlock) makePet(cfg PetConfig, enabledOnStart bool) *WarlockPet {
	wp := &WarlockPet{
		Pet:          core.NewPet(cfg.Name, &warlock.Character, cfg.Stats, warlock.makeStatInheritance(), enabledOnStart, false),
		owner:        warlock,
		OnPetDisable: func(sim *core.Simulation, isSacrifice bool) {},
	}

	wp.EnableManaBarWithModifier(cfg.PowerModifier)

	if cfg.Name == "Imp" {
		// Imp gets 1mp/5 non casting regen per spirit
		wp.PseudoStats.SpiritRegenMultiplier = 1
		wp.PseudoStats.SpiritRegenRateCasting = 0
		wp.SpiritManaRegenPerSecond = func() float64 {
			// 1mp5 per spirit
			return wp.GetStat(stats.Spirit) / 5
		}

		// Mage spell crit scaling for imp
		wp.AddStatDependency(stats.Intellect, stats.Crit, core.SpellCritPerIntAtLevel(proto.Class_ClassMage, wp.Level)*core.CritRatingPerCritChance)
	} else {
		// Warrior scaling for all other pets
		wp.AddStat(stats.AttackPower, -20)
		wp.AddStatDependency(stats.Strength, stats.AttackPower, 2)

		// Warrior crit scaling.
		//
		// The Intellect line is a no-op (the warrior row has no Intellect
		// rate at any level) and is kept so the pet wires both dependencies
		// exactly as AddCritStatDependencies does for a player.
		// unconfirmed: whether a Forever warrior-scaled pet should convert
		// Intellect to Crit at all is unpublished.
		wp.AddStatDependency(stats.Agility, stats.Crit, core.CritPerAgiAtLevel(proto.Class_ClassWarrior, wp.Level)*core.CritRatingPerCritChance)
		wp.AddStatDependency(stats.Intellect, stats.Crit, core.SpellCritPerIntAtLevel(proto.Class_ClassWarrior, wp.Level)*core.CritRatingPerCritChance)

		// Imps generally don't melee
		wp.EnableAutoAttacks(wp, cfg.AutoAttacks)
	}

	// Unholy Power: "Increases all damage done by your Imp, Voidwalker,
	// Succubus, Incubus, and Felhunter pets by 2%" a rank; the Forever
	// Firebolt tooltip applies it too, so it is pet-wide, not melee only.
	wp.PseudoStats.DamageDealtMultiplier *= 1 + unholyPowerDamagePerRank*float64(warlock.Talents.UnholyPower)

	// Forever demons scale from the warlock (Warlock Pet Scaling 416189, the
	// same server-computed channels as Hunter Pet Scaling 415429; see
	// makeStatInheritance) and, like hunter pets, take no player buffs.
	wp.RefusePlayerBuffs()

	core.ApplyPetConsumeEffects(&wp.Character, warlock.Consumes)

	warlock.AddPet(wp)

	return wp
}

func (warlock *Warlock) registerPetAbilities() {
	warlock.Imp.registerImpFireboltSpell()
	warlock.Succubus.registerSuccubusLashOfPainSpell()
}

func (wp *WarlockPet) GetPet() *core.Pet {
	return &wp.Pet
}

func (wp *WarlockPet) Initialize() {
}

func (wp *WarlockPet) Reset(_ *core.Simulation) {
	wp.manaPooling = false
	wp.owner.petSpellPowerSchool = higherSchoolPower(wp.owner.GetStats())
}

func (wp *WarlockPet) Disable(sim *core.Simulation, isSacrifice bool) {
	wp.Pet.Disable(sim)

	if wp.OnPetDisable != nil {
		wp.OnPetDisable(sim, isSacrifice)
	}
}

func (wp *WarlockPet) ApplyOnPetDisable(newOnPetDisable OnPetDisable) {
	oldOnPetDisable := wp.OnPetDisable
	if oldOnPetDisable == nil {
		wp.OnPetDisable = oldOnPetDisable
	} else {
		wp.OnPetDisable = func(sim *core.Simulation, isSacrifice bool) {
			oldOnPetDisable(sim, isSacrifice)
			newOnPetDisable(sim, isSacrifice)
		}
	}
}

func (wp *WarlockPet) ExecuteCustomRotation(sim *core.Simulation) {
	if !wp.IsEnabled() || wp.primaryAbility == nil {
		return
	}

	if wp.manaPooling {
		maxPossibleCasts := sim.GetRemainingDuration().Seconds() / wp.primaryAbility.CurCast.CastTime.Seconds()

		if wp.CurrentMana() > (maxPossibleCasts*wp.primaryAbility.CurCast.Cost)*0.75 {
			wp.manaPooling = false
			wp.WaitUntil(sim, sim.CurrentTime+10*time.Millisecond)
			return
		}

		if wp.CurrentMana() >= wp.MaxMana()*0.94 {
			wp.manaPooling = false
			wp.WaitUntil(sim, sim.CurrentTime+10*time.Millisecond)
			return
		}

		if wp.manaPooling {
			return
		}
	}

	if !wp.primaryAbility.IsReady(sim) {
		wp.WaitUntil(sim, wp.primaryAbility.CD.ReadyAt())
		return
	}

	if wp.Unit.CurrentMana() >= wp.primaryAbility.CurCast.Cost {
		wp.primaryAbility.Cast(sim, wp.CurrentTarget)
	} else if !wp.owner.Options.PetPoolMana {
		wp.manaPooling = true
	}
}

// Warlock Pet Scaling (client 416189, "Owner Power Scaling") gives a demon
// channels for Stamina, Intellect, armor and the five resistances, melee
// attack power, all-school spell damage, hit, haste, crit and mana regen;
// the server computes every amount and Blizzard publishes none. The shares
// below are Classic's, which two sources give for this same client spell:
// wowsims' Season of Discovery engine (wowsims/sod sim/warlock/pet.go,
// commits 378fec876, 70f6b1327 and 61102b576, 2024) and warcraft.wiki.gg's
// Warlock minion page (75% Stamina, 30% Intellect, 57% of spell damage as
// attack power, 15% as spell damage). The 35% armor is wowsims' and TBC's;
// the 40% resistances are TBC's alone (single source; the client has the
// channels). wowsims' tested 56.5% is taken over the wiki's rounded 57%.
const (
	petStaminaShare             = 0.75
	petIntellectShare           = 0.30
	petArmorShare               = 0.35
	petResistanceShare          = 0.40
	petAttackPowerPerSpellPower = 0.565
	petSpellDamagePerSpellPower = 0.15
)

// makeStatInheritance maps the warlock's stats to the demon's. Hit is the
// warlock's own, as in wowsims' SoD engine (single source; client channels
// aura 54 and 55). Crit is not inherited: the client has the channels, but
// 29 Forever beta logs (foreverlogs.gg, levels 13 to 30) show Imp Firebolt
// critting 2.2% of 2,383 hits and demon melee 1.4% of 2,432 against the
// warlock's own Shadow Bolt 5.6%, which a 100% share cannot produce. Haste
// is not inherited: this engine carries haste as speed multipliers, not
// stats. The mapping is linear in the owner's stats, so the same function
// maps the owner's mid-fight stat changes; the school power it reads is
// fixed per iteration (petSpellPowerSchool).
func (warlock *Warlock) makeStatInheritance() core.PetStatInheritance {
	return func(owner stats.Stats) stats.Stats {
		spellPower := owner[stats.SpellPower] + owner[stats.SpellDamage] + owner[warlock.petSpellPowerSchool]
		return stats.Stats{
			stats.Stamina:          owner[stats.Stamina] * petStaminaShare,
			stats.Intellect:        owner[stats.Intellect] * petIntellectShare,
			stats.Armor:            owner[stats.Armor] * petArmorShare,
			stats.ArcaneResistance: owner[stats.ArcaneResistance] * petResistanceShare,
			stats.FireResistance:   owner[stats.FireResistance] * petResistanceShare,
			stats.FrostResistance:  owner[stats.FrostResistance] * petResistanceShare,
			stats.NatureResistance: owner[stats.NatureResistance] * petResistanceShare,
			stats.ShadowResistance: owner[stats.ShadowResistance] * petResistanceShare,
			stats.AttackPower:      spellPower * petAttackPowerPerSpellPower,
			stats.SpellDamage:      spellPower * petSpellDamagePerSpellPower,
			stats.Hit:              owner[stats.Hit],
		}
	}
}

// higherSchoolPower names the warlock's larger school power, Fire or Shadow,
// the one a demon inherits beside the all-school spell power.
func higherSchoolPower(owner stats.Stats) stats.Stat {
	if owner[stats.FirePower] > owner[stats.ShadowPower] {
		return stats.FirePower
	}
	return stats.ShadowPower
}
