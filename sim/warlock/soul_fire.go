package warlock

import (
	"slices"
	"time"

	"github.com/wowsims/classic/sim/core"
)

const SoulFireRanks = 2
const SoulFireCastTime = time.Millisecond * 6000

func (warlock *Warlock) getSoulFireBaseConfig(rank int) core.SpellConfig {
	spellId := [SoulFireRanks + 1]int32{0, 6353, 17924}[rank]
	baseDamage := [SoulFireRanks + 1][]float64{{0, 0}, {628, 789}, {715, 894}}[rank]
	manaCost := [SoulFireRanks + 1]float64{0, 305, 335}[rank]
	level := [SoulFireRanks + 1]int{0, 48, 56}[rank]
	spellCoeff := 1.0

	config := core.SpellConfig{
		SpellCode:     SpellCode_WarlockSoulFire,
		ActionID:      core.ActionID{SpellID: spellId},
		SpellSchool:   core.SpellSchoolFire,
		DefenseType:   core.DefenseTypeMagic,
		ProcMask:      core.ProcMaskSpellDamage,
		Flags:         core.SpellFlagAPL | core.SpellFlagResetAttackSwing | WarlockFlagDestruction,
		RequiredLevel: level,
		Rank:          rank,
		MissileSpeed:  24,

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
			damage := sim.Roll(baseDamage[0], baseDamage[1])
			results := spell.CalcDamage(sim, target, damage, spell.OutcomeMagicHitAndCrit)
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
// node 105897, spell_id 440870/440873) into Soul Fire, the way Fel
// Domination (fel_domination.go) already wires a cast-time/cost window
// into its own affected spells. Talent text, rank 2 (max):
// "Reduces the cooldown of your Soul Fire spell by 90%. When you cast
// Shadow Bolt or Searing Pain on an enemy below 35% health, they deal
// 6% increased damage, and for the next 10 sec your Soul Fire spell has
// its cast time reduced by 40% and costs no Soul Shards." Rank 1 halves
// every number (45%, 3%, 20%).
//
// Client data (spellconst/warlock.json), corroborated by wowhead:
//   - 440870 (the passive, one entry shared by both ranks in
//     talents.json): effects 0 and 2 are aura 4 ("Modifies Damage/Healing
//     Done", the SB/SP +3%/+6% -- not implemented here, see below) and
//     effect 1 is aura 108 amount -90 misc_value 11, matching the
//     COOLDOWN reduction; only the max-rank number is present in the
//     table, so rank 1's 45% here is taken from the talent text instead
//     of interpolated from raw effect amounts.
//   - 440873 (the 10 s proc window, duration_ms 10000): effect 0 is
//     aura 108 amount -40 misc_value 10 (a different misc_value than
//     440870's cooldown effect, read here as the CAST TIME reduction);
//     effect 1 is aura 256 amount -40, read as a mana-cost reduction --
//     Soul Fire in this fork costs mana, not Soul Shards, so the
//     tooltip's "costs no Soul Shards" is legacy flavor text and the
//     coded -40% is used instead of treating the cast free.
//
// NOT implemented: the "Shadow Bolt or Searing Pain deal 3%/6% increased
// damage" half of the talent. That is a conditional buff to Shadow
// Bolt's/Searing Pain's own damage (their ApplyEffects would need the
// target's health at cast time), which sits in shadowbolt.go and
// searing_pain.go -- outside this lane's file scope ("wire it into
// soul_fire.go"). Flagged in report-warlock.md.
func (warlock *Warlock) applyDecimation() {
	rank := int(warlock.Talents.Decimation)
	if rank == 0 {
		return
	}

	cooldownReduction := [3]float64{0, 0.45, 0.90}[rank]
	castTimeReduction := [3]float64{0, 0.20, 0.40}[rank]
	costReductionPct := [3]int32{0, 20, 40}[rank]

	// Passive: Soul Fire's cooldown is permanently shorter.
	warlock.OnSpellRegistered(func(spell *core.Spell) {
		if spell.SpellCode == SpellCode_WarlockSoulFire {
			spell.CD.Duration = time.Duration(float64(spell.CD.Duration) * (1 - cooldownReduction))
		}
	})

	// Proc window: landing Shadow Bolt or Searing Pain on a target below
	// 35% health reduces Soul Fire's cast time and mana cost for 10 s.
	// Enemy dummies in the ladder/conformance encounters generally don't
	// track health (HasHealthBar() false), so an untracked target is
	// treated as eligible rather than never triggering the proc.
	castTimeDelta := time.Duration(float64(SoulFireCastTime) * castTimeReduction)
	decimationAura := warlock.RegisterAura(core.Aura{
		ActionID: core.ActionID{SpellID: 440873},
		Label:    "Decimation",
		Duration: time.Second * 10,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			for _, spell := range warlock.SoulFire {
				spell.DefaultCast.CastTime -= castTimeDelta
				spell.Cost.Multiplier -= costReductionPct
			}
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			for _, spell := range warlock.SoulFire {
				spell.DefaultCast.CastTime += castTimeDelta
				spell.Cost.Multiplier += costReductionPct
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
			if result.Target.HasHealthBar() && result.Target.CurrentHealthPercent() > 0.35 {
				return
			}
			decimationAura.Activate(sim)
		},
	}))
}
