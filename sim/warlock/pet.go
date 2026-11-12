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

		wp.AddStat(stats.SpellCrit, impSpellCritPercent*core.CritRatingPerCritChance)
	} else {
		// Warrior scaling for all other pets
		wp.AddStat(stats.AttackPower, -20)
		wp.AddStatDependency(stats.Strength, stats.AttackPower, 2)

		wp.AddStats(stats.Stats{
			stats.MeleeCrit: demonMeleeCritPercent * core.CritRatingPerCritChance,
			stats.SpellCrit: demonSpellCritPercent * core.CritRatingPerCritChance,
		})

		// Imps generally don't melee
		wp.EnableAutoAttacks(wp, cfg.AutoAttacks)
	}

	// The demon takes the crit its warlock wears, and only that (see
	// makeStatInheritance): EquipStats is the gear plus the request's bonus
	// stats, which stand for gear. It is fixed for the summon, so no owner
	// buff or proc reaches it.
	wp.AddStat(stats.Crit, warlock.EquipStats()[stats.Crit])

	// Unholy Power: "Increases all damage done by your Imp, Voidwalker,
	// Succubus, Incubus, and Felhunter pets by 2%" a rank; the Forever
	// Firebolt tooltip applies it too, so it is pet-wide, not melee only.
	// The client's 18769 adds points to the demon's Tamed Pet Passive
	// (aura 107, misc 8), so it adds with Orc Command's +5.
	wp.AddPassiveDamagePercent(100 * unholyPowerDamagePerRank * float64(warlock.Talents.UnholyPower))

	// Forever demons scale from the warlock (Warlock Pet Scaling 416189, the
	// same server-computed channels as Hunter Pet Scaling 415429; see
	// makeStatInheritance) and, like hunter pets, take no player buffs but
	// Heart of the Lion's area buff, the one group buff the 70755 client does
	// not flag "can only target players" (409583, ImplicitTarget 31).
	wp.RefusePlayerBuffs()
	wp.TakeHeartOfTheLion()

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
// the server computes every amount. Forever's character sheet states four of
// them: WARLOCK_PET_BONUS in the 1.60.1.70532 client's Camelot
// PaperDollFrame.lua gives 17% of the warlock's better Shadow or Fire spell
// damage as attack power, 10% as spell damage, 35% of its armor and 100% of
// its resistances (PET_BONUS_SPELLDMG_TO_AP, _SPELLDMG_TO_SPELLDMG, _ARMOR,
// _RES). wowsims/forever read the same tenth off beta pet sheets and a logged
// Sayaad (sim/warlock/pets.go, c3d9ea951, 2026-10-14). Classic's 57% and 15%
// (wowsims/sod, warcraft.wiki.gg) are not Forever's. The client sheet says
// pets have no primary attributes (PET_BONUS_STAM and _INT 0); the 20%
// Stamina and 30% Intellect stand in for the health and mana the server
// grants instead. Blizzard's 3 Nov notes: "Pets now gain 2 Health per point
// of Master's Stamina (was 7.5 Health per point of Master's Stamina)", so a
// fifth of the warlock's Stamina at 10 health a point (the beta's 75% is gone).
const (
	petStaminaShare             = 0.20
	petIntellectShare           = 0.30
	petArmorShare               = 0.35
	petResistanceShare          = 1.00
	petAttackPowerPerSpellPower = 0.17
	petSpellDamagePerSpellPower = 0.10
)

// A Forever demon's crit is its own, measured rate; Agility and Intellect do
// not raise it (the sheets' attributes change by level, the measured rates
// do not). Blizzard on forever-bugs #331 (2026-10-14): "Their melee crit is
// extremely low and their base spell crit is around normal levels".
//   - demonMeleeCritPercent: 12 crits in 1,718 Succubus swings (0.70%) by a
//     level-30 warlock on level-30 Cave Yetis (#331, three logs); the 29
//     foreverlogs.gg warlock logs give demon melee 33 in 2,400 (1.4%) on mobs
//     of mixed level. Against a boss the melee crit suppression (4.8%)
//     leaves the demon none.
//   - demonSpellCritPercent: Lash of Pain 13 in 205 in the same test and 7
//     in 105 in the public logs of warlocks without Malevolence (6.4%). The
//     Felhunter and Voidwalker cast nothing the engine models and take the
//     same value.
//   - impSpellCritPercent: Firebolt 207 in 3,501 (5.9%) by 13 live Imps
//     (foreverlogs.gg 2816 to 2853, 5 to 12 Nov, owners at 12 to 40; each
//     Imp between 3.6 and 7.7%, no owner with crit gear to speak of). The
//     beta's 17 public logs read 2.2% (53 in 2,383); the live Imp crits at
//     the other demons' spell rate, as Blizzard's "base spell crit is
//     around normal levels" says. The live Lash of Pain rate, 60 in 1,061
//     by eight Succubi (5.7%), keeps demonSpellCritPercent.
//
// unconfirmed: the rates are measured at levels 12 to 40; no level-60 demon
// log exists yet. One live Succubus (Zahnia, 2845 and 2853) crits 4.0% of
// 1,206 swings where Voidwalkers crit 0 of 733; whether that is her owner's
// gear or a Succubus rate needs his sheet.
const (
	demonMeleeCritPercent = 0.7
	demonSpellCritPercent = 6.4
	impSpellCritPercent   = 5.9
)

