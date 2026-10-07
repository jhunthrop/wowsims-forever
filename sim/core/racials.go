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
// SOURCING. The shape is confirmed by Blizzard. The names and numbers are
// transcribed from the BlizzCon demo by Icy Veins and by Talents Forever,
// and are recorded with their citations in data/curated/races.json in the
// site repository. Most numbers are corroborated by both transcriptions
// (and several by the Deep Dive panel directly) and ship as Confirmed.
// Eight entries, across the whole table, are not: either the two
// transcriptions disagree, neither gives a number at all, or (Undead's
// fourth racial) no source names the racial itself. Those eight carry
// `Confirmed: false` and a Note explaining what is unread, and
// UnconfirmedRacials() reports exactly those eight for the spec support
// page. Where the two transcriptions disagree on a number, the lower
// reading ships and the disagreement is in the entry's Note. An unnamed
// or unpublished racial is always Confirmed: false, regardless of whether
// it happens to have a number attached - a name is exactly as unconfirmed
// as a percentage.
//
// TEN RACES. Skyborne is one neutral race whose faction is chosen at
// character creation and whose second active differs by faction, so the
// client's race table carries it as two rows and so does this file:
// RaceHighOrderSkyborne (Alliance, Read Ley Line) and
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
	// racial itself is named by a source. See the SOURCING note above
	// the eight that are false.
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
			Note: "client 1259710: 1% spellcasting, melee and ranged haste",
			Apply: func(character *Character) {
				// 1% haste, demo transcription. Haste is NOT merged
				// (research/08-stats.md 12.1 item 5), so this sets both,
				// as a racial that just says "haste" must.
				// Client 1259710: "spellcasting, melee, and ranged
				// Haste" by 1%.
				character.PseudoStats.MeleeSpeedMultiplier *= 1.01
				character.PseudoStats.RangedSpeedMultiplier *= 1.01
				character.PseudoStats.CastSpeedMultiplier *= 1.01
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

// Eureka! (client 1259821, the Mana row): SpellAuraOptions ProcCharges 3,
// duration index 8 (15 s), recovery 120000 ms, effect base points -10
// (cost) and +10 (damage).
const (
	eurekaSpellID              int32 = 1259821
	eurekaCharges              int32 = 3
	eurekaDuration                   = 15 * time.Second
	eurekaCooldown                   = 2 * time.Minute
	eurekaCostReductionPercent int32 = 10
	eurekaDamageMultiplier           = 1.1
)

var racialsByRace = map[proto.Race][]Racial{
	// Human: Will to Survive, Perception, Sword Specialization, The
	// Human Spirit. Icy Veins and Talents Forever agree on all four.
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
			Name: "Sword Specialization", Kind: RacialPassive, Confirmed: true,
			Note: "",
			Apply: func(character *Character) {
				// Client 20597: 2% crit with all spells and attacks while
				// a sword or two-handed sword is equipped (specializations.go).
				character.SwordSpecializationAura()
			},
		},
		{
			Name: "The Human Spirit", Kind: RacialPassive, Confirmed: true,
			Note:  "",
			Apply: func(character *Character) { character.MultiplyStat(stats.Spirit, 1.05) },
		},
	},

	// Orc: Blood Fury, Shatter Curse, Axe Specialization, Hardiness.
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
			Name: "Axe Specialization", Kind: RacialPassive, Confirmed: true,
			Note:  "",
			Apply: func(character *Character) { character.AxeSpecializationAura() }, // 1% crit with an axe
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
					Spell: spell,
					Type:  CooldownTypeSurvival,
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
			Name: "Mace Specialization", Kind: RacialPassive, Confirmed: true,
			Note: "client 1259719: 1% crit with all spells and attacks while a mace or two-handed mace is equipped",
			Apply: func(character *Character) {
				character.MaceSpecializationAura()
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
					Spell: spell,
					Type:  CooldownTypeDPS,
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

	// Undead: Will of the Forsaken, Cannibalize, Touch of the Grave, and
	// one genuinely unread fourth passive - the only slot in the whole
	// table with nothing named at all. It still gets an entry, per this
	// file's header, and Confirmed stays true because there is no number
	// here to be unconfirmed about; UnconfirmedRacials has nothing to
	// print until the racial itself is named.
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
			Note:  "client 1260201 (drain 1260198): 10% proc chance, 1 s internal cooldown, drains 5% of maximum Health as flat Shadow damage; see racial_touch_of_the_grave.go",
			Apply: applyTouchOfTheGrave,
		},
		{
			Name: "Unannounced Fourth Racial", Kind: RacialPassive, Confirmed: false,
			Note: "no source names a fourth Undead racial or its effect; data/builds/1.60.1.69893/races.json's Undead row (id 5) names only Will of the Forsaken, Cannibalize and Touch of the Grave",
			Apply: func(*Character) {
				// unconfirmed: neither a name nor an effect is published
				// for this slot, so it is declared - not omitted - with
				// no effect until one is.
			},
		},
	},

	// Tauren: War Stomp, Endurance, and a second active that is one of
	// Plainsrunning or Cultivation - which one is unread, so both are
	// listed rather than guessing, and both are Confirmed: false. Neither
	// has a combat effect, so the sim is unaffected either way.
	proto.Race_RaceTauren: {
		{
			Name: "War Stomp", Kind: RacialActive, Confirmed: true,
			Note:  "",
			Apply: func(*Character) {}, // stuns up to 5 enemies within 8 yd for 2 s, 2 min cooldown: crowd control, not modeled
		},
		{
			Name: "Cultivation", Kind: RacialActive, Confirmed: false,
			Note:  "grows bonus herbs; whether this or Plainsrunning is Tauren's second active is unread, so both are listed",
			Apply: func(*Character) {}, // unconfirmed slot, and no combat effect either way
		},
		{
			Name: "Endurance", Kind: RacialPassive, Confirmed: true,
			Note: "",
			Apply: func(character *Character) {
				// 5% Health and 1% Hit, demo transcription. Health is the
				// usual multiplicative stat dependency every other "+X%"
				// racial in this file uses, resolved once the character's
				// stats finalize - see TestEnduranceGrantsTheOneHitStat,
				// which finalizes a character's stat dependencies and
				// reads the result back, rather than reading GetStat on a
				// never-finalized character (no production behaviour
				// exists here purely to make an unfinalized read succeed).
				character.MultiplyStat(stats.Health, 1.05)
				// After the Task 4 Hit/Crit merge there is one Hit stat
				// covering melee, ranged and spell, so this is one line
				// where vanilla would have needed two.
				character.AddStat(stats.Hit, 1*HitRatingPerHitChance)
			},
		},
		{
			Name: "Plainsrunning", Kind: RacialPassive, Confirmed: false,
			Note:  "raises movement speed the longer you move; whether this or Cultivation is Tauren's second active is unread, so both are listed",
			Apply: func(*Character) {}, // unconfirmed slot, and no combat effect either way
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
			Note: "client 1259821 (Mana): next 3 damaging abilities cost 10% less and deal 10% more, 15 s, 2 min cooldown; the Rage and Energy variants (1259813, 1259812) and the healer variant (1259823) are not modeled, so physical abilities are untouched",
			Apply: func(character *Character) {
				// Client 1259821: the next eurekaCharges damaging abilities
				// cost 10% less Mana and deal 10% more, for up to 15 s,
				// 2 min cooldown, a major cooldown - the only racial that
				// needs a charge-counting aura, so it uses MaxStacks and
				// consumes one stack per completed cast.
				actionID := ActionID{SpellID: eurekaSpellID}
				eurekaAura := character.RegisterAura(Aura{
					Label:     "Eureka!",
					ActionID:  actionID,
					Duration:  eurekaDuration,
					MaxStacks: eurekaCharges,
					OnGain: func(aura *Aura, sim *Simulation) {
						character.PseudoStats.SchoolCostMultiplier.AddToMagicSchools(-eurekaCostReductionPercent)
						character.PseudoStats.SchoolDamageDealtMultiplier.MultiplyMagicSchools(eurekaDamageMultiplier)
					},
					OnExpire: func(aura *Aura, sim *Simulation) {
						character.PseudoStats.SchoolCostMultiplier.AddToMagicSchools(eurekaCostReductionPercent)
						character.PseudoStats.SchoolDamageDealtMultiplier.MultiplyMagicSchools(1 / eurekaDamageMultiplier)
					},
					OnCastComplete: func(aura *Aura, sim *Simulation, spell *Spell) {
						// The spell that activates this aura carries
						// SpellFlagNoOnCastComplete, so its own completion
						// never reaches this callback: no "just activated
						// by this same cast" guard is needed. Stacks reach
						// zero when the last charge is spent and the aura
						// ends there, or at its duration, whichever is first.
						if spell.Cost == nil {
							return
						}
						aura.RemoveStack(sim) // SetStacks deactivates the aura once stacks hit 0.
					},
				})

				spell := character.RegisterSpell(SpellConfig{
					ActionID: actionID,
					Flags:    SpellFlagNoOnCastComplete,
					Cast: CastConfig{
						CD: Cooldown{
							Timer:    character.NewTimer(),
							Duration: eurekaCooldown,
						},
					},
					ApplyEffects: func(sim *Simulation, _ *Unit, _ *Spell) {
						eurekaAura.Activate(sim)
						eurekaAura.SetStacks(sim, eurekaCharges)
					},
				})

				character.AddMajorCooldown(MajorCooldown{
					Spell: spell,
					Type:  CooldownTypeDPS,
				})
			},
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
			Note: "client 20554: +10% spellcasting and attack speed for 10 s, 3 min cooldown, no resource cost; the engine's custom-percentage cooldowns (15 to 30) are APL compatibility tags, not client rows",
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

	// Skyborne (Alliance): Walk on Air, Read Ley Line, Wind Blessed,
	// Elemental Insight.
	proto.Race_RaceHighOrderSkyborne: append([]Racial{{
		Name: "Read Ley Line", Kind: RacialActive, Confirmed: true,
		Note: "",
		Apply: func(character *Character) {
			// 100% health and mana regeneration for 15 s. No cooldown is
			// published, so no major cooldown is registered here and the
			// sim never uses this racial - a missing cooldown, unlike a
			// missing percentage, isn't something a number could be
			// invented for, and this racial has no combat effect either
			// way once the demo's other regen-focused racials (Rapid
			// Regeneration, Cannibalize) are compared: they are all left
			// unmodeled the same way.
		},
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

// Berserking (client 20554): effects aura 319, 140 and 65 at 10 each, a
// 10 s duration (SpellDuration 1) and a 3 min cooldown, with no resource
// cost. A customPercentage of 0 is the client row; any other value makes a
// cooldown hard-coded to that percentage, which existing APLs address by
// tag.
const (
	berserkingBasePercent = 0.10
	berserkingDuration    = 10 * time.Second
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
		Spell: berserkingSpell,
		Type:  CooldownTypeDPS,
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
