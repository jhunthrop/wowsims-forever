package warlock

import (
	"slices"
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

const SoulFireRanks = 2
const SoulFireCastTime = time.Millisecond * 6000

// SoulFireDamage is spellconst/warlock.json's own roll for ids 6353 and
// 17924 (rank 2 rolls 389.3-487.9 at level 60: a centre of 431 at its own
// level, growing 1.9 a level, 0.225 wide; sp_coefficient 1.0), where the
// classic tooltip roll was {715, 894} - see shadowbolt.go's comment.
var SoulFireDamage = [SoulFireRanks + 1]clientdamage.Effect{
	{},
	{Amount: 377, Variance: 0.227596, PerLevel: 1.7, SpellLevel: 48, MaxLevel: 54},
	{Amount: 431, Variance: 0.224747, PerLevel: 1.9, SpellLevel: 56, MaxLevel: 62},
}

func (warlock *Warlock) getSoulFireBaseConfig(rank int) core.SpellConfig {
	spellId := [SoulFireRanks + 1]int32{0, 6353, 17924}[rank]
	damage := SoulFireDamage[rank]
	casterLevel := int(warlock.Level)
	manaCost := [SoulFireRanks + 1]float64{0, 305, 335}[rank]
	level := [SoulFireRanks + 1]int{0, 48, 56}[rank]
	spellCoeff := 1.0

	config := core.SpellConfig{
		SpellCode:        SpellCode_WarlockSoulFire,
		ActionID:         core.ActionID{SpellID: spellId},
		SpellSchool:      core.SpellSchoolFire,
		DefenseType:      core.DefenseTypeMagic,
		ProcMask:         core.ProcMaskSpellDamage,
		Flags:            core.SpellFlagAPL | core.SpellFlagResetAttackSwing | WarlockFlagDestruction,
		RequiredLevel:    level,
		Rank:             rank,
		ClientBaseDamage: damage.Range(casterLevel),
		MissileSpeed:     24,

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD:      core.GCDDefault,
				CastTime: SoulFireCastTime,
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: spellCoeff,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			results := spell.CalcDamage(sim, target, damage.Roll(sim, casterLevel), spell.OutcomeMagicHitAndCrit)
			spell.WaitTravelTime(sim, func(s *core.Simulation) {
				spell.DealDamage(sim, results)
			})
		},
	}

	config.Cast.CD = core.Cooldown{
		Timer:    warlock.NewTimer(),
		Duration: time.Minute,
	}

	return config
}

func (warlock *Warlock) registerSoulFireSpell() {
	warlock.SoulFire = make([]*core.Spell, 0)
	for rank := 1; rank <= SoulFireRanks; rank++ {
		config := warlock.getSoulFireBaseConfig(rank)

		if config.RequiredLevel <= int(warlock.Level) {
			warlock.SoulFire = append(warlock.SoulFire, warlock.GetOrRegisterSpell(config))
		}
	}

	warlock.applyDecimation()
}

// applyDecimation wires the Demonology talent Decimation (talents/warlock.json
// node 105897, spell_id 440870/440873) into Soul Fire. Talent text, rank 2:
// "Reduces the cooldown of your Soul Fire spell by 90%. When you cast Shadow
// Bolt or Searing Pain on an enemy below 35% health, they deal 6% increased
// damage, and for the next 10 sec your Soul Fire spell has its cast time
// reduced by 40% and costs no Soul Shards." Rank 1 halves every number.
//
// Client data (1.60.1.70707):
//   - 440870 effect 1 is aura 108 -90 misc 11 (cooldown), on Soul Fire's mask.
//   - 440873 (the 10 s buff) effect 0 is aura 108 -40 misc 10, a percent
//     cast-time modifier: it scales the cast time after flat modifiers, so
//     Bane's -2 s comes first and Soul Fire casts in 2.4 s. Effect 1 is aura
//     256, no reagent use: the free Soul Shard. The engine keeps no shard
//     count, so this half changes nothing here, and the mana cost stands.
//
// decimationDamageMultiplier is the 3%/6% on the qualifying cast's own
// damage, read at cast time; shadowbolt.go and searing_pain.go call it.
func (warlock *Warlock) decimationDamageMultiplier(sim *core.Simulation, target *core.Unit) float64 {
	rank := int(warlock.Talents.Decimation)
	if rank == 0 || !decimationTargetEligible(sim, target) {
		return 1
	}
	return 1 + [3]float64{0, 0.03, 0.06}[rank]
}

// decimationHealthThreshold is 440870 effect 2's 35.
const decimationHealthThreshold = 0.35

// decimationTargetEligible reports whether target is below 35% health. A
// target with no health bar (every encounter dummy) is below it in the
// encounter's execute window.
func decimationTargetEligible(sim *core.Simulation, target *core.Unit) bool {
	if target.HasHealthBar() {
		return target.CurrentHealthPercent() < decimationHealthThreshold
	}
	return sim.IsExecutePhase35()
}

func (warlock *Warlock) applyDecimation() {
	rank := int(warlock.Talents.Decimation)
	if rank == 0 {
		return
	}

	cooldownReduction := [3]float64{0, 0.45, 0.90}[rank]
	castTimeReduction := [3]float64{0, 0.20, 0.40}[rank]

	// Passive: Soul Fire's cooldown is permanently shorter.
	warlock.OnSpellRegistered(func(spell *core.Spell) {
		if spell.SpellCode == SpellCode_WarlockSoulFire {
			spell.CD.Duration = time.Duration(float64(spell.CD.Duration) * (1 - cooldownReduction))
		}
	})

	decimationAura := warlock.RegisterAura(core.Aura{
		ActionID: core.ActionID{SpellID: 440873},
		Label:    "Decimation",
		Duration: time.Second * 10,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			for _, spell := range warlock.SoulFire {
				spell.CastTimeMultiplier -= castTimeReduction
			}
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			for _, spell := range warlock.SoulFire {
				spell.CastTimeMultiplier += castTimeReduction
			}
		},
	})

	decimationTriggerSpellCodes := []int32{SpellCode_WarlockShadowBolt, SpellCode_WarlockSearingPain}
	core.MakePermanent(warlock.RegisterAura(core.Aura{
		Label: "Decimation Trigger",
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if !result.Landed() || !slices.Contains(decimationTriggerSpellCodes, spell.SpellCode) {
				return
			}
			if !decimationTargetEligible(sim, result.Target) {
				return
			}
			decimationAura.Activate(sim)
		},
	}))
}
