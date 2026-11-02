package core

import (
	"fmt"
	"time"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// Registers all consume-related effects to the Agent.
// TODO: Classic Consumes
func applyConsumeEffects(agent Agent) {
	character := agent.GetCharacter()
	consumes := character.Consumes

	if consumes == nil {
		return
	}

	applyFlaskConsumes(character, consumes)
	applyWeaponImbueConsumes(character, consumes)
	applyFoodConsumes(character, consumes)
	applyDefensiveBuffConsumes(character, consumes)
	applyPhysicalBuffConsumes(character, consumes)
	applySpellBuffConsumes(character, consumes)
	applyZanzaBuffConsumes(character, consumes)
	applyHitConsumableConsumes(character, consumes)
	applyMiscConsumes(character, consumes.MiscConsumes)

	registerPotionCD(agent, consumes)
	registerConjuredCD(agent, consumes)
	registerExplosivesCD(agent, consumes)
}

func ApplyPetConsumeEffects(pet *Character, ownerConsumes *proto.Consumes) {
	pet.AddStat(stats.AttackPower, []float64{0, 40}[ownerConsumes.PetAttackPowerConsumable])
	pet.AddStat(stats.Agility, []float64{0, 17, 13, 9, 5}[ownerConsumes.PetAgilityConsumable])
	pet.AddStat(stats.Strength, []float64{0, 30, 17, 13, 9, 5}[ownerConsumes.PetStrengthConsumable])
}

///////////////////////////////////////////////////////////////////////////
//                             Flasks
///////////////////////////////////////////////////////////////////////////

func applyFlaskConsumes(character *Character, consumes *proto.Consumes) {
	if consumes.Flask == proto.Flask_FlaskUnknown {
		return
	}

	switch consumes.Flask {
	case proto.Flask_FlaskOfDistilledWisdom:
		character.AddStats(stats.Stats{
			stats.Mana: 2000,
		})
	case proto.Flask_FlaskOfSupremePower:
		character.AddStats(stats.Stats{
			stats.SpellPower: 150,
		})
	case proto.Flask_FlaskOfTheTitans:
		character.AddStats(stats.Stats{
			stats.Health: 1200,
		})
	case proto.Flask_FlaskOfChromaticResistance:
		character.AddResistances(25)
	}
}

///////////////////////////////////////////////////////////////////////////
//                             Weapon Imbues
///////////////////////////////////////////////////////////////////////////

func applyWeaponImbueConsumes(character *Character, consumes *proto.Consumes) {
	// There must be a nicer way to do this...
	shadowOilIcd := Cooldown{
		Timer:    character.NewTimer(),
		Duration: time.Second * 10,
	}

	if character.HasMHWeapon() {
		addImbueStats(character, consumes.MainHandImbue, true, shadowOilIcd)
		addImbueStats(character, stoneBesidePoison(consumes.MainHandImbue, consumes.MainHandStone), true, shadowOilIcd)
	}
	if character.OffHand() != nil {
		addImbueStats(character, consumes.OffHandImbue, false, shadowOilIcd)
		addImbueStats(character, stoneBesidePoison(consumes.OffHandImbue, consumes.OffHandStone), false, shadowOilIcd)
	}
}

// roguePoisons are the imbues Forever lets share a weapon with a stone.
var roguePoisons = map[proto.WeaponImbue]bool{
	proto.WeaponImbue_InstantPoison: true,
	proto.WeaponImbue_DeadlyPoison:  true,
	proto.WeaponImbue_WoundPoison:   true,
}

// itemTemporaryEnchants are the stones, weightstones and oils: the
// temporary enchants a consumable item puts on a weapon.
var itemTemporaryEnchants = map[proto.WeaponImbue]bool{
	proto.WeaponImbue_MinorWizardOil: true, proto.WeaponImbue_LesserWizardOil: true,
	proto.WeaponImbue_WizardOil: true, proto.WeaponImbue_BrilliantWizardOil: true,
	proto.WeaponImbue_BlessedWizardOil: true, proto.WeaponImbue_MinorManaOil: true,
	proto.WeaponImbue_LesserManaOil: true, proto.WeaponImbue_BrilliantManaOil: true,
	proto.WeaponImbue_SolidSharpeningStone: true, proto.WeaponImbue_DenseSharpeningStone: true,
	proto.WeaponImbue_ElementalSharpeningStone: true, proto.WeaponImbue_ConsecratedSharpeningStone: true,
	proto.WeaponImbue_SolidWeightstone: true, proto.WeaponImbue_DenseWeightstone: true,
	proto.WeaponImbue_ShadowOil: true, proto.WeaponImbue_FrostOil: true,
}

// stoneBesidePoison is the second temporary enchant on a hand: Forever
// lets a stone, weightstone or oil sit beside a rogue poison (the rogue
// and warlock Deep Dive, 2026-10-07). The stone field must hold such an
// item enchant, and the hand's imbue must be a poison (or empty, when it
// is the hand's only enchant); anything else would put two enchants of
// one kind on a weapon and is refused.
func stoneBesidePoison(imbue, stone proto.WeaponImbue) proto.WeaponImbue {
	if stone == proto.WeaponImbue_WeaponImbueUnknown {
		return stone
	}
	if !itemTemporaryEnchants[stone] {
		panic(fmt.Sprintf("weapon stone field holds %s: it takes a stone, weightstone or oil", stone))
	}
	if imbue != proto.WeaponImbue_WeaponImbueUnknown && !roguePoisons[imbue] {
		panic(fmt.Sprintf("weapon stone %s sits beside %s: only a rogue poison shares a weapon with a stone", stone, imbue))
	}
	return stone
}

// manaOilStats are the mana oils' mana per five seconds and healing, as the
// client's item tooltips state them (items 20745, 20747 and 20748, whose
// enchants carry spells 25114, 25115 and 25116): Forever raised vanilla's 4/8/12 mana and
// 8/16/25 healing.
var manaOilStats = map[proto.WeaponImbue]stats.Stats{
	proto.WeaponImbue_MinorManaOil:     {stats.MP5: 5, stats.HealingPower: 10},
	proto.WeaponImbue_LesserManaOil:    {stats.MP5: 10, stats.HealingPower: 20},
	proto.WeaponImbue_BrilliantManaOil: {stats.MP5: 15, stats.HealingPower: 30},
}

// wizardOilStats are the wizard oils' enchant spells (25109, 25110, 25111,
// 25113): aura 13 and aura 135, damage and healing alike, and Brilliant
// Wizard Oil's aura 57, 1% spell crit (spell table only).
var wizardOilStats = map[proto.WeaponImbue]stats.Stats{
	proto.WeaponImbue_MinorWizardOil:     {stats.SpellPower: 8, stats.HealingPower: 8},
	proto.WeaponImbue_LesserWizardOil:    {stats.SpellPower: 16, stats.HealingPower: 16},
	proto.WeaponImbue_WizardOil:          {stats.SpellPower: 24, stats.HealingPower: 24},
	proto.WeaponImbue_BrilliantWizardOil: {stats.SpellPower: 36, stats.HealingPower: 36, stats.SpellCrit: 1 * CritRatingPerCritChance},
}

// addFlatWeaponDamage is a stone's or weightstone's flat damage on the
// weapon it sits on. Forever: "Weapon enchantments and procs now all work
// while shapeshifted, including effects like weightstones" (the druid deep
// dive, news.blizzard.com article 24301515), and a form's auto attack has
// "the same DPS as the Druid's equipped weapon". So on a feral druid the
// stone is the weapon's own damage per swing, at the weapon's speed: it
// raises the weapon's damage per second, which the form's paw is built
// from when it shifts.
func addFlatWeaponDamage(character *Character, isMh bool, damage float64) {
	if character.PseudoStats.FeralCombatEnabled {
		if weapon := character.GetMHWeapon(); isMh && weapon != nil && weapon.SwingSpeed > 0 {
			character.PseudoStats.BonusMHDps += damage / weapon.SwingSpeed
		}
		return
	}
	weapon := character.AutoAttacks.MH()
	if !isMh {
		weapon = character.AutoAttacks.OH()
	}
	weapon.BaseDamageMin += damage
	weapon.BaseDamageMax += damage
}

func addImbueStats(character *Character, imbue proto.WeaponImbue, isMh bool, shadowOilIcd Cooldown) {
	if imbue != proto.WeaponImbue_WeaponImbueUnknown {
		switch imbue {
		// Wizard Oils
		case proto.WeaponImbue_MinorWizardOil, proto.WeaponImbue_LesserWizardOil,
			proto.WeaponImbue_WizardOil, proto.WeaponImbue_BrilliantWizardOil:
			character.AddStats(wizardOilStats[imbue])
		case proto.WeaponImbue_BlessedWizardOil:
			character.Env.RegisterPostFinalizeEffect(func() {
				// The pool, not the active prefix: an imbue is set up
				// once and means every undead in the fight, including
				// the ones a target timeline brings in later.
				for _, target := range character.Env.Encounter.AllTargetUnits {
					if target.MobType != proto.MobType_MobTypeUndead {
						continue
					}

					for _, at := range character.AttackTables[target.UnitIndex] {
						at.BonusSpellDamageTaken += 60
					}
				}
			})

		// Mana Oils
		case proto.WeaponImbue_MinorManaOil, proto.WeaponImbue_LesserManaOil, proto.WeaponImbue_BrilliantManaOil:
			character.AddStats(manaOilStats[imbue])

		// Sharpening Stones
		case proto.WeaponImbue_SolidSharpeningStone:
			addFlatWeaponDamage(character, isMh, 6)
		case proto.WeaponImbue_DenseSharpeningStone:
			addFlatWeaponDamage(character, isMh, 8)
		case proto.WeaponImbue_ElementalSharpeningStone:
			// Enchant 2506, spell 22755: aura 52 with SpellEquippedItems
			// mask 42483 (melee weapons), so a weapon-dependent crit: the
			// character's own melee attacks only. Never a spell, never a
			// ranged attack, and never a pet (a hunter pet inherits its
			// owner's crit stats, not the owner's weapon). It works in a
			// feral form too: "Weapon enchantments and procs now all work
			// while shapeshifted" (the druid deep dive).
			addMeleeWeaponCrit(&character.Unit, 2*CritRatingPerCritChance)
		case proto.WeaponImbue_ConsecratedSharpeningStone:
			character.Env.RegisterPostFinalizeEffect(func() {
				// The pool, not the active prefix; see
				// BlessedWizardOil above.
				for _, target := range character.Env.Encounter.AllTargetUnits {
					if target.MobType != proto.MobType_MobTypeUndead {
						continue
					}

					for _, at := range character.AttackTables[target.UnitIndex] {
						at.BonusAttackPowerTaken += 100
					}
				}
			})

		// Weightstones
		case proto.WeaponImbue_SolidWeightstone:
			addFlatWeaponDamage(character, isMh, 6)
		case proto.WeaponImbue_DenseWeightstone:
			addFlatWeaponDamage(character, isMh, 8)
		// Windfury
		case proto.WeaponImbue_Windfury:
			if !character.PseudoStats.FeralCombatEnabled {
				ApplyWindfury(character)
			}
		case proto.WeaponImbue_ShadowOil:
			registerShadowOil(character, isMh, shadowOilIcd)
		case proto.WeaponImbue_FrostOil:
			registerFrostOil(character, isMh)
		}
	}
}

func registerShadowOil(character *Character, isMh bool, icd Cooldown) {
	procChance := 0.15

	procSpell := character.GetOrRegisterSpell(SpellConfig{
		ActionID:    ActionID{SpellID: 1382},
		SpellSchool: SpellSchoolShadow,
		DefenseType: DefenseTypeMagic,
		ProcMask:    ProcMaskEmpty,
		Flags:       SpellFlagNoOnCastComplete | SpellFlagPassiveSpell,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: 0.56,

		ApplyEffects: func(sim *Simulation, target *Unit, spell *Spell) {
			damage := sim.Roll(52, 61)
			spell.CalcAndDealDamage(sim, target, damage, spell.OutcomeMagicHitAndCrit)
		},
	})

	label := " MH"
	procMask := ProcMaskMeleeMH
	if !isMh {
		label = " OH"
		procMask = ProcMaskMeleeOH
	}

	MakePermanent(character.GetOrRegisterAura(Aura{
		Label: "Shadow Oil" + label,
		OnSpellHitDealt: func(aura *Aura, sim *Simulation, spell *Spell, result *SpellResult) {
			if !result.Landed() {
				return
			}

			if !spell.ProcMask.Matches(procMask) {
				return
			}

			if !icd.IsReady(sim) {
				return
			}

			if sim.RandomFloat("Shadow Oil") < procChance {
				icd.Use(sim)
				procSpell.Cast(sim, result.Target)
			}
		},
	}))
}

func registerFrostOil(character *Character, isMh bool) {
	procChance := 0.10

	procSpell := character.GetOrRegisterSpell(SpellConfig{
		ActionID:    ActionID{SpellID: 1191},
		SpellSchool: SpellSchoolFrost,
		DefenseType: DefenseTypeMagic,
		ProcMask:    ProcMaskEmpty,
		Flags:       SpellFlagNoOnCastComplete | SpellFlagPassiveSpell,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: 0.269,

		ApplyEffects: func(sim *Simulation, target *Unit, spell *Spell) {
			damage := sim.Roll(33, 38)
			spell.CalcAndDealDamage(sim, target, damage, spell.OutcomeMagicHitAndCrit)
		},
	})

	label := " MH"
	procMask := ProcMaskMeleeMHAuto
	if !isMh {
		label = " OH"
		procMask = ProcMaskMeleeOHAuto
	}

	MakePermanent(character.GetOrRegisterAura(Aura{
		Label: "Frost Oil" + label,
		OnSpellHitDealt: func(aura *Aura, sim *Simulation, spell *Spell, result *SpellResult) {
			if !result.Landed() {
				return
			}

			if !spell.ProcMask.Matches(procMask) {
				return
			}

			if sim.RandomFloat("Frost Oil") < procChance {
				procSpell.Cast(sim, result.Target)
			}
		},
	}))
}

///////////////////////////////////////////////////////////////////////////
//                             Food
///////////////////////////////////////////////////////////////////////////

// nightfinSoupStats is what Nightfin Soup gives in this client: 22 spell
// damage for the well-fed hour (item 13931, spell 1249513). Forever turned
// the food into a stat meal and dropped vanilla's 8 mana per five seconds,
// so it gives a healer nothing.
var nightfinSoupStats = stats.Stats{stats.SpellDamage: 22}

// sagesTeaStats is what Sage's Tea gives in this client: 44 healing for the
// well-fed quarter hour (item 249870, spell 1249925, aura 227 effect of 44
// triggering Well Fed 1249927, aura 135).
var sagesTeaStats = stats.Stats{stats.HealingPower: 44}

// foodStats is each food's well-fed buff in this client. Forever's foods
// are "Nutritious Food": an aura 227 effect passes its value to the Well
// Fed it triggers, whose aura names the stat (1248420 Agility, 1248421
// Intellect, 1248422 Strength, 1249520 spell damage, 1249523 critical
// strike), so several differ from Era: Grilled Squid (13928) is 1% crit,
// Tender Wolf Steak (18045) and Hot Wolf Ribs (13851) Agility, Runn Tum
// Tuber Surprise (18254) 15 Intellect, the two sagefish spell damage.
// Dirge's Chops and the Blessed Sunfruits keep their own Well Fed spells
// (25661, 18141, 18125).
var foodStats = map[proto.Food]stats.Stats{
	proto.Food_FoodHotWolfRibs:              {stats.Agility: 10},
	proto.Food_FoodSmokedSagefish:           {stats.SpellDamage: 4},
	proto.Food_FoodSagefishDelight:          {stats.SpellDamage: 7},
	proto.Food_FoodTenderWolfSteak:          {stats.Agility: 15},
	proto.Food_FoodGrilledSquid:             {stats.Crit: 1 * CritRatingPerCritChance},
	proto.Food_FoodSmokedDesertDumpling:     {stats.Strength: 20},
	proto.Food_FoodNightfinSoup:             nightfinSoupStats,
	proto.Food_FoodSagesTea:                 sagesTeaStats,
	proto.Food_FoodRunnTumTuberSurprise:     {stats.Intellect: 15},
	proto.Food_FoodDirgesKickChimaerokChops: {stats.Stamina: 25},
	proto.Food_FoodBlessedSunfruitJuice:     {stats.Spirit: 10},
	proto.Food_FoodBlessSunfruit:            {stats.Strength: 10},
	// Forever's level-45 spell damage food (Baked Salmon 13935; Lobster
	// Stew 13933 is the same Well Fed) and its 40 attack power food from
	// level 35 (Poached Sunscale Salmon 13932; Mightfish Steak 13934 is
	// the same at 45): Well Feds 1249520 (aura 13) and 1249519 (aura 99
	// and aura 124).
	proto.Food_FoodBakedSalmon:           {stats.SpellDamage: 28},
	proto.Food_FoodPoachedSunscaleSalmon: {stats.AttackPower: 40, stats.RangedAttackPower: 40},
}

func applyFoodConsumes(character *Character, consumes *proto.Consumes) {
	character.AddStats(foodStats[consumes.Food])

	if consumes.Alcohol != proto.Alcohol_AlcoholUnknown {
		switch consumes.Alcohol {
		case proto.Alcohol_AlcoholRumseyRumBlackLabel:
			character.AddStats(stats.Stats{
				stats.Stamina: 15,
			})
		case proto.Alcohol_AlcoholGordokGreenGrog:
			character.AddStats(stats.Stats{
				stats.Stamina: 10,
			})
		case proto.Alcohol_AlcoholRumseyRumDark:
			character.AddStats(stats.Stats{
				stats.Stamina: 10,
			})
		case proto.Alcohol_AlcoholRumseyRumLight:
			character.AddStats(stats.Stats{
				stats.Stamina: 5,
			})
		case proto.Alcohol_AlcoholKreegsStoutBeatdown:
			character.AddStats(stats.Stats{
				stats.Spirit:    25,
				stats.Intellect: -5,
			})
		}
	}

	if consumes.DragonBreathChili {
		MakePermanent(DragonBreathChiliAura(character))
	}
}

func DragonBreathChiliAura(character *Character) *Aura {
	baseDamage := 60.0
	procChance := .05
	icd := Cooldown{
		Timer:    character.NewTimer(),
		Duration: time.Second * 10,
	}

	procSpell := character.RegisterSpell(SpellConfig{
		ActionID:    ActionID{SpellID: 15851},
		SpellSchool: SpellSchoolFire,
		DefenseType: DefenseTypeMagic,
		ProcMask:    ProcMaskSpellDamageProc | ProcMaskSpellProc,
		Flags:       SpellFlagNoOnCastComplete | SpellFlagPassiveSpell,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *Simulation, target *Unit, spell *Spell) {
			for _, aoeTarget := range sim.Environment.Encounter.TargetUnits {
				spell.CalcAndDealDamage(sim, aoeTarget, baseDamage, spell.OutcomeMagicHitAndCrit)
			}
		},
	})

	aura := character.GetOrRegisterAura(Aura{
		Label:    "Dragonbreath Chili",
		ActionID: ActionID{SpellID: 15852},
		Duration: NeverExpires,
		OnSpellHitDealt: func(aura *Aura, sim *Simulation, spell *Spell, result *SpellResult) {
			if !result.Landed() || !spell.ProcMask.Matches(ProcMaskMelee) {
				return
			}
			if icd.IsReady(sim) && sim.RandomFloat("Dragonbreath Chili") < procChance {
				icd.Use(sim)
				procSpell.Cast(sim, result.Target)
			}
		},
	})
	return aura
}

