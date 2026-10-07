package warlock

import (
	"strconv"
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
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

	baseDamage *= 1 + warlock.shadowMasteryBonus()
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

func (warlock *Warlock) registerCurseOfElementsSpell() {
	playerLevel := warlock.Level
	if playerLevel < 40 {
		return
	}

	warlock.CurseOfElementsAuras = warlock.NewEnemyAuraArray(core.CurseOfElementsAura)

	spellID := map[int32]int32{
		40: 1490,
		50: 11721,
		60: 11722,
	}[playerLevel]

	rank := map[int32]int{
		40: 1,
		50: 2,
		60: 3,
	}[playerLevel]

	manaCost := map[int32]float64{
		40: 100.0,
		50: 150.0,
		60: 200.0,
	}[playerLevel]

	warlock.CurseOfElements = warlock.RegisterSpell(core.SpellConfig{
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
				aura := warlock.CurseOfElementsAuras.Get(target)
				if activeCurse := warlock.ActiveCurseAura.Get(target); activeCurse != nil && activeCurse != aura {
					activeCurse.Deactivate(sim)
				}

				warlock.ActiveCurseAura[target.UnitIndex] = aura
				warlock.ActiveCurseAura.Get(target).Activate(sim)
			}
		},

		RelatedAuras: []core.AuraArray{warlock.CurseOfElementsAuras},
	})
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

func (warlock *Warlock) registerCurseOfDoomSpell() {
	if warlock.Level < 60 {
		return
	}

	warlock.CurseOfDoom = warlock.RegisterSpell(core.SpellConfig{
		SpellCode:   SpellCode_WarlockCurseOfDoom,
		ActionID:    core.ActionID{SpellID: 603},
		SpellSchool: core.SpellSchoolShadow,
		DefenseType: core.DefenseTypeMagic,
		ProcMask:    core.ProcMaskSpellDamage,
		Flags:       core.SpellFlagAPL | WarlockFlagAffliction,

		RequiredLevel: 60,

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
		BonusCoefficient: 1,

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "CurseofDoom",
			},
			NumberOfTicks: 1,
			TickLength:    time.Minute,
			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
				dot.Snapshot(target, 3200, isRollover)
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
