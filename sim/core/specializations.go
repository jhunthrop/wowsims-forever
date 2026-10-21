package core

import (
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

///////////////////////////////////////////////////////////////////////////
//                            Weapon Specialization Auras
///////////////////////////////////////////////////////////////////////////

// Forever's racial weapon specializations are not vanilla's +5 weapon
// skill. The client (1.60.1.70009) states them as critical strike chance
// with all spells and attacks while a weapon of the type is equipped:
// Sword Specialization 20597 "by 2% while you have a sword or two-handed
// sword equipped", Axe Specialization 20574 "by 1% while you have an axe
// or a two-handed axe equipped", Mace Specialization 1259719 "by 1% while
// you have a mace or two-handed mace equipped". Either hand qualifies;
// two-handers share the one-hand WeaponType.
const (
	swordSpecializationCritPercent = 2
	axeSpecializationCritPercent   = 1
	maceSpecializationCritPercent  = 1
)

func (character *Character) SwordSpecializationAura() *Aura {
	return character.weaponSpecializationCritAura("Sword Specialization", proto.WeaponType_WeaponTypeSword, swordSpecializationCritPercent)
}

func (character *Character) AxeSpecializationAura() *Aura {
	return character.weaponSpecializationCritAura("Axe Specialization", proto.WeaponType_WeaponTypeAxe, axeSpecializationCritPercent)
}

func (character *Character) MaceSpecializationAura() *Aura {
	return character.weaponSpecializationCritAura("Mace Specialization", proto.WeaponType_WeaponTypeMace, maceSpecializationCritPercent)
}

// HasWeaponOfType reports whether either hand holds a weapon of the type.
func (character *Character) HasWeaponOfType(weaponType proto.WeaponType) bool {
	for _, weapon := range []*Item{character.GetMHWeapon(), character.GetOHWeapon()} {
		if weapon != nil && weapon.WeaponType == weaponType {
			return true
		}
	}
	return false
}

func (character *Character) weaponSpecializationCritAura(label string, weaponType proto.WeaponType, critPercent float64) *Aura {
	bonus := critPercent * CritRatingPerCritChance
	return character.GetOrRegisterAura(Aura{
		Label:      label,
		BuildPhase: CharacterBuildPhaseGear,
		Duration:   NeverExpires,
		OnGain: func(aura *Aura, sim *Simulation) {
			if character.HasWeaponOfType(weaponType) {
				character.AddBuildPhaseStatDynamic(sim, stats.Crit, bonus)
			}
		},
		OnExpire: func(aura *Aura, sim *Simulation) {
			if character.HasWeaponOfType(weaponType) {
				character.AddBuildPhaseStatDynamic(sim, stats.Crit, -bonus)
			}
		},
	})
}

func (character *Character) DaggerSpecializationAura() *Aura {
	return character.GetOrRegisterAura(Aura{
		Label:      "Dagger Skill Specialization",
		BuildPhase: CharacterBuildPhaseGear,
		Duration:   NeverExpires,
		OnGain: func(aura *Aura, sim *Simulation) {
			character.PseudoStats.DaggersSkill += 5
		},
	})
}

func (character *Character) FistWeaponSpecializationAura() *Aura {
	return character.GetOrRegisterAura(Aura{
		Label:      "Fist Weapon Skill Specialization",
		BuildPhase: CharacterBuildPhaseGear,
		Duration:   NeverExpires,
		OnGain: func(aura *Aura, sim *Simulation) {
			character.PseudoStats.UnarmedSkill += 5
		},
	})
}

func (character *Character) PoleWeaponSpecializationAura() *Aura {
	return character.GetOrRegisterAura(Aura{
		Label:      "Pole Weapon Skill Specialization",
		BuildPhase: CharacterBuildPhaseGear,
		Duration:   NeverExpires,
		OnGain: func(aura *Aura, sim *Simulation) {
			character.PseudoStats.StavesSkill += 5
			character.PseudoStats.PolearmsSkill += 5
		},
	})
}

func (character *Character) GunSpecializationAura() *Aura {
	return character.GetOrRegisterAura(Aura{
		Label:      "Gun Skill Specialization",
		BuildPhase: CharacterBuildPhaseGear,
		Duration:   NeverExpires,
		OnGain: func(aura *Aura, sim *Simulation) {
			character.PseudoStats.GunsSkill += 5
		},
	})
}

func (character *Character) BowSpecializationAura() *Aura {
	return character.GetOrRegisterAura(Aura{
		Label:      "Bow Skill Specialization",
		BuildPhase: CharacterBuildPhaseGear,
		Duration:   NeverExpires,
		OnGain: func(aura *Aura, sim *Simulation) {
			character.PseudoStats.BowsSkill += 5
		},
	})
}

func (character *Character) CrossbowSpecializationAura() *Aura {
	return character.GetOrRegisterAura(Aura{
		Label:      "Crossbow Skill Specialization",
		BuildPhase: CharacterBuildPhaseGear,
		Duration:   NeverExpires,
		OnGain: func(aura *Aura, sim *Simulation) {
			character.PseudoStats.CrossbowsSkill += 5
		},
	})
}

func (character *Character) ThrownSpecializationAura() *Aura {
	return character.GetOrRegisterAura(Aura{
		Label:      "Thrown Skill Specialization",
		BuildPhase: CharacterBuildPhaseGear,
		Duration:   NeverExpires,
		OnGain: func(aura *Aura, sim *Simulation) {
			character.PseudoStats.ThrownSkill += 5
		},
	})
}

func (character *Character) FeralCombatSpecializationAura() *Aura {
	return character.GetOrRegisterAura(Aura{
		Label:      "Feral Combat Skill Specialization",
		BuildPhase: CharacterBuildPhaseGear,
		Duration:   NeverExpires,
		OnGain: func(aura *Aura, sim *Simulation) {
			character.PseudoStats.FeralCombatSkill += 5
		},
	})
}
