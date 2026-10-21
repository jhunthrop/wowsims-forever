package warlock

import (
	"strconv"
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

const CurseOfAgonyRanks = 6

// CurseOfAgonyTickDamage is spellconst/warlock.json's own per-tick amount for
// ids 980 through 11713 (rank 6: 46 every 2 s over 12 ticks, no growth),
// the figure the engine's ramp (half, then full, then one and a half times)
// is built on.
var CurseOfAgonyTickDamage = [CurseOfAgonyRanks + 1]clientdamage.Effect{
	{},
	{Amount: 6, SpellLevel: 8},
	{Amount: 10, SpellLevel: 18},
	{Amount: 14, SpellLevel: 28},
	{Amount: 21, SpellLevel: 38},
	{Amount: 33, SpellLevel: 48},
	{Amount: 46, SpellLevel: 58},
}

func (warlock *Warlock) getCurseOfAgonyBaseConfig(rank int) core.SpellConfig {
	numTicks := int32(12)
	tickLength := time.Second * 2

	spellId := [CurseOfAgonyRanks + 1]int32{0, 980, 1014, 6217, 11711, 11712, 11713}[rank]
	spellCoeff := [CurseOfAgonyRanks + 1]float64{0, .133, .133, .133, .133, .133, .133}[rank]
	// FOREVER: Improved Curse of Agony is not in the client's trees.
	// baseDamage := [CurseOfAgonyRanks + 1]float64{0, 7, 15, 27, 42, 65, 87}[rank] * (1 + .03*float64(warlock.Talents.ImprovedCurseOfAgony))
	// baseDamage (the steady-state per-tick value the ramp below scales
	// around) was the classic tooltip's per-tick value until the
	// rotation-accuracy audit compared it against spellconst/
	// warlock.json's own per-rank flat per-tick "amount" (rank 6,
	// 11713, amount 46, period_ms 2000 unchanged) - the same halving
	// shadowbolt.go's comment documents across the rest of the kit.
	damage := CurseOfAgonyTickDamage[rank]
	casterLevel := int(warlock.Level)
	baseDamage := damage.Center(casterLevel)
	manaCost := [CurseOfAgonyRanks + 1]float64{0, 25, 50, 90, 130, 170, 215}[rank]
	level := [CurseOfAgonyRanks + 1]int{0, 8, 18, 28, 38, 48, 58}[rank]

	snapshotBaseDmgNoBonus := 0.0

	return core.SpellConfig{
		SpellCode:        SpellCode_WarlockCurseOfAgony,
		ActionID:         core.ActionID{SpellID: spellId},
		SpellSchool:      core.SpellSchoolShadow,
		DefenseType:      core.DefenseTypeMagic,
		Flags:            core.SpellFlagAPL | core.SpellFlagResetAttackSwing | core.SpellFlagPureDot | WarlockFlagAffliction,
		ProcMask:         core.ProcMaskSpellDamage,
		RequiredLevel:    level,
		Rank:             rank,
		ClientBaseDamage: damage.Range(casterLevel),

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
		},

		CritDamageBonus:  0,
		BonusCoefficient: spellCoeff, // the report compares the spell's, which a pure DoT never reads

		DamageMultiplierAdditive: 1,
		DamageMultiplier:         1,
		ThreatMultiplier:         1,

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "CurseofAgony-" + warlock.Label + strconv.Itoa(rank),
			},
			NumberOfTicks:    numTicks,
			TickLength:       tickLength,
			BonusCoefficient: spellCoeff,

			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
				baseDmg := baseDamage

				if warlock.AmplifyCurseAura.IsActive() {
					baseDmg *= 1.5
					warlock.AmplifyCurseAura.Deactivate(sim)
				}

				// CoA starts with 50% base damage, but bonus from spell power is not changed.
				// Every 4 ticks this base damage is added again, resulting in 150% base damage for the last 4 ticks
				snapshotBaseDmgNoBonus = baseDmg * 0.5

				dot.Snapshot(target, snapshotBaseDmgNoBonus, isRollover)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
				if dot.TickCount%4 == 0 { // CoA ramp up
					dot.SnapshotBaseDamage += snapshotBaseDmgNoBonus
				}
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcOutcome(sim, target, spell.OutcomeMagicHitNoHitCounter)
			if result.Landed() {
				dot := spell.Dot(target)

				if activeCurse := warlock.ActiveCurseAura.Get(target); activeCurse != nil && activeCurse != dot.Aura {
					activeCurse.Deactivate(sim)
				}

				dot.Apply(sim)
				warlock.ActiveCurseAura[target.UnitIndex] = dot.Aura
			}
			spell.DealOutcome(sim, result)
		},
	}
}