///////////////////////////////////////////////////////////////////////////
//                             Defensive Buff Consumes
///////////////////////////////////////////////////////////////////////////

// armorElixirStats and healthElixirStats are the client's amounts: Elixir
// of Minor Defense 5997 (50), Elixir of Defense 8951 (250, Era's 150),
// Elixir of Greater Defense 13445 (450, Era's 250; aura 674 armor), Elixir
// of Minor Fortitude 2458 (27) and Forever's Elixir of Fortitude 250334
// (200, Era's 3825 gave 120; aura 34 health). Elixir of Superior Defense is
// not in this client's item table.
var (
	armorElixirStats = map[proto.ArmorElixir]stats.Stats{
		proto.ArmorElixir_ElixirOfSuperiorDefense: {stats.BonusArmor: 450},
		proto.ArmorElixir_ElixirOfGreaterDefense:  {stats.BonusArmor: 450},
		proto.ArmorElixir_ElixirOfDefense:         {stats.BonusArmor: 250},
		proto.ArmorElixir_ElixirOfMinorDefense:    {stats.BonusArmor: 50},
		proto.ArmorElixir_ScrollOfProtection:      BuffSpellValues[ScrollOfProtection],
	}
	healthElixirStats = map[proto.HealthElixir]stats.Stats{
		proto.HealthElixir_ElixirOfFortitude:      {stats.Health: 200},
		proto.HealthElixir_ElixirOfMinorFortitude: {stats.Health: 27},
	}
)