// petMeleeHitShare is the part of the warlock's hit a demon's melee takes.
// forever-bugs #394 (AidanZMoon for Blizzard, 2026-10-20): "Warlocks will
// give spell hit equal to their spell hit, and melee hit equal to 8/17ths
// of their spell hit." The demon's spells take the whole of it: its stat
// carries the melee share and applyOwnerSpellHit adds the rest per cast.
const petMeleeHitShare = 8.0 / 17

// makeStatInheritance maps the warlock's stats to the demon's. Crit is not
// in it. Blizzard's intent is that a demon mirrors the warlock's spell crit
// with no crit of its own (forever-bugs #331, 2026-10-14), but it is not
// enabled: demons keep their own crit (demonMeleeCritPercent and its
// neighbours) and, by the same reply, "only scale with their master's +%
// critical strike gear". makePet adds that gear crit once at the summon; the
// warlock's crit from Intellect, racials, raid auras and procs never reaches
// the demon (#371, crit racials missing from the pet sheet, closed as #331's
// duplicate on 2026-10-16; no note through 2026-11-04 enables the mirror).
// Haste is not
// inherited: this engine carries haste as speed multipliers, not stats. The
// mapping is linear in the owner's stats, so the same function maps the
// owner's mid-fight stat changes; the school power it reads is fixed per
// iteration (petSpellPowerSchool).
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
			stats.Hit:              owner[stats.Hit] * petMeleeHitShare,
		}
	}
}

// applyOwnerSpellHit tops a demon spell's hit up from the melee share its
// stat carries to the warlock's whole hit, read at the cast.
func (wp *WarlockPet) applyOwnerSpellHit(spell *core.Spell) {
	spell.BonusHitRating = wp.owner.GetStat(stats.Hit) * (1 - petMeleeHitShare)
}

// higherSchoolPower names the warlock's larger school power, Fire or Shadow,
// the one a demon inherits beside the all-school spell power.
func higherSchoolPower(owner stats.Stats) stats.Stat {
	if owner[stats.FirePower] > owner[stats.ShadowPower] {
		return stats.FirePower
	}
	return stats.ShadowPower
}

// petSheetLevels are the levels each demon's base sheet is carried at.
var petSheetLevels = []int32{25, 40, 50, 60}

// petSheetAtLevel is a demon's base stats and weapon at level. The sheets
// exist only at petSheetLevels (wowsims' Classic figures); the site also sims
// levels 20 and 30, where a missing sheet left the demon with no stats or
// weapon. Elsewhere the sheet is interpolated linearly by level between the
// two nearest carried levels, or extrapolated from the lowest two below 25
// (unconfirmed: Forever's creature tables are server-side).
func petSheetAtLevel(level int32, sheet func(level int32) PetConfig) (stats.Stats, core.AutoAttackOptions) {
	lo, hi := petSheetLevels[0], petSheetLevels[1]
	for i, anchor := range petSheetLevels {
		if anchor == level {
			cfg := sheet(level)
			return cfg.Stats, cfg.AutoAttacks
		}
		if anchor < level && i+1 < len(petSheetLevels) {
			lo, hi = anchor, petSheetLevels[i+1]
		}
	}
	low, high := sheet(lo), sheet(hi)
	t := float64(level-lo) / float64(hi-lo)
	lerp := func(a, b float64) float64 { return max(0, a+(b-a)*t) }

	var sheetStats stats.Stats
	for s := range sheetStats {
		sheetStats[s] = lerp(low.Stats[s], high.Stats[s])
	}
	attacks := low.AutoAttacks
	attacks.MainHand.BaseDamageMin = lerp(low.AutoAttacks.MainHand.BaseDamageMin, high.AutoAttacks.MainHand.BaseDamageMin)
	attacks.MainHand.BaseDamageMax = lerp(low.AutoAttacks.MainHand.BaseDamageMax, high.AutoAttacks.MainHand.BaseDamageMax)
	return sheetStats, attacks
}

// topLearnedRank is the highest rank, counted from 1, whose learn level in
// rankLevels is at most level; a demon below every rank's level still casts
// rank 1.
func topLearnedRank(rankLevels []int, level int) int {
	rank := 1
	for i, learnedAt := range rankLevels {
		if learnedAt <= level {
			rank = i + 1
		}
	}
	return rank
}