func (warlock *Warlock) registerCurseOfAgonySpell() {
	warlock.CurseOfAgony = make([]*core.Spell, 0)
	for rank := 1; rank <= CurseOfAgonyRanks; rank++ {
		config := warlock.getCurseOfAgonyBaseConfig(rank)

		if config.RequiredLevel <= int(warlock.Level) {
			warlock.CurseOfAgony = append(warlock.CurseOfAgony, warlock.GetOrRegisterSpell(config))
		}
	}
}

func (warlock *Warlock) registerCurseOfRecklessnessSpell() {
	playerLevel := warlock.Level

	warlock.CurseOfRecklessnessAuras = warlock.NewEnemyAuraArray(core.CurseOfRecklessnessAura)

	spellID := map[int32]int32{
		25: 704,
		40: 7658,
		50: 7659,
		60: 11717,
	}[playerLevel]

	rank := map[int32]int{
		25: 1,
		40: 2,
		50: 3,
		60: 4,
	}[playerLevel]

	manaCost := map[int32]float64{
		25: 35.0,
		40: 60.0,
		50: 90.0,
		60: 115.0,
	}[playerLevel]

	// spell_level per rank, from spellconst/warlock.json build
	// 1.60.1.70009 (704/7658/7659/11717): conformance golden
	// sim/core/testdata/conformance/warlock.golden.md flagged every
	// rank's RequiredLevel as unset (client 14/28/42/56 vs engine 0).
	requiredLevel := map[int32]int{
		25: 14,
		40: 28,
		50: 42,
		60: 56,
	}[playerLevel]

	warlock.CurseOfRecklessness = warlock.RegisterSpell(core.SpellConfig{
		ActionID:      core.ActionID{SpellID: spellID},
		SpellSchool:   core.SpellSchoolShadow,
		ProcMask:      core.ProcMaskEmpty,
		Flags:         core.SpellFlagAPL | WarlockFlagAffliction,
		Rank:          rank,
		RequiredLevel: requiredLevel,

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
		},

		ThreatMultiplier: 1,
		FlatThreatBonus:  156,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcOutcome(sim, target, spell.OutcomeMagicHitNoHitCounter)
			if result.Landed() {
				aura := warlock.CurseOfRecklessnessAuras.Get(target)
				if activeCurse := warlock.ActiveCurseAura.Get(target); activeCurse != nil && activeCurse != aura {
					activeCurse.Deactivate(sim)
				}

				warlock.ActiveCurseAura[target.UnitIndex] = aura
				warlock.ActiveCurseAura.Get(target).Activate(sim)
			}
		},

		RelatedAuras: []core.AuraArray{warlock.CurseOfRecklessnessAuras},
	})
}

const curseOfTheElementsDuration = 5 * time.Minute

// curseOfTheElementsEffect is what one rank of the Forever curse does, the
// client's "Curses the target for 5 min, reducing Magic resistances by N and
// increasing Magic damage taken by M%. Only one Curse per Warlock can be active
// on any one target." (spellconst/warlock.json ids 440892, 1311676, 1311677 and
// 1311680: effect 0 is aura 22 and effect 1 aura 87, both over misc value 126.)
// Ids, levels and costs are constants_auto_gen.go's CurseOfTheElements* arrays.
type curseOfTheElementsEffect struct {
	resistance     float64 // magic resistance removed
	damageTakenPct float64 // magic damage taken increase, in percent
}

var curseOfTheElementsEffects = [CurseOfTheElementsRanks + 1]curseOfTheElementsEffect{
	{},
	{resistance: 30, damageTakenPct: 4},
	{resistance: 45, damageTakenPct: 6},
	{resistance: 60, damageTakenPct: 8},
	{resistance: 75, damageTakenPct: 10},
}