func applyDefensiveBuffConsumes(character *Character, consumes *proto.Consumes) {
	character.AddStats(armorElixirStats[consumes.ArmorElixir])
	character.AddStats(healthElixirStats[consumes.HealthElixir])
}

///////////////////////////////////////////////////////////////////////////
//                             Physical Buff Consumes
///////////////////////////////////////////////////////////////////////////

func applyPhysicalBuffConsumes(character *Character, consumes *proto.Consumes) {
	if consumes.AttackPowerBuff != proto.AttackPowerBuff_AttackPowerBuffUnknown {
		switch consumes.AttackPowerBuff {
		case proto.AttackPowerBuff_JujuMight:
			character.AddStats(stats.Stats{
				stats.AttackPower:       40,
				stats.RangedAttackPower: 40,
			})
		case proto.AttackPowerBuff_WinterfallFirewater:
			character.AddStats(stats.Stats{
				stats.AttackPower: 35,
			})
		}
	}

	if consumes.AgilityElixir != proto.AgilityElixir_AgilityElixirUnknown {
		switch consumes.AgilityElixir {
		case proto.AgilityElixir_ElixirOfTheMongoose:
			character.AddStats(stats.Stats{
				stats.Agility: 25,
				stats.Crit:    2 * CritRatingPerCritChance,
			})
		case proto.AgilityElixir_ElixirOfGreaterAgility:
			character.AddStats(stats.Stats{
				stats.Agility: 25,
			})
		case proto.AgilityElixir_ElixirOfAgility:
			character.AddStats(stats.Stats{
				stats.Agility: 15,
			})
		case proto.AgilityElixir_ElixirOfLesserAgility:
			character.AddStats(stats.Stats{
				stats.Agility: 8,
			})
		case proto.AgilityElixir_ScrollOfAgility:
			character.AddStats(BuffSpellValues[ScrollOfAgility])
		}
	}

	if consumes.StrengthBuff != proto.StrengthBuff_StrengthBuffUnknown {
		switch consumes.StrengthBuff {
		case proto.StrengthBuff_JujuPower:
			character.AddStats(stats.Stats{
				stats.Strength: 30,
			})
		case proto.StrengthBuff_ElixirOfGiants:
			character.AddStats(stats.Stats{
				stats.Strength: 25,
			})
		case proto.StrengthBuff_ElixirOfOgresStrength:
			character.AddStats(stats.Stats{
				stats.Strength: 8,
			})
		case proto.StrengthBuff_ScrollOfStrength:
			character.AddStats(BuffSpellValues[ScrollOfStrength])
		}
	}
}

