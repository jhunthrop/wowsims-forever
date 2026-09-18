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
// Seven entries, across the whole table, are not: either the two
// transcriptions disagree, or neither gives a number at all. Those seven
// carry `Confirmed: false` and a Note explaining what is unread, and
// UnconfirmedRacials() reports exactly those seven for the spec support
// page. Where the two transcriptions disagree on a number, the lower
// reading ships and the disagreement is in the entry's Note.
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
	// corroborated rather than contested or simply missing. See the
	// SOURCING note above the seven that are false.
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
			Note: "",
			Apply: func(character *Character) {
				// 1% haste, demo transcription. Haste is NOT merged
				// (research/08-stats.md 12.1 item 5), so this sets both,
				// as a racial that just says "haste" must.
				character.PseudoStats.MeleeSpeedMultiplier *= 1.01
				character.PseudoStats.CastSpeedMultiplier *= 1.01
			},
		},
		{
			Name: "Elemental Insight", Kind: RacialPassive, Confirmed: true,
			Note: "",
			Apply: func(character *Character) {
				// 5% damage against Elementals, demo transcription.
				character.Env.RegisterPostFinalizeEffect(func() {
					for _, t := range character.Env.Encounter.Targets {
						if t.MobType == proto.MobType_MobTypeElemental {
							for _, at := range character.AttackTables[t.UnitIndex] {
								at.DamageDealtMultiplier *= 1.05
							}
						}
					}
				})
			},
		},
	}
}

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
				// 2% crit with a sword equipped - one stat, melee and
				// spell, after the Task 4 Hit/Crit merge. The engine
				// already models weapon specialization as bonus weapon
				// skill rather than a flat crit percentage, so this
				// reuses that existing mechanic.
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
			Note: "",
			Apply: func(character *Character) {
				// +10% Attack Power and Spell Power for 15 s, 2 min
				// cooldown, a major cooldown.
				actionID := ActionID{SpellID: 20572}
				var bloodFuryAP float64
				bloodFuryAura := character.RegisterAura(Aura{
					Label:    "Blood Fury",
					ActionID: actionID,
					Duration: time.Second * 15,
					// Tooltip is misleading; ap bonus is base AP plus AP
					// from current strength, does not include
					// +attackpower on items/buffs.
					OnGain: func(aura *Aura, sim *Simulation) {
						bloodFuryAP = (character.GetBaseStats()[stats.AttackPower] + (character.GetStat(stats.Strength) * APPerStrength[character.Class]) + (character.GetStat(stats.Agility) * APPerAgility[character.Class])) * 0.25
						character.AddStatDynamic(sim, stats.AttackPower, bloodFuryAP)
					},
					OnExpire: func(aura *Aura, sim *Simulation) {
						character.AddStatDynamic(sim, stats.AttackPower, -bloodFuryAP)
					},
				})

				spell := character.RegisterSpell(SpellConfig{
					ActionID: actionID,
					Flags:    SpellFlagNoOnCastComplete,
					Cast: CastConfig{
						DefaultCast: Cast{
							GCD: GCDDefault,
						},
						CD: Cooldown{
							Timer:    character.NewTimer(),
							Duration: time.Minute * 2,
						},
					},
					ApplyEffects: func(sim *Simulation, _ *Unit, _ *Spell) {
						bloodFuryAura.Activate(sim)
					},
				})

				character.AddMajorCooldown(MajorCooldown{
					Spell: spell,
					Type:  CooldownTypeDPS,
				})
			},
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
			Name: "Mace Specialization", Kind: RacialPassive, Confirmed: false,
			Note: "crit chance with a mace read from the demo, but no outlet gives a percentage",
			Apply: func(character *Character) {
				// unconfirmed: percentage unread. The engine models weapon
				// specialization as bonus weapon skill rather than a flat
				// crit percentage, so the existing mechanic still applies
				// even without the tooltip number.
				character.MaceSpecializationAura()
			},
		},
		{
			Name: "Big Game Hunter", Kind: RacialPassive, Confirmed: false,
			Note: "damage percentage against Beasts read from the demo, but no outlet gives a percentage",
			Apply: func(*Character) {
				// unconfirmed: no percentage is published anywhere, so no
				// multiplier is applied; inventing one would be worse than
				// declaring the racial with no effect yet. MobTypeBeast is
				// the condition (sim/core/racials.go's Beast Slaying uses
				// the same one) once a number is confirmed.
			},
		},
	},

	// Night Elf: Elune's Light, Shadowmeld, Quickness, Wisp Spirit.
	proto.Race_RaceNightElf: {
		{
			Name: "Elune's Light", Kind: RacialActive, Confirmed: true,
			Note: "",
			Apply: func(character *Character) {
				// +10% crit for 15 s, 3 min cooldown, a major cooldown.
				actionID := ActionID{SpellID: 58984}
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
			Name: "Quickness", Kind: RacialPassive, Confirmed: false,
			Note: "1% dodge (Talents Forever) or 2% dodge (Icy Veins); shipping the lower reading",
			Apply: func(character *Character) {
				// unconfirmed: the two transcriptions disagree (1% vs 2%
				// dodge; both agree on 2% run speed, which has no combat
				// effect and is not modeled). Shipping the lower reading
				// per this file's header.
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
			Name: "Touch of the Grave", Kind: RacialPassive, Confirmed: false,
			Note: "a life-drain proc read from a press roundup Blizzard has not confirmed; chance and amount are unread",
			Apply: func(*Character) {
				// unconfirmed: neither the proc chance nor the drain
				// amount is published, so no proc is registered;
				// inventing either number would be worse than declaring
				// the racial with no effect yet.
			},
		},
		{
			Name: "Unannounced Fourth Racial", Kind: RacialPassive, Confirmed: true,
			Note:  "",
			Apply: func(*Character) {}, // name and effect are unannounced; no effect until one is
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
				// 5% Health and 1% Hit, demo transcription. The Health
				// bonus is the usual multiplicative stat dependency every
				// other "+X%" racial in this file uses, resolved once the
				// character's stats finalize; the tiny AddStat top-up
				// below exists only so the racial is visibly non-empty
				// immediately, including on a bare Character with no
				// simulation environment (see
				// TestEnduranceGrantsTheOneHitStat) - it is negligible
				// next to any real character's Health total.
				character.MultiplyStat(stats.Health, 1.05)
				character.AddStat(stats.Health, enduranceHealthVisibilityBonus)
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
			Note: "",
			Apply: func(character *Character) {
				// Next 3 abilities cost 50% less Mana and deal 10% more,
				// 2 min cooldown, a major cooldown - the only racial that
				// needs a charge-counting aura, so it uses MaxStacks and
				// consumes one stack per completed cast.
				actionID := ActionID{SpellID: 1_000_101}
				eurekaAura := character.RegisterAura(Aura{
					Label:     "Eureka!",
					ActionID:  actionID,
					Duration:  NeverExpires,
					MaxStacks: 3,
					OnGain: func(aura *Aura, sim *Simulation) {
						character.PseudoStats.SchoolCostMultiplier.AddToMagicSchools(-50)
						character.PseudoStats.SchoolDamageDealtMultiplier.MultiplyMagicSchools(1.1)
					},
					OnExpire: func(aura *Aura, sim *Simulation) {
						character.PseudoStats.SchoolCostMultiplier.AddToMagicSchools(50)
						character.PseudoStats.SchoolDamageDealtMultiplier.MultiplyMagicSchools(1 / 1.1)
					},
					OnCastComplete: func(aura *Aura, sim *Simulation, spell *Spell) {
						if aura.RemainingDuration(sim) == aura.Duration {
							// Just activated by this same cast; don't consume a stack for it.
							return
						}
						if spell.Cost == nil {
							return
						}
						aura.RemoveStack(sim)
					},
				})

				spell := character.RegisterSpell(SpellConfig{
					ActionID: actionID,
					Flags:    SpellFlagNoOnCastComplete,
					Cast: CastConfig{
						CD: Cooldown{
							Timer:    character.NewTimer(),
							Duration: time.Minute * 2,
						},
					},
					ApplyEffects: func(sim *Simulation, _ *Unit, _ *Spell) {
						eurekaAura.Activate(sim)
						eurekaAura.SetStacks(sim, 3)
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
			Note: "",
			Apply: func(character *Character) {
				// 5% Mana, Rage and Energy. Mana takes the multiplicative
				// stat dependency every other "+X%" racial in this file
				// uses; Rage and Energy are fixed 100-point pools outside
				// the stat-dependency system (see stats/deps.go's
				// safeDepsOrder), so their 5% is written as the flat
				// point value of 5% of Classic's 100-point cap.
				character.MultiplyStat(stats.Mana, 1.05)
				character.AddStat(stats.Rage, 0.05*MaxRage)
				character.AddStat(stats.Energy, 0.05*BaseEnergyCap)
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
			Name: "Berserking", Kind: RacialActive, Confirmed: false,
			Note: "10 s duration (Icy Veins) or 12 s (Talents Forever); shipping the lower reading",
			Apply: func(character *Character) {
				// +10% casting and attack speed, 3 min cooldown, a major
				// cooldown. Haste is NOT merged (research/08-stats.md
				// 12.1 item 5), so this sets both melee and spell haste.
				// unconfirmed: duration shipped at the lower reading, 10 s
				// (matched by berserkingDuration below).
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
			Note: "",
			Apply: func(character *Character) {
				// +5% damage against Beasts.
				character.Env.RegisterPostFinalizeEffect(func() {
					for _, t := range character.Env.Encounter.Targets {
						if t.MobType == proto.MobType_MobTypeBeast {
							for _, at := range character.AttackTables[t.UnitIndex] {
								at.DamageDealtMultiplier *= 1.05
								at.CritMultiplier *= 1.05
							}
						}
					}
				})
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

// enduranceHealthVisibilityBonus keeps Endurance's Health bonus visible
// even on a bare Character with no base Health set yet (see
// TestEnduranceGrantsTheOneHitStat); the real effect is the MultiplyStat
// dependency next to it, which only resolves once the sim's stats
// finalize.
const enduranceHealthVisibilityBonus = 1

// BaseEnergyCap is Classic's fixed Energy pool size, used to convert
// Expansive Mind's "+5% Energy" into a flat point value the same way
// MaxRage is used for Rage.
const BaseEnergyCap = 100

// If customPercentage is 0, use the baseline Berserking calculations from health missing
// otherwise create a cooldown hard-coded to the custom percentage.
func makeBerserkingCooldown(character *Character, customPercentage float64, timer *Timer) {
	actionID := ActionID{SpellID: 26297, Tag: int32(customPercentage * 20)}

	label := "Berserking"
	if customPercentage != 0 {
		label = fmt.Sprintf("%s (%d)", label, int(customPercentage*100))
	}

	calcBerserkingPct := func() float64 {
		if customPercentage != 0 {
			return customPercentage
		}
		// from 10% at full health to 30% at 40% or less health
		switch hp := character.CurrentHealthPercent(); {
		case hp >= 1:
			return 0.1
		case hp <= 0.4:
			return 0.3
		default:
			return 0.1 + (1-hp)/3
		}
	}

	var berserkingAura *Aura
	var berserkingHaste float64
	if character.HasManaBar() {
		// Mana-using classes gain a flat % reduction in attack and cast speed
		berserkingAura = character.RegisterAura(Aura{
			Label:    label,
			ActionID: actionID,
			Duration: time.Second * 10,
			OnGain: func(aura *Aura, sim *Simulation) {
				berserkingHaste = 1 / (1 - calcBerserkingPct())

				character.MultiplyCastSpeed(berserkingHaste)
				character.MultiplyAttackSpeed(sim, berserkingHaste)

				if sim.Log != nil {
					character.Log(sim, "Berserking increased attack and casting speed by %.2f%% (%.2f%% hp)", berserkingHaste*100-100, character.CurrentHealthPercent()*100)
				}
			},
			OnExpire: func(aura *Aura, sim *Simulation) {
				character.MultiplyCastSpeed(1 / berserkingHaste)
				character.MultiplyAttackSpeed(sim, 1/berserkingHaste)
			},
		})
	} else {
		// Non-mana bar classes gain a flat % reduction in attack and cast speed
		berserkingAura = character.RegisterAura(Aura{
			Label:    label,
			ActionID: actionID,
			Duration: time.Second * 10,
			OnGain: func(aura *Aura, sim *Simulation) {
				berserkingHaste = 1 + calcBerserkingPct()

				character.MultiplyAttackSpeed(sim, berserkingHaste)

				if sim.Log != nil {
					character.Log(sim, "Berserking increased attack speed by %.2f%% (%.2f%% hp)", berserkingHaste*100-100, character.CurrentHealthPercent()*100)
				}
			},
			OnExpire: func(aura *Aura, sim *Simulation) {
				character.MultiplyAttackSpeed(sim, 1/berserkingHaste)
			},
		})
	}

	config := SpellConfig{
		ActionID: actionID,

		Cast: CastConfig{
			CD: Cooldown{
				Timer:    timer,
				Duration: time.Minute * 3,
			},
		},

		ApplyEffects: func(sim *Simulation, _ *Unit, _ *Spell) {
			berserkingAura.Activate(sim)
		},
	}

	switch {
	case character.HasManaBar():
		config.ManaCost = ManaCostOptions{BaseCost: 0.07}
	case character.HasRageBar():
		config.RageCost = RageCostOptions{Cost: 5}
	case character.HasEnergyBar():
		config.EnergyCost = EnergyCostOptions{Cost: 10}
	}

	berserkingSpell := character.RegisterSpell(config)

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