// curseOfTheElementsSchools is every school the client's misc value 126 names,
// which is all magic: the vanilla curse stopped at fire and frost.
var curseOfTheElementsSchools = []stats.SchoolIndex{
	stats.SchoolIndexArcane, stats.SchoolIndexFire, stats.SchoolIndexFrost,
	stats.SchoolIndexHoly, stats.SchoolIndexNature, stats.SchoolIndexShadow,
}

func (warlock *Warlock) newCurseOfTheElementsAura(rank int) func(*core.Unit) *core.Aura {
	effect := curseOfTheElementsEffects[rank]
	multiplier := 1 + effect.damageTakenPct/100

	return func(target *core.Unit) *core.Aura {
		return target.GetOrRegisterAura(core.Aura{
			Label:    "Curse of the Elements-" + warlock.Label + strconv.Itoa(rank),
			ActionID: core.ActionID{SpellID: CurseOfTheElementsSpellId[rank]},
			Duration: curseOfTheElementsDuration,
			OnGain: func(aura *core.Aura, sim *core.Simulation) {
				for _, school := range curseOfTheElementsSchools {
					aura.Unit.PseudoStats.SchoolDamageTakenMultiplier[school] *= multiplier
				}
				aura.Unit.AddResistancesDynamic(sim, -effect.resistance)
			},
			OnExpire: func(aura *core.Aura, sim *core.Simulation) {
				for _, school := range curseOfTheElementsSchools {
					aura.Unit.PseudoStats.SchoolDamageTakenMultiplier[school] /= multiplier
				}
				aura.Unit.AddResistancesDynamic(sim, effect.resistance)
			},
		})
	}
}

func (warlock *Warlock) registerCurseOfElementsSpell() {
	warlock.CurseOfElements = make([]*core.Spell, 0, CurseOfTheElementsRanks)
	for rank := 1; rank <= CurseOfTheElementsRanks; rank++ {
		if CurseOfTheElementsLevel[rank] > int(warlock.Level) {
			continue
		}
		warlock.CurseOfElements = append(warlock.CurseOfElements, warlock.registerCurseDebuff(core.SpellConfig{
			ActionID:      core.ActionID{SpellID: CurseOfTheElementsSpellId[rank]},
			Rank:          rank,
			RequiredLevel: CurseOfTheElementsLevel[rank],
			ManaCost:      core.ManaCostOptions{FlatCost: CurseOfTheElementsManaCost[rank]},
		}, warlock.newCurseOfTheElementsAura(rank)))
	}
}

// registerCurseDebuff registers an instant, aura-only curse: base is the
// per-curse part of its config (id, rank, level, cost) and newAura builds the
// debuff its hit puts on the target, replacing whichever curse was there.
func (warlock *Warlock) registerCurseDebuff(base core.SpellConfig, newAura func(*core.Unit) *core.Aura) *core.Spell {
	auras := warlock.NewEnemyAuraArray(newAura)

	base.SpellSchool = core.SpellSchoolShadow
	base.ProcMask = core.ProcMaskEmpty
	base.Flags = core.SpellFlagAPL | WarlockFlagAffliction
	base.Cast = core.CastConfig{DefaultCast: core.Cast{GCD: core.GCDDefault}}
	base.ThreatMultiplier = 1
	base.FlatThreatBonus = 156
	base.RelatedAuras = []core.AuraArray{auras}
	base.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		result := spell.CalcOutcome(sim, target, spell.OutcomeMagicHitNoHitCounter)
		if result.Landed() {
			aura := auras.Get(target)
			if activeCurse := warlock.ActiveCurseAura.Get(target); activeCurse != nil && activeCurse != aura {
				activeCurse.Deactivate(sim)
			}

			warlock.ActiveCurseAura[target.UnitIndex] = aura
			aura.Activate(sim)
		}
	}
	return warlock.RegisterSpell(base)
}