///////////////////////////////////////////////////////////////////////////
//                             Spell Buff Consumes
///////////////////////////////////////////////////////////////////////////

func applySpellBuffConsumes(character *Character, consumes *proto.Consumes) {
	if consumes.SpellPowerBuff != proto.SpellPowerBuff_SpellPowerBuffUnknown {
		switch consumes.SpellPowerBuff {
		case proto.SpellPowerBuff_ArcaneElixir:
			character.AddStats(stats.Stats{
				stats.SpellDamage: 20,
			})
		case proto.SpellPowerBuff_GreaterArcaneElixir:
			character.AddStats(stats.Stats{
				stats.SpellDamage: 35,
			})
		}
	}

	if consumes.FirePowerBuff != proto.FirePowerBuff_FirePowerBuffUnknown {
		switch consumes.FirePowerBuff {
		case proto.FirePowerBuff_ElixirOfFirepower:
			character.AddStats(stats.Stats{
				stats.FirePower: 10,
			})
		case proto.FirePowerBuff_ElixirOfGreaterFirepower:
			character.AddStats(stats.Stats{
				stats.FirePower: 40,
			})
		}
	}

	if consumes.ShadowPowerBuff != proto.ShadowPowerBuff_ShadowPowerBuffUnknown {
		switch consumes.ShadowPowerBuff {
		case proto.ShadowPowerBuff_ElixirOfShadowPower:
			character.AddStats(stats.Stats{
				stats.ShadowPower: 40,
			})
		}
	}

	if consumes.FrostPowerBuff != proto.FrostPowerBuff_FrostPowerBuffUnknown {
		switch consumes.FrostPowerBuff {
		case proto.FrostPowerBuff_ElixirOfFrostPower:
			character.AddStats(stats.Stats{
				stats.FrostPower: 15,
			})
		}
	}

	character.AddStats(manaRegenElixirStats[consumes.ManaRegenElixir])
	character.AddStats(healingPowerBuffStats[consumes.HealingPowerBuff])
	character.AddStats(spiritElixirStats[consumes.SpiritElixir])
}

// The healer elixirs, as the client states them. Mageblood Elixir is item
// 20007 (spell 24363, aura 85 of 12); Greater Mageblood Elixir is item
// 250341 (spell 1250948, aura 85 of 20). Cleric's Elixir (250332, spell
// 1250925) and Greater Cleric's Elixir (250333, spell 1250926) are aura 135
// healing of 30 and 40. Elixir of Sages (250338, spell 1250941) is aura 29
// spirit of 25 and aura 290 critical strike of 2 percent.
var (
	manaRegenElixirStats = map[proto.ManaRegenElixir]stats.Stats{
		proto.ManaRegenElixir_MagebloodPotion:        {stats.MP5: 12},
		proto.ManaRegenElixir_GreaterMagebloodElixir: {stats.MP5: 20},
	}
	healingPowerBuffStats = map[proto.HealingPowerBuff]stats.Stats{
		proto.HealingPowerBuff_ClericsElixir:        {stats.HealingPower: 30},
		proto.HealingPowerBuff_GreaterClericsElixir: {stats.HealingPower: 40},
	}
	spiritElixirStats = map[proto.SpiritElixir]stats.Stats{
		proto.SpiritElixir_ElixirOfSages: {stats.Spirit: 25, stats.Crit: 2 * CritRatingPerCritChance},
	}
)

///////////////////////////////////////////////////////////////////////////
//                             Zanza-esque Consumes
///////////////////////////////////////////////////////////////////////////

func applyZanzaBuffConsumes(character *Character, consumes *proto.Consumes) {
	if consumes.ZanzaBuff == proto.ZanzaBuff_ZanzaBuffUnknown {
		return
	}

	switch consumes.ZanzaBuff {
	case proto.ZanzaBuff_SpiritOfZanza:
		character.AddStats(stats.Stats{
			stats.Stamina: 50,
			stats.Spirit:  50,
		})
	case proto.ZanzaBuff_ROIDS:
		character.AddStats(stats.Stats{
			stats.Strength: 25,
		})
	case proto.ZanzaBuff_GroundScorpokAssay:
		character.AddStats(stats.Stats{
			stats.Agility: 25,
		})
	case proto.ZanzaBuff_CerebralCortexCompound:
		character.AddStats(stats.Stats{
			stats.Intellect: 25,
		})
	case proto.ZanzaBuff_GizzardGum:
		character.AddStats(stats.Stats{
			stats.Spirit: 25,
		})
	case proto.ZanzaBuff_LungJuiceCocktail:
		character.AddStats(stats.Stats{
			stats.Stamina: 25,
		})
	case proto.ZanzaBuff_DarnassusGiftCollection:
		character.AddStats(stats.Stats{
			stats.Agility: 30,
		})
	}
}

///////////////////////////////////////////////////////////////////////////
//                             Hit Consumables
///////////////////////////////////////////////////////////////////////////

func applyHitConsumableConsumes(character *Character, consumes *proto.Consumes) {
	if consumes.HitConsumable == proto.HitConsumable_HitConsumableUnknown {
		return
	}

	switch consumes.HitConsumable {
	case proto.HitConsumable_FireToastedBun:
		character.AddStats(stats.Stats{
			stats.Hit: 2 * HitRatingPerHitChance,
		})
	case proto.HitConsumable_DarkDesire:
		character.AddStats(stats.Stats{
			stats.Hit: 2 * HitRatingPerHitChance,
		})
	}
}

///////////////////////////////////////////////////////////////////////////
//                             Misc Consumes
///////////////////////////////////////////////////////////////////////////

