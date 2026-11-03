package core

import (
	"fmt"
	"slices"
	"sort"
	"time"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// Forever gives every race two active and two passive racials, "tuned for
// similar offensive power" (Blizzard, Deep Dive panel recap). This file is
// that table.
//
// SOURCING. The shape is confirmed by Blizzard. Every name and number is
// read from the 1.60.1.70753 client tables (SkillLineAbility on each race's
// racial skill line, then SpellEffect, SpellMisc, SpellAuraOptions and
// SpellCooldowns); data/curated/races.json in the site repository keeps the
// citations. An entry stays Confirmed: false, with a Note saying what is
// unread, only while the client leaves a name or a number open, and
// UnconfirmedRacials() reports exactly those for the spec support page. As
// of 1.60.1.70753 there are none.
//
// TEN RACES. Skyborne is one neutral race whose faction is chosen at
// character creation and whose second active differs by faction, so the
// client's race table carries it as two rows and so does this file:
// RaceHighOrderSkyborne (Alliance, Read Leyline) and
// RaceWindshaperSkyborne (Horde, Skysight). Their other three racials are
// identical and are written once, by skyborneShared().
//
// A racial with no combat effect still gets an entry, with an Apply that
// does nothing and a comment saying why. A missing entry and a
// deliberately empty one are different facts and the support page prints
// them differently.

type RacialKind int

const (
	RacialPassive RacialKind = iota
	RacialActive
)

func (k RacialKind) String() string {
	if k == RacialActive {
		return "active"
	}
	return "passive"
}

// Racial is one of a race's four abilities.
type Racial struct {
	Name string
	Kind RacialKind
	// SpellID is the client's id where one is known, and 0 where it is
	// not. Forever keeps vanilla ids for abilities that already existed
	// and uses ids above 1,000,000 only for new objects, so a new
	// Forever racial's id is above a million or it is unknown.
	SpellID int32
	// Confirmed is true when the entry's numbers (if it has any) are
	// corroborated rather than contested or simply missing, and when the
	// racial itself is named by a source. See the SOURCING note above.
	Confirmed bool
	// Note says what is unread. Required when Confirmed is false.
	Note  string
	Apply func(*Character)
}

// playableRaces is the ten, in the client race table's own order.
var playableRaces = []proto.Race{
	proto.Race_RaceHuman,
	proto.Race_RaceOrc,
	proto.Race_RaceDwarf,
	proto.Race_RaceNightElf,
	proto.Race_RaceUndead,
	proto.Race_RaceTauren,
	proto.Race_RaceGnome,
	proto.Race_RaceTroll,
	proto.Race_RaceHighOrderSkyborne,
	proto.Race_RaceWindshaperSkyborne,
}

// PlayableRaces returns the ten races, in the client's order.
func PlayableRaces() []proto.Race {
	return append([]proto.Race(nil), playableRaces...)
}

// RacialsFor returns a race's four racials. An unknown race returns nil,
// which applyRaceEffects treats as "apply nothing" exactly as the old
// switch's default arm did.
func RacialsFor(race proto.Race) []Racial {
	return racialsByRace[race]
}

// UnconfirmedRacials names every entry whose numbers are not settled, for
// the spec support page. Sorted, so the page does not churn.
func UnconfirmedRacials() []string {
	var out []string
	for _, race := range playableRaces {
		for _, r := range racialsByRace[race] {
			if !r.Confirmed {
				out = append(out, fmt.Sprintf("%s: %s (%s)", race.String(), r.Name, r.Note))
			}
		}
	}
	sort.Strings(out)
	return out
}

func applyRaceEffects(agent Agent) {
	character := agent.GetCharacter()
	for _, r := range RacialsFor(character.Race) {
		r.Apply(character)
	}
}

// skyborneShared is Walk on Air, Wind Blessed and Elemental Insight: the
// three racials both Skyborne rows have. Only the second active differs
// by faction, so only that one is written per row.
func skyborneShared() []Racial {
	return []Racial{
		{
			Name: "Walk on Air", Kind: RacialActive, Confirmed: true,
			Note:  "",
			Apply: func(*Character) {}, // glide downward for 10 s: no combat effect
		},
		{
			Name: "Wind Blessed", Kind: RacialPassive, Confirmed: true,
			Note: "client 1259710: 2% spellcasting, melee and ranged haste (aura 342 and aura 65 at 2 in 1.60.1.70753)",
			Apply: func(character *Character) {
				// Haste is NOT merged (research/08-stats.md 12.1 item 5),
				// so this sets all three, as a racial that just says
				// "haste" must.
				character.PseudoStats.MeleeSpeedMultiplier *= windBlessedHasteMultiplier
				character.PseudoStats.RangedSpeedMultiplier *= windBlessedHasteMultiplier
				character.PseudoStats.CastSpeedMultiplier *= windBlessedHasteMultiplier
			},
		},
		{
			Name: "Elemental Insight", Kind: RacialPassive, Confirmed: true,
			Note: "client 1259707: 5% damage dealt versus Elementals",
			Apply: func(character *Character) {
				applyMobTypeDamageMultiplier(character, proto.MobType_MobTypeElemental, mobTypeDamageMultiplier)
			},
		},
	}
}

// applyMobTypeDamageMultiplier registers a post-finalize effect that
// multiplies DamageDealtMultiplier against every target of the given
// MobType: the client's "damage dealt versus <type> increased by 5%" aura
// (aura 168). Shared by Elemental Insight, Beast Slaying and Big Game
// Hunter, the racials that key off mob type the way the item-side
// NewMobTypeDamageEffect does.
func applyMobTypeDamageMultiplier(character *Character, mobType proto.MobType, damageMultiplier float64) {
	character.Env.RegisterPostFinalizeEffect(func() {
		for _, t := range character.Env.Encounter.Targets {
			if t.MobType == mobType {
				for _, at := range character.AttackTables[t.UnitIndex] {
					at.DamageDealtMultiplier *= damageMultiplier
				}
			}
		}
	})
}

// mobTypeDamageMultiplier is the client's 5% for Elemental Insight 1259707,
// Beast Slaying 20557 and Big Game Hunter 1259721 (effect aura 168, base
// points 5). Damage dealt covers crits already, so no separate crit
// multiplier is applied.
const mobTypeDamageMultiplier = 1.05

// ruthlessCritPercent and artisansPrecisionHitPercent are the Human and
// Dwarf weapon racials after the October rework, from the current client
// text of spells 20597 and 1259719.
const (
	ruthlessCritPercent         = 1
	artisansPrecisionHitPercent = 1
	// humanSpiritMultiplier is The Human Spirit (20598) after Blizzard's
	// 26 October notes ("increases Spirit by 10% (was 5%)") and the
	// 30 October hotfix cache; the shipped 1.60.1.70707 row still reads 5.
	humanSpiritMultiplier = 1.10
	// windBlessedHasteMultiplier is Skyborne's Wind Blessed (1259710):
	// 26 October notes, "Haste bonus reduced to 2% (was 3%)"; the shipped
	// 1.60.1.70707 rows still read 3.
	windBlessedHasteMultiplier = 1.02
)

var racialsByRace = map[proto.Race][]Racial{
	// Human: Will to Survive, Perception, Ruthless, The Human Spirit.
	proto.Race_RaceHuman: {
		{
			Name: "Will to Survive", Kind: RacialActive, Confirmed: true,
			Note:  "",
			Apply: func(*Character) {}, // removes all Stuns, 3 min cooldown: no combat effect
		},
		{
			Name: "Perception", Kind: RacialActive, Confirmed: true,
			Note:  "",
			Apply: func(*Character) {}, // 20 s of stealth detection: no combat effect
		},
		{
			Name: "Ruthless", Kind: RacialPassive, Confirmed: true,
			Note: "client 20597: 1% crit with all spells and attacks, any weapon",
			Apply: func(character *Character) {
				// The 15 October rework replaced Sword Specialization; no
				// SpellEquippedItems row gates it. The 70658 tables carry
				// 2%, the current client text 1% (cut on 26 October).
				character.AddStat(stats.Crit, ruthlessCritPercent*CritRatingPerCritChance)
			},
		},
		{
			Name: "The Human Spirit", Kind: RacialPassive, Confirmed: true,
			Note:  "client 20598: 10% Spirit (aura 137 at 10 in 1.60.1.70753)",
			Apply: func(character *Character) { character.MultiplyStat(stats.Spirit, humanSpiritMultiplier) },
		},
	},

	// Orc: Blood Fury, Shatter Curse, Command, Hardiness.
	proto.Race_RaceOrc: {
		{
			Name: "Blood Fury", Kind: RacialActive, Confirmed: true,
			Note:  "client 20572: 10% attack power, ranged attack power and spell power for 15 s, 2 min cooldown",
			Apply: applyBloodFury,
		},
		{
			Name: "Shatter Curse", Kind: RacialActive, Confirmed: true,
			Note:  "",
			Apply: func(*Character) {}, // removes/immunes Curses and Banes, -15% magic damage taken 8 s: utility, no offensive effect
		},
		{
			Name: "Command", Kind: RacialPassive, Confirmed: true,
			Note:  "hunter and warlock pets deal 5% more damage (20576 ClassMask 4, 20575 ClassMask 256, each +5 to the pet's Tamed Pet Passive); the warlock's Imp Firebolt and Succubus Lash of Pain take 5% more on top (20575 effect 2, 1.60.1.70753)",
			Apply: applyCommand,
		},
		{
			Name: "Hardiness", Kind: RacialPassive, Confirmed: true,
			Note:  "",
			Apply: func(*Character) {}, // -20% stun duration: no combat effect
		},
	},

	// Dwarf: Stoneform, Find Treasure, Mace Specialization, Big Game
	// Hunter. The Deep Dive panel confirms Stoneform now reduces
	// physical damage taken instead of raising armor.
	proto.Race_RaceDwarf: {
		{
			Name: "Stoneform", Kind: RacialActive, Confirmed: true,
			Note: "",
			Apply: func(character *Character) {
				actionID := ActionID{SpellID: 20594}

				stoneformAura := character.NewTemporaryStatsAuraWrapped("Stoneform", actionID, stats.Stats{}, time.Second*8, func(aura *Aura) {
					aura.ApplyOnGain(func(aura *Aura, sim *Simulation) {
						aura.Unit.PseudoStats.SchoolDamageTakenMultiplier[stats.SchoolIndexPhysical] *= 0.9
					})
					aura.ApplyOnExpire(func(aura *Aura, sim *Simulation) {
						aura.Unit.PseudoStats.SchoolDamageTakenMultiplier[stats.SchoolIndexPhysical] /= 0.9
					})
				})

				spell := character.RegisterSpell(SpellConfig{
					ActionID: actionID,
					Flags:    SpellFlagNoOnCastComplete,
					Cast: CastConfig{
						CD: Cooldown{
							Timer:    character.NewTimer(),
							Duration: time.Minute * 3,
						},
					},
					ApplyEffects: func(sim *Simulation, _ *Unit, _ *Spell) {
						stoneformAura.Activate(sim)
					},
				})

				character.AddMajorCooldown(MajorCooldown{
					Spell:    spell,
					Type:     CooldownTypeSurvival,
					SelfBuff: true,
					ShouldActivate: func(s *Simulation, c *Character) bool {
						// Only castable with manual APL Action
						return false
					},
				})
			},
		},
		{
			Name: "Find Treasure", Kind: RacialActive, Confirmed: true,
			Note:  "",
			Apply: func(*Character) {}, // finds treasure, stacks with other tracking: no combat effect
		},
		{
			Name: "Artisan's Precision", Kind: RacialPassive, Confirmed: true,
			Note: "client 1259719: 1% hit with all spells and attacks",
			Apply: func(character *Character) {
				// Replaced Earth Fury (the 70658 tables' crit damage with
				// maces) on 26 October; the current client text reads 1% hit.
				character.AddStat(stats.Hit, artisansPrecisionHitPercent*HitRatingPerHitChance)
			},
		},
		{
			Name: "Big Game Hunter", Kind: RacialPassive, Confirmed: true,
			Note: "client 1259721: 5% damage dealt versus Beasts",
			Apply: func(character *Character) {
				applyMobTypeDamageMultiplier(character, proto.MobType_MobTypeBeast, mobTypeDamageMultiplier)
			},
		},
	},

	// Night Elf: Elune's Light, Shadowmeld, Quickness, Wisp Spirit.
	proto.Race_RaceNightElf: {
		{
			Name: "Elune's Light", Kind: RacialActive, Confirmed: true,
			Note: "client 1259799: 10% crit with all spells and attacks for 15 s, 3 min cooldown",
			Apply: func(character *Character) {
				// +10% crit for 15 s, 3 min cooldown, a major cooldown.
				actionID := ActionID{SpellID: 1259799}
				elunesLightAura := character.NewTemporaryStatsAura("Elune's Light", actionID, stats.Stats{stats.Crit: 10 * CritRatingPerCritChance}, time.Second*15)

				spell := character.RegisterSpell(SpellConfig{
					ActionID: actionID,
					Flags:    SpellFlagNoOnCastComplete,
					Cast: CastConfig{
						CD: Cooldown{
							Timer:    character.NewTimer(),
							Duration: time.Minute * 3,
						},
					},
					ApplyEffects: func(sim *Simulation, _ *Unit, _ *Spell) {
						elunesLightAura.Activate(sim)
					},
				})

				character.AddMajorCooldown(MajorCooldown{
					Spell:    spell,
					Type:     CooldownTypeDPS,
					SelfBuff: true,
				})
			},
		},
		{
			Name: "Shadowmeld", Kind: RacialActive, Confirmed: true,
			Note:  "",
			Apply: func(*Character) {}, // stealth until you move, usable in combat, 2 min cooldown: no combat effect
		},
		{
			Name: "Quickness", Kind: RacialPassive, Confirmed: true,
			Note: "client 20582: 1% dodge and 2% movement speed (speed has no combat effect and is not modeled)",
			Apply: func(character *Character) {
				character.AddStat(stats.Dodge, 1*DodgeRatingPerDodgeChance)
			},
		},
		{
			Name: "Wisp Spirit", Kind: RacialPassive, Confirmed: true,
			Note:  "",
			Apply: func(*Character) {}, // 75% speed while dead: no combat effect
		},
	},

	// Undead: Will of the Forsaken, Cannibalize, Touch of the Grave and
	// Underwater Breathing, the four SkillLineAbility rows of Racial -
	// Undead (220) in 1.60.1.70753.
	proto.Race_RaceUndead: {
		{
			Name: "Will of the Forsaken", Kind: RacialActive, Confirmed: true,
			Note:  "",
			Apply: func(*Character) {}, // removes Fear, Charm and Sleep: no combat effect
		},
		{
			Name: "Cannibalize", Kind: RacialActive, Confirmed: true,
			Note:  "",
			Apply: func(*Character) {}, // channeled heal from a nearby corpse, out of combat: no combat effect
		},
		{
			Name: "Touch of the Grave", Kind: RacialPassive, Confirmed: true,
			Note:  "client 1260189 (warrior, paladin, rogue: 20%) or 1260201 (priest, mage, warlock: 30%), 15 s internal cooldown (SpellAuraOptions 15000 ms in 1.60.1.70753); drain 1260198 takes 5% of maximum Health as flat Shadow damage; see racial_touch_of_the_grave.go",
			Apply: applyTouchOfTheGrave,
		},
		{
			Name: "Underwater Breathing", Kind: RacialPassive, SpellID: 5227, Confirmed: true,
			Note:  "client 5227 (SkillLineAbility, Racial - Undead): aura 155, breath lasts 300% longer",
			Apply: func(*Character) {}, // underwater breath: no combat effect
		},
	},

	// Tauren: War Stomp, Cultivation, Endurance, Plainsrunning. The
	// 1.60.1.70753 SkillLineAbility rows of Tauren Racial (124) carry all
	// four; Spell's subtext marks Cultivation the active and Plainsrunning
	// the passive.
	proto.Race_RaceTauren: {
		{
			Name: "War Stomp", Kind: RacialActive, Confirmed: true,
			Note:  "",
			Apply: func(*Character) {}, // stuns up to 5 enemies within 8 yd for 2 s, 2 min cooldown: crowd control, not modeled
		},
		{
			Name: "Cultivation", Kind: RacialActive, SpellID: 20552, Confirmed: true,
			Note:  "client 20552 (subtext Racial): grows a duplicate herb",
			Apply: func(*Character) {}, // herb duplication: no combat effect
		},
		{
			Name: "Endurance", Kind: RacialPassive, Confirmed: true,
			Note: "client 20550: 5% Health",
			Apply: func(character *Character) {
				// 5% Health, the usual multiplicative stat dependency
				// every other "+X%" racial in this file uses. The 1% Hit an
				// early demo transcription gave it is in neither the
				// 1.60.1.70707 table nor Wowhead Forever's live text. The 2%
				// critical strike damage of earlier builds (aura 163) is gone:
				// 20550 is one effect, aura 133 at 5, in 70707 and 70753, and
				// the tooltip reads "Total Health increased by 5%".
				character.MultiplyStat(stats.Health, 1.05)
			},
		},
		{
			Name: "Plainsrunning", Kind: RacialPassive, SpellID: 1259918, Confirmed: true,
			Note:  "client 1259918 (subtext Racial Passive): movement speed that grows while moving",
			Apply: func(*Character) {}, // movement speed: no combat effect
		},
	},

	// Gnome: Escape Artist, Eureka!, Expansive Mind, Engineering
	// Specialization.
	proto.Race_RaceGnome: {
		{
			Name: "Escape Artist", Kind: RacialActive, Confirmed: true,
			Note:  "",
			Apply: func(*Character) {}, // breaks movement impairment and grants brief immunity, 2 min cooldown: no combat effect either at the 3 s or 5 s reading
		},
		{
			Name: "Eureka!", Kind: RacialActive, Confirmed: true,
			Note:  "one client row per class (1259813 Warrior, 1259812 Rogue, 1259817 Mage, 1259823 Priest, 1259821 Warlock): 8 s, no charges, 2 min cooldown; +10% damage or healing, +10% periodic damage and -10% cost on the spells each effect's class mask names; see eureka.go",
			Apply: applyEureka,
		},
		{
			Name: "Expansive Mind", Kind: RacialPassive, Confirmed: true,
			Note: "client 20591 (Mana), 1259802 (Rage), 1259803 (Energy): +5% maximum of each",
			Apply: func(character *Character) {
				// 5% Mana, Rage and Energy. Mana takes the multiplicative
				// stat dependency every other "+X%" racial in this file
				// uses; Rage and Energy are fixed 100-point pools outside
				// the stat-dependency system (see stats/deps.go's
				// safeDepsOrder), so their 5% is written as the flat
				// point value of 5% of Classic's 100-point cap.
				character.MultiplyStat(stats.Mana, 1.05)
				character.AddStat(stats.Rage, 0.05*MaxRage)
				character.AddStat(stats.Energy, 0.05*expansiveMindEnergyCap)
			},
		},
		{
			Name: "Engineering Specialization", Kind: RacialPassive, Confirmed: true,
			Note:  "",
			Apply: func(*Character) {}, // engineering devices more reliable: no combat effect
		},
	},

	// Troll: Berserking, Rapid Regeneration, Beast Slaying, Regeneration.
	proto.Race_RaceTroll: {
		{
			Name: "Berserking", Kind: RacialActive, Confirmed: true,
			Note: "client 20554: +10% spellcasting and attack speed for 15 s, 3 min cooldown, no resource cost; the engine's custom-percentage cooldowns (15 to 30) are APL compatibility tags, not client rows",
			Apply: func(character *Character) {
				// Haste is NOT merged (research/08-stats.md 12.1 item 5),
				// so the aura sets both melee/ranged and spell haste.
				berserkingTimer := character.NewTimer()
				makeBerserkingCooldown(character, 0, berserkingTimer)
				makeBerserkingCooldown(character, .1, berserkingTimer)
				makeBerserkingCooldown(character, .15, berserkingTimer)
				makeBerserkingCooldown(character, .2, berserkingTimer)
				makeBerserkingCooldown(character, .25, berserkingTimer)
				makeBerserkingCooldown(character, .3, berserkingTimer)
			},
		},
		{
			Name: "Rapid Regeneration", Kind: RacialActive, Confirmed: true,
			Note:  "",
			Apply: func(*Character) {}, // channels back 50% of maximum Health: out-of-combat utility, no combat effect
		},
		{
			Name: "Beast Slaying", Kind: RacialPassive, Confirmed: true,
			Note: "client 20557: 5% damage dealt versus Beasts",
			Apply: func(character *Character) {
				applyMobTypeDamageMultiplier(character, proto.MobType_MobTypeBeast, mobTypeDamageMultiplier)
			},
		},
		{
			Name: "Regeneration", Kind: RacialPassive, Confirmed: true,
			Note:  "",
			Apply: func(*Character) {}, // keeps 10% of Health regeneration running in combat: no combat effect
		},
	},

	// Skyborne (Alliance): Walk on Air, Read Leyline, Wind Blessed,
	// Elemental Insight.
	proto.Race_RaceHighOrderSkyborne: append([]Racial{{
		Name: "Read Leyline", Kind: RacialActive, SpellID: readLeylineSpellID, Confirmed: true,
		Note:  "client 1259705: Energized 1270842, 50% mana and health regeneration for 15 s, 2 min cooldown, on the global cooldown; APL-cast only, see racial_read_leyline.go",
		Apply: applyReadLeyline,
	}}, skyborneShared()...),

	// Skyborne (Horde): Walk on Air, Skysight, Wind Blessed, Elemental
	// Insight - identical to the Alliance row except the second active.
	proto.Race_RaceWindshaperSkyborne: append([]Racial{{
		Name: "Skysight", Kind: RacialActive, Confirmed: true,
		Note:  "",
		Apply: func(*Character) {}, // movement and mounted speed: no combat effect
	}}, skyborneShared()...),
}

// expansiveMindEnergyCap is Classic's fixed Energy pool size, used to
// convert Expansive Mind's "+5% Energy" into a flat point value the same
// way MaxRage is used for Rage. Unexported and file-local rather than a
// new shared constant: sim/core/energy.go and sim/rogue/rogue.go already
// hardcode this same 100-point baseline inline, and unifying all three
// into one shared constant is a cleanup outside this file's scope.
const expansiveMindEnergyCap = 100

// Berserking (client 20554): effects aura 319, 140 and 65 at 10 each and a
// 3 min cooldown, with no resource cost. The duration is 15 s: Blizzard's
// 27 October notes ("increased to 15 seconds (was 10 seconds)"), and the
// 1.60.1.70753 client ships it (SpellMisc DurationIndex 15000 ms).
// A customPercentage of 0 is the client row; any other value makes a
// cooldown hard-coded to that percentage, which existing APLs address by
// tag.
const (
	berserkingBasePercent = 0.10
	berserkingDuration    = 15 * time.Second
	berserkingCooldown    = 3 * time.Minute
)

func makeBerserkingCooldown(character *Character, customPercentage float64, timer *Timer) {
	actionID := ActionID{SpellID: 26297, Tag: int32(customPercentage * 20)}

	label := "Berserking"
	percent := berserkingBasePercent
	if customPercentage != 0 {
		label = fmt.Sprintf("%s (%d)", label, int(customPercentage*100))
		percent = customPercentage
	}
	haste := 1 + percent

	berserkingAura := character.RegisterAura(Aura{
		Label:    label,
		ActionID: actionID,
		Duration: berserkingDuration,
		OnGain: func(aura *Aura, sim *Simulation) {
			character.MultiplyCastSpeed(haste)
			character.MultiplyAttackSpeed(sim, haste)
		},
		OnExpire: func(aura *Aura, sim *Simulation) {
			character.MultiplyCastSpeed(1 / haste)
			character.MultiplyAttackSpeed(sim, 1/haste)
		},
	})

	berserkingSpell := character.RegisterSpell(SpellConfig{
		ActionID: actionID,
		Cast: CastConfig{
			CD: Cooldown{
				Timer:    timer,
				Duration: berserkingCooldown,
			},
		},
		ApplyEffects: func(sim *Simulation, _ *Unit, _ *Spell) {
			berserkingAura.Activate(sim)
		},
	})

	character.AddMajorCooldown(MajorCooldown{
		Spell:    berserkingSpell,
		Type:     CooldownTypeDPS,
		SelfBuff: true,
	})
}

func (character *Character) GetFaction() proto.Faction {
	if slices.Contains([]proto.Race{proto.Race_RaceHuman, proto.Race_RaceDwarf, proto.Race_RaceGnome, proto.Race_RaceNightElf}, character.Race) {
		return proto.Faction_Alliance
	} else if slices.Contains([]proto.Race{proto.Race_RaceOrc, proto.Race_RaceTroll, proto.Race_RaceTauren, proto.Race_RaceUndead}, character.Race) {
		return proto.Faction_Horde
	} else {
		return proto.Faction_Unknown
	}
}
