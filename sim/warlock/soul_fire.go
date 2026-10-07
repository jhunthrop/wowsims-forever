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
// decimationDamageMultiplier is Decimation's other half: "Shadow Bolt or
// Searing Pain on an enemy below 35% health, they deal 3%/6% increased
// damage" (rank 1/2) - a flat multiplier on the qualifying cast's OWN
// damage, evaluated at cast time against the target's current health,
// not the 10 s proc window applyDecimation wires above (that window is
// Soul Fire's cast-time/cost discount only). shadowbolt.go and
// searing_pain.go each call this from their own ApplyEffects, the same
// way soul_fire.go's own ApplyEffects would if this talent modified
// Soul Fire's own damage instead.
func (warlock *Warlock) decimationDamageMultiplier(target *core.Unit) float64 {
	rank := int(warlock.Talents.Decimation)
	if rank == 0 {
		return 1
	}
	// Enemy dummies in the ladder/conformance encounters generally
	// don't track health (HasHealthBar() false); treated as eligible
	// the same way applyDecimation's own proc trigger does.
	if target.HasHealthBar() && target.CurrentHealthPercent() > 0.35 {
		return 1
	}
	return 1 + [3]float64{0, 0.03, 0.06}[rank]
}

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