func applyMiscConsumes(character *Character, miscConsumes *proto.MiscConsumes) {
	if miscConsumes == nil {
		return
	}

	if miscConsumes.BoglingRoot {
		character.PseudoStats.BonusPhysicalDamage += 1
	}

	if miscConsumes.RaptorPunch {
		character.AddStats(stats.Stats{
			stats.Intellect: 4,
			stats.Stamina:   -5,
		})
	}

	if miscConsumes.JujuEmber {
		character.AddStat(stats.FireResistance, 15)
	}

	if miscConsumes.JujuChill {
		character.AddStat(stats.FrostResistance, 15)
	}

	if miscConsumes.JujuFlurry && character.AutoAttacks.enabled {
		actionID := ActionID{SpellID: 16322}
		// In Vanilla Juju Flurry was bugged to act like Seal of the Crusader where it gave attack speed but also reduced damage done.
		jujuFlurryAura := character.RegisterAura(Aura{
			Label:    "Juju Flurry",
			ActionID: actionID,
			Duration: time.Second * 20,
			OnGain: func(aura *Aura, sim *Simulation) {
				aura.Unit.MultiplyMeleeSpeed(sim, 1.03)
				aura.Unit.AutoAttacks.MHAuto().DamageMultiplier /= 1.03
				aura.Unit.AutoAttacks.OHAuto().DamageMultiplier /= 1.03
			},
			OnExpire: func(aura *Aura, sim *Simulation) {
				aura.Unit.MultiplyMeleeSpeed(sim, 1/1.03)
				aura.Unit.AutoAttacks.MHAuto().DamageMultiplier *= 1.03
				aura.Unit.AutoAttacks.OHAuto().DamageMultiplier *= 1.03
			},
		})
		jujuFlurrySpell := character.RegisterSpell(SpellConfig{
			ActionID: actionID,
			ProcMask: ProcMaskEmpty,
			Cast: CastConfig{
				CD: Cooldown{
					Timer:    character.NewTimer(),
					Duration: time.Minute,
				},
				SharedCD: Cooldown{
					Timer:    character.GetAttackSpeedBuffCD(),
					Duration: time.Second * 10,
				},
			},
			ApplyEffects: func(sim *Simulation, target *Unit, spell *Spell) {
				jujuFlurryAura.Activate(sim)
			},
		})
		character.AddMajorCooldown(MajorCooldown{
			Spell:    jujuFlurrySpell,
			Type:     CooldownTypeDPS,
			Priority: CooldownPriorityDefault,
		})
	}

	if miscConsumes.JujuEscape {
		actionID := ActionID{SpellID: 16321}
		jujuEscapeAura := character.RegisterAura(Aura{
			Label:    "Juju Escape",
			ActionID: actionID,
			Duration: time.Second * 10,
			OnGain: func(aura *Aura, sim *Simulation) {
				aura.Unit.AddStatDynamic(sim, stats.Dodge, 5*DodgeRatingPerDodgeChance)
			},
			OnExpire: func(aura *Aura, sim *Simulation) {
				aura.Unit.AddStatDynamic(sim, stats.Dodge, -5*DodgeRatingPerDodgeChance)
			},
		})
		jujuEscapeSpell := character.RegisterSpell(SpellConfig{
			ActionID: actionID,
			ProcMask: ProcMaskEmpty,
			Cast: CastConfig{
				CD: Cooldown{
					Timer:    character.NewTimer(),
					Duration: time.Minute,
				},
			},
			ApplyEffects: func(sim *Simulation, target *Unit, spell *Spell) {
				jujuEscapeAura.Activate(sim)
			},
		})
		character.AddMajorCooldown(MajorCooldown{
			Spell:    jujuEscapeSpell,
			Type:     CooldownTypeSurvival,
			Priority: CooldownPriorityDefault,
		})
	}
}

///////////////////////////////////////////////////////////////////////////
//                             Engineering Explosives
///////////////////////////////////////////////////////////////////////////

var SapperActionID = ActionID{ItemID: 10646}
var SolidDynamiteActionID = ActionID{ItemID: 10507}
var DenseDynamiteActionID = ActionID{ItemID: 18641}
var ThoriumGrenadeActionID = ActionID{ItemID: 15993}
var GoblinLandMineActionID = ActionID{ItemID: 4395}

func registerExplosivesCD(agent Agent, consumes *proto.Consumes) {
	character := agent.GetCharacter()
	hasFiller := consumes.FillerExplosive != proto.Explosive_ExplosiveUnknown
	hasSapper := consumes.SapperExplosive != proto.SapperExplosive_SapperUnknown

	if !hasSapper && !hasFiller {
		return
	}
	sharedTimer := character.NewTimer()

	if hasSapper {
		if !character.HasProfession(proto.Profession_Engineering) {
			return
		}
		var filler *Spell
		switch consumes.SapperExplosive {
		case proto.SapperExplosive_SapperGoblinSapper:
			filler = character.newSapperSpell(sharedTimer)
		}
		character.AddMajorCooldown(MajorCooldown{
			Spell:    filler,
			Type:     CooldownTypeDPS | CooldownTypeExplosive,
			Priority: CooldownPriorityLow + 30,
			ShouldActivate: func(s *Simulation, c *Character) bool {
				return !character.IsShapeshifted()
			},
		})
	}

	if hasFiller {
		if !character.HasProfession(proto.Profession_Engineering) {
			return
		}

		var filler *Spell
		switch consumes.FillerExplosive {
		case proto.Explosive_ExplosiveSolidDynamite:
			filler = character.newSolidDynamiteSpell(sharedTimer)
		case proto.Explosive_ExplosiveDenseDynamite:
			filler = character.newDenseDynamiteSpell(sharedTimer)
		case proto.Explosive_ExplosiveThoriumGrenade:
			filler = character.newThoriumGrenadeSpell(sharedTimer)
		case proto.Explosive_ExplosiveGoblinLandMine:
			filler = character.newGoblinLandMineSpell(sharedTimer)
		}

		character.AddMajorCooldown(MajorCooldown{
			Spell:    filler,
			Type:     CooldownTypeDPS | CooldownTypeExplosive,
			Priority: CooldownPriorityLow + 10,
			ShouldActivate: func(s *Simulation, c *Character) bool {
				return !character.IsShapeshifted()
			},
		})
	}
}

// Creates a spell object for the common explosive case.
// TODO: create 10s delay on Goblin Landmine cast to damage
func (character *Character) newBasicExplosiveSpellConfig(sharedTimer *Timer, actionID ActionID, school SpellSchool, minDamage float64, maxDamage float64, cooldown Cooldown, selfMinDamage float64, selfMaxDamage float64) SpellConfig {
	isSapper := actionID.SameAction(SapperActionID)

	var defaultCast Cast
	if !isSapper {
		defaultCast = Cast{
			CastTime: time.Second,
		}
	}

	return SpellConfig{
		ActionID:    actionID,
		SpellSchool: school,
		DefenseType: DefenseTypeMagic,
		ProcMask:    ProcMaskEmpty,
		Flags:       SpellFlagCastTimeNoGCD,

		Cast: CastConfig{
			DefaultCast: defaultCast,
			CD:          cooldown,
			IgnoreHaste: true,
			SharedCD: Cooldown{
				Timer:    sharedTimer,
				Duration: time.Minute,
			},
			ModifyCast: func(sim *Simulation, _ *Spell, _ *Cast) {
				character.CancelShapeshift(sim)
			},
		},

		// Explosives always have 1% resist chance, so just give them hit cap.
		BonusHitRating: 100 * HitRatingPerHitChance,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *Simulation, target *Unit, spell *Spell) {
			for _, aoeTarget := range sim.Encounter.TargetUnits {
				baseDamage := sim.Roll(minDamage, maxDamage) * sim.Encounter.AOECapMultiplier()
				spell.CalcAndDealDamage(sim, aoeTarget, baseDamage, spell.OutcomeMagicHitAndCrit)
			}

			if isSapper {
				baseDamage := sim.Roll(selfMinDamage, selfMaxDamage)
				spell.CalcAndDealDamage(sim, &character.Unit, baseDamage, spell.OutcomeMagicHitAndCrit)
			}
		},
	}
}
func (character *Character) newSapperSpell(sharedTimer *Timer) *Spell {
	return character.GetOrRegisterSpell(character.newBasicExplosiveSpellConfig(sharedTimer, SapperActionID, SpellSchoolFire, 450, 750, Cooldown{Timer: character.NewTimer(), Duration: time.Minute * 5}, 375, 625))
}

