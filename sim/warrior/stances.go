package warrior

import (
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

type Stance uint8

const (
	BattleStance Stance = 1 << iota
	DefensiveStance
	BerserkerStance

	AnyStance = BattleStance | DefensiveStance | BerserkerStance
)

func (stance Stance) Matches(other Stance) bool {
	return (stance & other) != 0
}

var StanceCodes = []int32{SpellCode_WarriorStanceBattle, SpellCode_WarriorStanceDefensive, SpellCode_WarriorStanceBerserker}

const stanceEffectCategory = "Stance"

// improvedTacticalMasteryRagePerPoint is Improved Tactical Mastery's
// "additional 5/10/15 Rage" (Arms node 105954, 3 ranks since
// 1.60.1.70755): the max-rank amount the client's rank spell 12295 e0
// carries (15, the generated ImprovedTacticalMasteryBaseDamage) spread
// over the talent's ranks, which is exactly its trait curve 83313
// (1: 5, 2: 10, 3: 15).
var improvedTacticalMasteryRagePerPoint = ImprovedTacticalMasteryBaseDamage[0][0] / float64(len(TalentSpellIDs["improved_tactical_mastery"]))

// Tactical Mastery (1310185) is baseline in Forever: trained at 14, it
// keeps 10 rage through a stance change before Improved Tactical Mastery.
const (
	tacticalMasteryRetainedRage = 10.0
	tacticalMasteryLevel        = 14
)

// retainedStanceRage is the most rage a stance change keeps.
func retainedStanceRage(level int32, improvedTacticalMasteryPoints int32) float64 {
	retained := improvedTacticalMasteryRagePerPoint * float64(improvedTacticalMasteryPoints)
	if level >= tacticalMasteryLevel {
		retained += tacticalMasteryRetainedRage
	}
	return retained
}

func (warrior *Warrior) StanceMatches(other Stance) bool {
	return warrior.Stance.Matches(other)
}

// stanceRequiredLevel is each stance's own generated required level
// (constants_auto_gen.go): Battle Stance 1, Defensive Stance 10,
// Berserker Stance 30.
var stanceRequiredLevel = map[Stance]int{
	BattleStance:    BattleStanceLevel[0],
	DefensiveStance: DefensiveStanceLevel[0],
	BerserkerStance: BerserkerStanceLevel[0],
}

// knowsStance reports whether the warrior has learned stance by its level.
func (warrior *Warrior) knowsStance(stance Stance) bool {
	return int(warrior.Level) >= stanceRequiredLevel[stance]
}

// defaultStance is the stance a fight opens in when the inputs name none:
// Battle Stance for Arms, Berserker Stance for Fury once Whirlwind is
// learned at 36 (before that Berserker Stance, learned at 30, has no
// ability of its own and costs Fury its Overpower), Defensive Stance
// otherwise.
func (warrior *Warrior) defaultStance() Stance {
	switch warrior.PrimaryTalentTree {
	case ArmsTree:
		return BattleStance
	case FuryTree:
		if warrior.knowsStance(BerserkerStance) && warrior.knowsWhirlwind() {
			return BerserkerStance
		}
		return BattleStance
	default:
		return DefensiveStance
	}
}

func (warrior *Warrior) makeStanceSpell(stance Stance, aura *core.Aura, stanceCD *core.Timer) *WarriorSpell {
	spellCode := map[Stance]int32{
		BattleStance:    SpellCode_WarriorStanceBattle,
		DefensiveStance: SpellCode_WarriorStanceDefensive,
		BerserkerStance: SpellCode_WarriorStanceBerserker,
	}[stance]
	actionID := aura.ActionID
	// Baseline Tactical Mastery's 10 rage plus Improved Tactical
	// Mastery's "additional 5 Rage" a rank (Arms node 105954).
	maxRetainedRage := retainedStanceRage(warrior.Level, warrior.Talents.ImprovedTacticalMastery)
	rageMetrics := warrior.NewRageMetrics(actionID)

	stanceSpell := warrior.RegisterSpell(AnyStance, core.SpellConfig{
		SpellCode: spellCode,
		ActionID:  actionID,
		Flags:     core.SpellFlagAPL,

		RequiredLevel: stanceRequiredLevel[stance],

		Cast: core.CastConfig{
			CD: core.Cooldown{
				Timer:    stanceCD,
				Duration: time.Second,
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return warrior.knowsStance(stance) && !warrior.StanceMatches(stance)
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			if warrior.CurrentRage() > maxRetainedRage {
				warrior.SpendRage(sim, warrior.CurrentRage()-maxRetainedRage, rageMetrics)
			}

			if warrior.WarriorInputs.StanceSnapshot {
				// Delayed, so same-GCD casts are affected by the current aura.
				//  Alternatively, those casts could just (artificially) happen before the stance change.
				core.StartDelayedAction(sim, core.DelayedActionOptions{
					DoAt:     sim.CurrentTime + 10*time.Millisecond,
					OnAction: aura.Activate,
				})
			} else {
				aura.Activate(sim)
			}

			warrior.PreviousStance = warrior.Stance
			warrior.Stance = stance
		},
	})

	warrior.Stances = append(warrior.Stances, stanceSpell)

	return stanceSpell
}

func (warrior *Warrior) registerBattleStanceAura() {
	warrior.BattleStanceAura = warrior.RegisterAura(core.Aura{
		Label:    "Battle Stance",
		ActionID: core.ActionID{SpellID: 2457},
		Duration: core.NeverExpires,
	})
	warrior.BattleStanceAura.NewExclusiveEffect(stanceEffectCategory, true, core.ExclusiveEffect{
		OnGain: func(ee *core.ExclusiveEffect, sim *core.Simulation) {
			ee.Aura.Unit.PseudoStats.ThreatMultiplier *= 0.8
		},
		OnExpire: func(ee *core.ExclusiveEffect, sim *core.Simulation) {
			ee.Aura.Unit.PseudoStats.ThreatMultiplier /= 0.8
		},
	})
}

// Defensive Stance Passive (client spell 7376): "-10%" damage taken and
// damage dealt, and a threat bonus, each an aura on every school.
//
// The threat bonus is 80%: Blizzard's Forever notes of 3 November, "Defensive
// Stance: Threat bonus increased from 30% to 80%", and SpellEffect row 686475
// (aura 10): the 1.60.1.70790 server push 113326 carried it (the shipped row
// read 30), and client 1.60.1.70938's own row reads 80.
// TestShieldAbilitiesMatchClient (spellconst_tank_test.go) checks the
// constant against it.
const (
	defensiveStanceDamageMultiplier = 0.9
	defensiveStanceThreatBonus      = 0.8
)

// defensiveStanceThreat is the stance's threat multiplier. The stance's +80%
// and Defiance's bonus are separate SPELL_AURA_MOD_THREAT effects, which the
// server multiplies (cmangos Aura::HandleModThreat, TrinityCore).
func (warrior *Warrior) defensiveStanceThreat() float64 {
	return (1 + defensiveStanceThreatBonus) * (1 + warrior.defianceThreat())
}

func (warrior *Warrior) registerDefensiveStanceAura() {
	warrior.defensiveStanceThreatMultiplier = warrior.defensiveStanceThreat()

	warrior.DefensiveStanceAura = warrior.RegisterAura(core.Aura{
		Label:    "Defensive Stance",
		ActionID: core.ActionID{SpellID: 71},
		Duration: core.NeverExpires,
	})
	warrior.DefensiveStanceAura.NewExclusiveEffect(stanceEffectCategory, true, core.ExclusiveEffect{
		OnGain: func(ee *core.ExclusiveEffect, sim *core.Simulation) {
			ee.Aura.Unit.PseudoStats.ThreatMultiplier *= warrior.defensiveStanceThreatMultiplier
			ee.Aura.Unit.PseudoStats.DamageDealtMultiplier *= defensiveStanceDamageMultiplier
			ee.Aura.Unit.PseudoStats.DamageTakenMultiplier *= defensiveStanceDamageMultiplier
		},
		OnExpire: func(ee *core.ExclusiveEffect, sim *core.Simulation) {
			ee.Aura.Unit.PseudoStats.ThreatMultiplier /= warrior.defensiveStanceThreatMultiplier
			ee.Aura.Unit.PseudoStats.DamageDealtMultiplier /= defensiveStanceDamageMultiplier
			ee.Aura.Unit.PseudoStats.DamageTakenMultiplier /= defensiveStanceDamageMultiplier
		},
	})
}

func (warrior *Warrior) registerBerserkerStanceAura() {
	warrior.BerserkerStanceAura = warrior.RegisterAura(core.Aura{
		Label:    "Berserker Stance",
		ActionID: core.ActionID{SpellID: 2458},
		Duration: core.NeverExpires,
	})
	warrior.BerserkerStanceAura.NewExclusiveEffect(stanceEffectCategory, true, core.ExclusiveEffect{
		OnGain: func(ee *core.ExclusiveEffect, sim *core.Simulation) {
			ee.Aura.Unit.PseudoStats.ThreatMultiplier *= 0.8
			ee.Aura.Unit.PseudoStats.DamageTakenMultiplier *= 1.1
			ee.Aura.Unit.AddStatDynamic(sim, stats.Crit, core.CritRatingPerCritChance*3)
		},
		OnExpire: func(ee *core.ExclusiveEffect, sim *core.Simulation) {
			ee.Aura.Unit.PseudoStats.ThreatMultiplier /= 0.8
			// Divide, not multiply: this line read `*= 1.1` and so
			// compounded +10% damage taken on every stance change
			// instead of undoing the OnGain. Pre-existing at master;
			// only a stance-dancing tank shows it.
			ee.Aura.Unit.PseudoStats.DamageTakenMultiplier /= 1.1
			ee.Aura.Unit.AddStatDynamic(sim, stats.Crit, -core.CritRatingPerCritChance*3)
		},
	})
}

func (warrior *Warrior) registerStances() {
	warrior.Stances = make([]*WarriorSpell, 0)
	stanceCD := warrior.NewTimer()
	warrior.registerBattleStanceAura()
	warrior.registerDefensiveStanceAura()
	warrior.registerBerserkerStanceAura()
	warrior.BattleStance = warrior.makeStanceSpell(BattleStance, warrior.BattleStanceAura, stanceCD)
	warrior.DefensiveStance = warrior.makeStanceSpell(DefensiveStance, warrior.DefensiveStanceAura, stanceCD)
	warrior.BerserkerStance = warrior.makeStanceSpell(BerserkerStance, warrior.BerserkerStanceAura, stanceCD)
}