func (warlock *Warlock) registerCurseOfShadowSpell() {
	playerLevel := warlock.Level
	if playerLevel < 50 {
		return
	}

	warlock.CurseOfShadowAuras = warlock.NewEnemyAuraArray(core.CurseOfShadowAura)

	spellID := map[int32]int32{
		50: 17862,
		60: 17937,
	}[playerLevel]

	rank := map[int32]int{
		50: 1,
		60: 2,
	}[playerLevel]

	manaCost := map[int32]float64{
		50: 150.0,
		60: 200.0,
	}[playerLevel]

	warlock.CurseOfShadow = warlock.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: spellID},
		SpellSchool: core.SpellSchoolShadow,
		ProcMask:    core.ProcMaskEmpty,
		Flags:       core.SpellFlagAPL | WarlockFlagAffliction,
		Rank:        rank,

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
		},

		ThreatMultiplier: 1,
		FlatThreatBonus:  156,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcOutcome(sim, target, spell.OutcomeMagicHitNoHitCounter)
			if result.Landed() {
				aura := warlock.CurseOfShadowAuras.Get(target)
				if activeCurse := warlock.ActiveCurseAura.Get(target); activeCurse != nil && activeCurse != aura {
					activeCurse.Deactivate(sim)
				}

				warlock.ActiveCurseAura[target.UnitIndex] = aura
				warlock.ActiveCurseAura.Get(target).Activate(sim)
			}
		},

		RelatedAuras: []core.AuraArray{warlock.CurseOfShadowAuras},
	})
}

func (warlock *Warlock) registerAmplifyCurseSpell() {
	if !warlock.Talents.AmplifyCurse {
		return
	}

	actionID := core.ActionID{SpellID: 18288}

	warlock.AmplifyCurseAura = warlock.GetOrRegisterAura(core.Aura{
		Label:    "Amplify Curse",
		ActionID: actionID,
		Duration: time.Second * 30,
	})

	warlock.AmplifyCurse = warlock.GetOrRegisterSpell(core.SpellConfig{
		ActionID:        actionID,
		SpellSchool:     core.SpellSchoolShadow,
		Flags:           core.SpellFlagAPL | WarlockFlagAffliction,
		RelatedSelfBuff: warlock.AmplifyCurseAura,

		Cast: core.CastConfig{
			CD: core.Cooldown{
				Timer:    warlock.NewTimer(),
				Duration: 3 * time.Minute,
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			warlock.AmplifyCurseAura.Activate(sim)
		},
	})
}

// BaneOfDoomDamage is spellconst/warlock.json's row 603, the id the client
// teaches: 1742 shadow damage after one minute, coefficient 4.0. (Row 449432,
// "Curse of Doom", states 3200 at 1.0 but has no learn row.)
var BaneOfDoomDamage = []clientdamage.Effect{{Amount: 1742, SpellLevel: 60}}

const baneOfDoomCoefficient = 4.0

func (warlock *Warlock) registerCurseOfDoomSpell() {
	if warlock.Level < 60 {
		return
	}

	damage := BaneOfDoomDamage[0]
	casterLevel := int(warlock.Level)

	warlock.CurseOfDoom = warlock.RegisterSpell(core.SpellConfig{
		SpellCode:   SpellCode_WarlockCurseOfDoom,
		ActionID:    core.ActionID{SpellID: 603},
		SpellSchool: core.SpellSchoolShadow,
		DefenseType: core.DefenseTypeMagic,
		ProcMask:    core.ProcMaskSpellDamage,
		Flags:       core.SpellFlagAPL | WarlockFlagAffliction,

		RequiredLevel:    60,
		ClientBaseDamage: damage.Range(casterLevel),

		ManaCost: core.ManaCostOptions{
			FlatCost: 300,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			CD: core.Cooldown{
				Timer:    warlock.NewTimer(),
				Duration: time.Second * 60,
			},
		},

		CritDamageBonus: 0,

		DamageMultiplier: 1,
		// FOREVER: Improved Drain Soul is not in the client's trees.
		// ThreatMultiplier: 1 - 0.1*float64(warlock.Talents.ImprovedDrainSoul),
		ThreatMultiplier: 1,
		FlatThreatBonus:  160,
		BonusCoefficient: baneOfDoomCoefficient,

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "CurseofDoom",
			},
			NumberOfTicks:    1,
			TickLength:       time.Minute,
			BonusCoefficient: baneOfDoomCoefficient,
			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
				dot.Snapshot(target, damage.Center(casterLevel), isRollover)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcOutcome(sim, target, spell.OutcomeMagicHitNoHitCounter)
			if result.Landed() {
				dot := spell.Dot(target)
				if activeCurse := warlock.ActiveCurseAura.Get(target); activeCurse != nil && activeCurse != dot.Aura {
					activeCurse.Deactivate(sim)
				}

				dot.Apply(sim)
				warlock.ActiveCurseAura[target.UnitIndex] = dot.Aura
			}
		},
	})
}