func (character *Character) newSolidDynamiteSpell(sharedTimer *Timer) *Spell {
	return character.GetOrRegisterSpell(character.newBasicExplosiveSpellConfig(sharedTimer, SolidDynamiteActionID, SpellSchoolFire, 213, 287, Cooldown{}, 0, 0))
}
func (character *Character) newDenseDynamiteSpell(sharedTimer *Timer) *Spell {
	return character.GetOrRegisterSpell(character.newBasicExplosiveSpellConfig(sharedTimer, DenseDynamiteActionID, SpellSchoolFire, 340, 460, Cooldown{}, 0, 0))
}
func (character *Character) newThoriumGrenadeSpell(sharedTimer *Timer) *Spell {
	return character.GetOrRegisterSpell(character.newBasicExplosiveSpellConfig(sharedTimer, ThoriumGrenadeActionID, SpellSchoolFire, 300, 500, Cooldown{}, 0, 0))
}
func (character *Character) newGoblinLandMineSpell(sharedTimer *Timer) *Spell {
	return character.GetOrRegisterSpell(character.newBasicExplosiveSpellConfig(sharedTimer, GoblinLandMineActionID, SpellSchoolFire, 394, 506, Cooldown{}, 0, 0))
}

///////////////////////////////////////////////////////////////////////////
//                             Potions
///////////////////////////////////////////////////////////////////////////

// Forever's druid deep dive (news.blizzard.com article 24301515): "All
// potions and almost all similar consumables are now useable while
// shapeshifted and do not cancel the Druid's shapeshift. Rage potions and
// Thistletea are also now usable by Druids. One notable exception is
// Engineering bombs." So no potion, rune or healthstone here cancels a form
// or waits for one to end; the explosives still do.
func registerPotionCD(agent Agent, consumes *proto.Consumes) {
	character := agent.GetCharacter()
	defaultPotion := consumes.DefaultPotion

	potionCD := character.NewTimer()

	if defaultPotion == proto.Potions_UnknownPotion {
		return
	}

	defaultMCD := makePotionActivation(defaultPotion, character, potionCD)

	if defaultMCD.Spell != nil {
		character.AddMajorCooldown(defaultMCD)
	}
}

func makePotionActivation(potionType proto.Potions, character *Character, potionCD *Timer) MajorCooldown {
	mcd := makePotionActivationInternal(potionType, character, potionCD)
	if mcd.Spell != nil {
		// Mark as 'Encounter Only' so that users are forced to select the generic Potion
		// placeholder action instead of specific potion spells, in APL prepull. This
		// prevents a mismatch between Consumes and Rotation settings.
		mcd.Spell.Flags |= SpellFlagEncounterOnly | SpellFlagPotion | SpellFlagCastTimeNoGCD
		oldApplyEffects := mcd.Spell.ApplyEffects
		mcd.Spell.ApplyEffects = func(sim *Simulation, target *Unit, spell *Spell) {
			oldApplyEffects(sim, target, spell)
			if sim.CurrentTime < 0 {
				potionCD.Set(sim.CurrentTime + 2*time.Minute)
				character.UpdateMajorCooldowns()
			}
		}
	}
	return mcd
}

// SelfCastConsumableFlags are the flags of a consumable that only ever
// acts on the one who uses it (a mana or health potion, a rune, a
// healthstone). It is helpful, so the major-cooldown pass uses it whatever
// the user's current target is: a healer's current target is a friend.
const SelfCastConsumableFlags = SpellFlagNoOnCastComplete | SpellFlagHelpful

func makeHealthConsumableMCD(itemId int32, character *Character, cdTimer *Timer) MajorCooldown {
	// Using min values for healthstones as locks generally don't spec into improved
	minRoll := map[int32]float64{
		858:   140,
		929:   280,
		1710:  455,
		3928:  700,
		5509:  500,
		5510:  800,
		9421:  1200,
		13446: 1050,
	}[itemId]

	maxRoll := map[int32]float64{
		858:   180,
		929:   360,
		1710:  585,
		3928:  900,
		5509:  500,
		5510:  800,
		9421:  1200,
		13446: 1750,
	}[itemId]

	cdDuration := time.Minute * 2

	actionID := ActionID{ItemID: itemId}
	healthMetrics := character.NewHealthMetrics(actionID)

	return MajorCooldown{
		Type: CooldownTypeSurvival,
		ShouldActivate: func(sim *Simulation, character *Character) bool {
			// Only pop if we have less than the max mana provided by the potion minus 1mp5 tick.
			return (character.MaxHealth()-(character.CurrentHealth()) >= maxRoll)
		},
		Spell: character.GetOrRegisterSpell(SpellConfig{
			ActionID: actionID,
			Flags:    SelfCastConsumableFlags,
			Cast: CastConfig{
				CD: Cooldown{
					Timer:    cdTimer,
					Duration: cdDuration,
				},
			},
			ApplyEffects: func(sim *Simulation, _ *Unit, _ *Spell) {
				healthGain := sim.RollWithLabel(minRoll, maxRoll, "Health Consumable")
				character.GainHealth(sim, healthGain, healthMetrics)
			},
		}),
	}
}

// demonicRuneHealthCost is the Demonic Rune's (12662, spell 16666) cost in
// health: SpellEffect 694075 deals 800 to the user with variance 0.5, so 600
// to 1000. The mean is taken, not rolled, so a rune draws no extra random
// number and every other stream stays where it was; the health only matters
// to health gates and to heals that land on the user.
const demonicRuneHealthCost = 800.0

func makeManaConsumableMCD(itemId int32, character *Character, cdTimer *Timer) MajorCooldown {
	minRoll := map[int32]float64{
		3827:  455.0,
		6149:  700.0,
		4381:  150.0,
		12662: 900.0,
		13443: 900.0,
		13444: 1350.0,
	}[itemId]

	maxRoll := map[int32]float64{
		3827:  585.0,
		6149:  900.0,
		4381:  250.0,
		12662: 1500.0,
		13443: 1500.0,
		13444: 2250.0,
	}[itemId]

	cdDuration := time.Minute * 2

	actionID := ActionID{ItemID: itemId}
	manaMetrics := character.NewManaMetrics(actionID)

	return MajorCooldown{
		Type: CooldownTypeMana,
		ShouldActivate: func(sim *Simulation, character *Character) bool {
			// Only pop if we have less than the max mana provided by the potion minus 1mp5 tick.
			totalRegen := character.ManaRegenPerSecondWhileCasting() * 2
			return (character.MaxMana()-(character.CurrentMana()+totalRegen) >= maxRoll)
		},
		Spell: character.GetOrRegisterSpell(SpellConfig{
			ActionID: actionID,
			Flags:    SelfCastConsumableFlags,
			Cast: CastConfig{
				CD: Cooldown{
					Timer:    cdTimer,
					Duration: cdDuration,
				},
			},
			ApplyEffects: func(sim *Simulation, _ *Unit, _ *Spell) {
				manaGain := sim.RollWithLabel(minRoll, maxRoll, "Mana Consumable")
				character.AddMana(sim, manaGain, manaMetrics)
				if itemId == 12662 {
					character.RemoveHealth(sim, demonicRuneHealthCost)
				}
			},
		}),
	}
}

func makeArmorConsumableMCD(itemId int32, character *Character, cdTimer *Timer) MajorCooldown {
	actionID := ActionID{ItemID: itemId}
	cdDuration := time.Minute * 2
	lesserStoneshieldAura := character.NewTemporaryStatsAura("Lesser Stoneshield Potion", actionID, stats.Stats{stats.BonusArmor: 1000}, time.Second*90)
	greaterStoneshieldAura := character.NewTemporaryStatsAura("Greater Stoneshield Potion", actionID, stats.Stats{stats.BonusArmor: 2000}, time.Second*120)

	return MajorCooldown{
		Type: CooldownTypeSurvival,
		Spell: character.GetOrRegisterSpell(SpellConfig{
			ActionID: actionID,
			Flags:    SpellFlagNoOnCastComplete,
			Cast: CastConfig{
				CD: Cooldown{
					Timer:    cdTimer,
					Duration: cdDuration,
				},
			},
			ApplyEffects: func(sim *Simulation, _ *Unit, _ *Spell) {
				switch itemId {
				case 4623:
					lesserStoneshieldAura.Activate(sim)
				case 13455:

					greaterStoneshieldAura.Activate(sim)
				}
			},
		}),
	}
}

func makeMagicResistancePotionMCD(character *Character, cdTimer *Timer) MajorCooldown {
	actionID := ActionID{ItemID: 4623}
	cdDuration := time.Minute * 2

	stats := stats.Stats{
		stats.ArcaneResistance: 50,
		stats.FireResistance:   50,
		stats.FrostResistance:  50,
		stats.NatureResistance: 50,
		stats.ShadowResistance: 50,
	}

	// Since many people will keep this rolling as a substitute for capping Fire Resistance, show the stats as a baseline
	aura := character.NewTemporaryStatsAura("Magic Resistance Potion", actionID, stats, time.Minute*3)
	aura.BuildPhase = CharacterBuildPhaseConsumes

	return MajorCooldown{
		Type: CooldownTypeSurvival,
		Spell: character.GetOrRegisterSpell(SpellConfig{
			ActionID: actionID,
			Flags:    SpellFlagNoOnCastComplete,
			Cast: CastConfig{
				CD: Cooldown{
					Timer:    cdTimer,
					Duration: cdDuration,
				},
			},
			ApplyEffects: func(sim *Simulation, _ *Unit, _ *Spell) {
				aura.Activate(sim)
			},
		}),
	}
}

func makeRageConsumableMCD(itemId int32, character *Character, cdTimer *Timer) MajorCooldown {
	minRoll := map[int32]float64{
		5631:  20.0,
		5633:  30.0,
		13442: 45.0,
	}[itemId]

	maxRoll := map[int32]float64{
		5631:  40.0,
		5633:  60.0,
		13442: 75.0,
	}[itemId]

	cdDuration := time.Minute * 2

	actionID := ActionID{ItemID: itemId}
	rageMetrics := character.NewRageMetrics(actionID)
	aura := character.NewTemporaryStatsAura("Mighty Rage Potion", actionID, stats.Stats{stats.Strength: 60}, time.Second*20)
	return MajorCooldown{
		Type: CooldownTypeDPS,
		Spell: character.GetOrRegisterSpell(SpellConfig{
			ActionID: actionID,
			Flags:    SpellFlagNoOnCastComplete,
			Cast: CastConfig{
				CD: Cooldown{
					Timer:    cdTimer,
					Duration: cdDuration,
				},
			},
			ApplyEffects: func(sim *Simulation, _ *Unit, _ *Spell) {
				// A class with no rage bar can drink it (a request may carry
				// any consumable), and its rage has nowhere to go.
				if character.HasRageBar() {
					rageGain := sim.RollWithLabel(minRoll, maxRoll, "Rage Consumable")
					character.AddRage(sim, rageGain, rageMetrics)
				}
				if itemId == 13442 {
					aura.Activate(sim)
				}
			},
		}),
	}
}

// burstPotion is one of Forever's 30-second potions: a flat stat bonus
// for burstPotionDuration on the shared potion timer.
type burstPotion struct {
	itemID int32
	label  string
	bonus  stats.Stats
}

// burstPotionDuration is every 30-second potion's aura length (SpellMisc
// duration 30000 ms on 1251923, 1251932, 1251936 and 1251940).
const burstPotionDuration = 30 * time.Second

// burstPotions are Forever's 30-second potions with a launch vendor source
// (the alchemy recipes the Azeroth Commerce Authority and Durotar Supply and
// Logistics sell): Frenzy Potion 250940 (aura 99 and aura 124 of 28), Major
// Frenzy Potion 250943 (80), Spellblasting Potion 250934 (aura 13 of 16 on
// every school) and Major Spellblasting Potion 250937 (47). The 1.60.1.70707
// client puts each in the potion category (4, 120000 ms).
var burstPotions = map[proto.Potions]burstPotion{
	proto.Potions_FrenzyPotion:             {itemID: 250940, label: "Frenzy Potion", bonus: stats.Stats{stats.AttackPower: 28, stats.RangedAttackPower: 28}},
	proto.Potions_MajorFrenzyPotion:        {itemID: 250943, label: "Major Frenzy Potion", bonus: stats.Stats{stats.AttackPower: 80, stats.RangedAttackPower: 80}},
	proto.Potions_SpellblastingPotion:      {itemID: 250934, label: "Spellblasting Potion", bonus: stats.Stats{stats.SpellDamage: 16}},
	proto.Potions_MajorSpellblastingPotion: {itemID: 250937, label: "Major Spellblasting Potion", bonus: stats.Stats{stats.SpellDamage: 47}},
}

// makeBurstPotionMCD drinks a 30-second potion as a DPS cooldown.
func makeBurstPotionMCD(potion burstPotion, character *Character, cdTimer *Timer) MajorCooldown {
	actionID := ActionID{ItemID: potion.itemID}
	aura := character.NewTemporaryStatsAura(potion.label, actionID, potion.bonus, burstPotionDuration)
	return MajorCooldown{
		Type: CooldownTypeDPS,
		Spell: character.GetOrRegisterSpell(SpellConfig{
			ActionID: actionID,
			Flags:    SpellFlagNoOnCastComplete,
			Cast: CastConfig{
				CD: Cooldown{
					Timer:    cdTimer,
					Duration: time.Minute * 2,
				},
			},
			ApplyEffects: func(sim *Simulation, _ *Unit, _ *Spell) {
				aura.Activate(sim)
			},
		}),
	}
}

func makePotionActivationInternal(potionType proto.Potions, character *Character, potionCD *Timer) MajorCooldown {
	if potionType == proto.Potions_UnknownPotion {
		return MajorCooldown{}
	}

	switch potionType {
	case proto.Potions_LesserHealingPotion:
		return makeHealthConsumableMCD(858, character, potionCD)
	case proto.Potions_HealingPotion:
		return makeHealthConsumableMCD(929, character, potionCD)
	case proto.Potions_GreaterHealingPotion:
		return makeHealthConsumableMCD(1710, character, potionCD)
	case proto.Potions_SuperiorHealingPotion:
		return makeHealthConsumableMCD(3928, character, potionCD)
	case proto.Potions_MajorHealingPotion:
		return makeHealthConsumableMCD(13446, character, potionCD)

	case proto.Potions_LesserManaPotion:
		return makeManaConsumableMCD(3385, character, potionCD)
	case proto.Potions_ManaPotion:
		return makeManaConsumableMCD(3827, character, potionCD)
	case proto.Potions_GreaterManaPotion:
		return makeManaConsumableMCD(6149, character, potionCD)
	case proto.Potions_SuperiorManaPotion:
		return makeManaConsumableMCD(13443, character, potionCD)
	case proto.Potions_MajorManaPotion:
		return makeManaConsumableMCD(13444, character, potionCD)

	case proto.Potions_RagePotion:
		return makeRageConsumableMCD(5631, character, potionCD)
	case proto.Potions_GreatRagePotion:
		return makeRageConsumableMCD(5633, character, potionCD)
	case proto.Potions_MightyRagePotion:
		return makeRageConsumableMCD(13442, character, potionCD)

	case proto.Potions_LesserStoneshieldPotion:
		return makeArmorConsumableMCD(4623, character, potionCD)
	case proto.Potions_GreaterStoneshieldPotion:
		return makeArmorConsumableMCD(13455, character, potionCD)

	case proto.Potions_MagicResistancePotion:
		return makeMagicResistancePotionMCD(character, potionCD)

	case proto.Potions_FrenzyPotion, proto.Potions_MajorFrenzyPotion,
		proto.Potions_SpellblastingPotion, proto.Potions_MajorSpellblastingPotion:
		return makeBurstPotionMCD(burstPotions[potionType], character, potionCD)
	// case proto.Potions_GreaterArcaneProtectionPotion:
	// 	return makeSchoolProtectionConsumableMCD(13461, character, potionCD)
	// case proto.Potions_GreaterFireProtectionPotion:
	// 	return makeSchoolProtectionConsumableMCD(13457, character, potionCD)
	// case proto.Potions_GreaterFrostProtectionPotion:
	// 	return makeSchoolProtectionConsumableMCD(13456, character, potionCD)
	// case proto.Potions_GreaterFrostProtectionPotion:
	// 	return makeSchoolProtectionConsumableMCD(13460, character, potionCD)
	// case proto.Potions_GreaterFrostProtectionPotion:
	// 	return makeSchoolProtectionConsumableMCD(13458, character, potionCD)
	// case proto.Potions_GreaterFrostProtectionPotion:
	// 	return makeSchoolProtectionConsumableMCD(13459, character, potionCD)
	default:
		return MajorCooldown{}
	}
}

func registerConjuredCD(agent Agent, consumes *proto.Consumes) {
	character := agent.GetCharacter()
	conjuredType := consumes.DefaultConjured

	if conjuredType == proto.Conjured_ConjuredUnknown {
		return
	}

	timer := character.GetConjuredCD()

	var mcd MajorCooldown
	switch conjuredType {
	case proto.Conjured_ConjuredHealthstone:
		mcd = makeHealthConsumableMCD(5509, character, timer)
	case proto.Conjured_ConjuredGreaterHealthstone:
		mcd = makeHealthConsumableMCD(5510, character, timer)
	case proto.Conjured_ConjuredMajorHealthstone:
		mcd = makeHealthConsumableMCD(9421, character, timer)
	case proto.Conjured_ConjuredDemonicRune:
		mcd = makeManaConsumableMCD(12662, character, timer)
	case proto.Conjured_ConjuredMinorRecombobulator:
		mcd = makeManaConsumableMCD(4381, character, timer)
	case proto.Conjured_ConjuredRogueThistleTea:
		mcd = makeThistleTeaMCD(character, timer)
		if mcd.Spell == nil {
			return
		}
	default:
		return
	}

	// Demonic Rune's 16666 is a category-30 spell with a 60 s category
	// recovery (RestoreCategoryRecovery), which holds Robe of the Archmage.
	// The healthstones are in category 30 with no recovery row, and the
	// Minor Recombobulator's 18805 is in no category.
	if conjuredType == proto.Conjured_ConjuredDemonicRune {
		apply := mcd.Spell.ApplyEffects
		mcd.Spell.ApplyEffects = func(sim *Simulation, target *Unit, spell *Spell) {
			apply(sim, target, spell)
			character.HoldRestoreCategory(sim)
		}
	}

	character.AddMajorCooldown(mcd)
}

// addMeleeWeaponCrit gives crit to the unit's own melee attacks: the
// spells that roll the melee table (melee defense, or physical with none)
// and are not ranged. It is a per-spell bonus, not a stat, because a
// weapon enchant's crit belongs to the weapon: a stat would also reach
// whatever inherits the owner's stats.
func addMeleeWeaponCrit(unit *Unit, rating float64) {
	unit.OnSpellRegistered(func(spell *Spell) {
		if spell.CastType == proto.CastType_CastTypeRanged {
			return
		}
		if spell.DefenseType == DefenseTypeMelee ||
			(spell.DefenseType == DefenseTypeNone && spell.SpellSchool == SpellSchoolPhysical) {
			spell.BonusCritRating += rating
		}
	})
}

// thistleTeaEnergy is Thistle Tea's 100 Energy (item 7676, spell 9512,
// energize of 100 at every level).
const thistleTeaEnergy = 100.0

// makeThistleTeaMCD is Thistle Tea for any character with an energy bar: a
// rogue, and a druid since Forever ("Rage potions and Thistletea are also
// now usable by Druids", the druid deep dive, news.blizzard.com article
// 24301515). It has its own five-minute cooldown and the two-minute conjured
// category timer, and is drunk when the bar is nearly empty so the energy is
// not capped away. A character with no energy bar gets nothing.
func makeThistleTeaMCD(character *Character, conjuredTimer *Timer) MajorCooldown {
	if !character.HasEnergyBar() {
		return MajorCooldown{}
	}
	actionID := ActionID{ItemID: 7676}
	energyMetrics := character.NewEnergyMetrics(actionID)

	return MajorCooldown{
		Type: CooldownTypeDPS,
		ShouldActivate: func(sim *Simulation, character *Character) bool {
			return character.CurrentEnergy() <= 10
		},
		Spell: character.RegisterSpell(SpellConfig{
			ActionID: actionID,
			Cast: CastConfig{
				CD: Cooldown{
					Timer:    character.NewTimer(),
					Duration: time.Minute * 5,
				},
				SharedCD: Cooldown{
					Timer:    conjuredTimer,
					Duration: time.Minute * 2,
				},
			},
			ApplyEffects: func(sim *Simulation, _ *Unit, _ *Spell) {
				character.AddEnergy(sim, thistleTeaEnergy, energyMetrics)
			},
		}),
	}
}
