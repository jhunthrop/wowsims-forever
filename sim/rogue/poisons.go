package rogue

import (
	"strconv"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

/**
Instant Poison: 20% proc chance
25: 22 +/- 3 damage, 8679 ID, 40 charges
40: 50 +/- 6 damage, 8688 ID, 70 charges
50: 76 +/- 9 damage, 11338 ID, 85 charges
60: 130 =/- 18 damage, 11340 ID, 115 charges

Deadly Poison: 30% proc chance, 5 stacks
40: 52 damage, 2824 ID, 75 charges
50: 80 damage, 11355 ID, 90 charges
60: 108 damage, 11356 ID, 105 charges (Rank 4, Rank 5 is by book)

Wound Poison: 30% proc chance, 5 stacks
25: x damage, x ID (none, first rank is level 32)
40: -75 healing, 11325 ID, 75 charges (Rank 2)
50: -105 healing, 13226 ID, 90 charges (Rank 3)
60: -135 healing, 13227 ID, 105 charges (Rank 4)
*/

// TODO: Add charges to poisons

type PoisonProcSource int

const (
	NormalProc PoisonProcSource = iota
)

func (rogue *Rogue) GetInstantPoisonProcChance() float64 {
	return 0.2 + rogue.improvedPoisons() + rogue.additivePoisonBonusChance
}

func (rogue *Rogue) GetDeadlyPoisonProcChance() float64 {
	return 0.3 + rogue.improvedPoisons() + rogue.additivePoisonBonusChance
}

func (rogue *Rogue) GetWoundPoisonProcChance() float64 {
	return 0.3 + rogue.improvedPoisons() + rogue.additivePoisonBonusChance
}

func (rogue *Rogue) improvedPoisons() float64 {
	return []float64{0, 0.02, 0.04, 0.06, 0.08, 0.1}[rogue.Talents.ImprovedPoisons]
}

func (rogue *Rogue) getPoisonDamageMultiplier() float64 {
	return []float64{1, 1.04, 1.08, 1.12, 1.16, 1.2}[rogue.Talents.VilePoisons]
}

///////////////////////////////////////////////////////////////////////////
//                               Apply Poisons
///////////////////////////////////////////////////////////////////////////

func (rogue *Rogue) applyPoisons() {
	rogue.applyDeadlyPoison()
	rogue.applyInstantPoison()
	rogue.applyWoundPoison()
}

// Apply Instant Poison to weapon and enable procs
func (rogue *Rogue) applyInstantPoison() {
	procMask := rogue.getImbueProcMask(proto.WeaponImbue_InstantPoison)
	if procMask == core.ProcMaskUnknown {
		return
	}

	rogue.RegisterAura(core.Aura{
		Label:    "Instant Poison",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if !result.Landed() || !spell.ProcMask.Matches(procMask) {
				return
			}

			if sim.RandomFloat("Instant Poison") < rogue.GetInstantPoisonProcChance() {
				rogue.InstantPoison.Cast(sim, result.Target)
			}
		},
	})
}

// Apply Deadly Poison to weapon and enable procs
func (rogue *Rogue) applyDeadlyPoison() {
	procMask := rogue.getImbueProcMask(proto.WeaponImbue_DeadlyPoison)
	if procMask == core.ProcMaskUnknown {
		return
	}

	rogue.RegisterAura(core.Aura{
		Label:    "Deadly Poison",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if !result.Landed() || !spell.ProcMask.Matches(procMask) {
				return
			}
			if sim.RandomFloat("Deadly Poison") < rogue.GetDeadlyPoisonProcChance() {
				rogue.DeadlyPoison.Cast(sim, result.Target)
			}
		},
	})
}

// Apply Wound Poison to weapon and enable procs
func (rogue *Rogue) applyWoundPoison() {
	procMask := rogue.getImbueProcMask(proto.WeaponImbue_WoundPoison)
	if procMask == core.ProcMaskUnknown {
		return
	}

	rogue.RegisterAura(core.Aura{
		Label:    "Wound Poison",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if !result.Landed() || !spell.ProcMask.Matches(procMask) {
				return
			}

			if sim.RandomFloat("Wound Poison") < rogue.GetWoundPoisonProcChance() {
				rogue.WoundPoison.Cast(sim, result.Target)
			}
		},
	})
}

///////////////////////////////////////////////////////////////////////////
//                              Register Poisons
///////////////////////////////////////////////////////////////////////////

func (rogue *Rogue) registerInstantPoisonSpell() {
	rogue.InstantPoison = rogue.makeInstantPoison()
}

// deadlyPoisonLearnLevels are Deadly Poison's five rank learn levels;
// source: 1.60.1.70009 client spell data. Each rank is a separate name
// ("Deadly Poison", "Deadly Poison II", ... "V"), each with several
// weapon-slot-variant ids at the same level; deadlyPoisonSpellID picks one
// representative id per rank. Rank 5 only exists with AQ content, same as
// the old code's ternary (which reused rank 4's id, 11356, for the
// pre-AQ 60 case since only 4 ranks existed then).
var deadlyPoisonLearnLevels = []int{30, 38, 46, 54, 60}

// deadlyPoisonSpellID is Deadly Poison's rank -> spell id, index 0 unused.
var deadlyPoisonSpellID = [6]int32{0, 2823, 2824, 11355, 11356, core.TernaryInt32(core.IncludeAQ, 25347, 11356)}

// deadlyPoisonBaseDamageTick is Deadly Poison's rank -> tick damage, index 0
// unused. The old bracket-25 entry (2823/9) was for a level below Deadly
// Poison's real learn level (30) and cannot be rank 1's value; rank 1
// instead carries rank 2's number backward, and rank 4 carries rank 3's
// number forward, until real numbers are sourced.
var deadlyPoisonBaseDamageTick = [6]float64{0, 13, 13, 20, 20, core.TernaryFloat64(core.IncludeAQ, 34, 27)}

func (rogue *Rogue) registerDeadlyPoisonSpell() {
	rank := core.HighestRankAtLevel(deadlyPoisonLearnLevels, rogue.Level)
	if rank == 0 {
		return
	}

	baseDamageTick := deadlyPoisonBaseDamageTick[rank]
	spellID := deadlyPoisonSpellID[rank]

	rogue.deadlyPoisonTick = rogue.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: spellID, Tag: 100},
		SpellSchool: core.SpellSchoolNature,
		DefenseType: core.DefenseTypeMagic,
		ProcMask:    core.ProcMaskSpellDamageProc,
		Flags:       core.SpellFlagPoison | core.SpellFlagPassiveSpell | SpellFlagRoguePoison,

		DamageMultiplier: rogue.getPoisonDamageMultiplier(),
		ThreatMultiplier: 1,

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label:     "DeadlyPoison",
				MaxStacks: 5,
				Duration:  time.Second * 12,
			},
			NumberOfTicks: 4,
			TickLength:    time.Second * 3,

			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, applyStack bool) {
				if !applyStack {
					return
				}

				// only the first stack snapshots the multiplier
				if dot.GetStacks() == 1 {
					attackTable := dot.Spell.Unit.AttackTables[target.UnitIndex][dot.Spell.CastType]
					dot.SnapshotAttackerMultiplier = dot.Spell.AttackerDamageMultiplier(attackTable, true)
					dot.SnapshotBaseDamage = 0
				}

				dot.SnapshotBaseDamage += baseDamageTick
			},

			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
			},
		},
	})

	rogue.DeadlyPoison = rogue.makeDeadlyPoison()
}

func (rogue *Rogue) registerWoundPoisonSpell() {
	woundPoisonDebuffAura := core.Aura{
		Label:     "WoundPoison-" + strconv.Itoa(int(rogue.Index)),
		ActionID:  core.ActionID{SpellID: 13219},
		MaxStacks: 5,
		Duration:  time.Second * 15,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			// all healing effects used on target reduced by x, stacks 5 times
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			// undo reduced healing effects used on targets
		},
	}

	rogue.woundPoisonDebuffAuras = rogue.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
		return target.RegisterAura(woundPoisonDebuffAura)
	})
	rogue.WoundPoison = rogue.makeWoundPoison()
}

///////////////////////////////////////////////////////////////////////////
//                              Make Poisons
///////////////////////////////////////////////////////////////////////////

// instantPoisonLearnLevels are Instant Poison's six rank learn levels;
// source: 1.60.1.70009 client spell data. Each rank is a separate name
// ("Instant Poison", "Instant Poison II", ... "VI"), each with three
// weapon-slot-variant ids at the same level; instantPoisonSpellID picks one
// representative id per rank.
var instantPoisonLearnLevels = []int{20, 28, 36, 44, 52, 60}

// instantPoisonSpellID is Instant Poison's rank -> spell id, index 0 unused.
var instantPoisonSpellID = [7]int32{0, 8679, 8686, 8688, 11338, 11339, 11340}

// instantPoisonBaseDamageByLevel/DamageVariance are Instant Poison's rank ->
// damage terms, index 0 unused. Only ranks 1, 3, 4 and 6 have a tuned value
// in this file; rank 2 carries rank 1's terms forward, and rank 5 carries
// rank 4's terms forward, until real numbers are sourced.
var instantPoisonBaseDamageByLevel = [7]float64{0, 19, 19, 44, 67, 67, 112}
var instantPoisonDamageVariance = [7]float64{0, 6, 6, 12, 18, 18, 36}

// Make a source based variant of Instant Poison
func (rogue *Rogue) makeInstantPoison() *core.Spell {
	rank := core.HighestRankAtLevel(instantPoisonLearnLevels, rogue.Level)
	if rank == 0 {
		return nil
	}

	baseDamageByLevel := instantPoisonBaseDamageByLevel[rank]
	damageVariance := instantPoisonDamageVariance[rank]
	spellID := instantPoisonSpellID[rank]

	return rogue.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: spellID},
		SpellSchool: core.SpellSchoolNature,
		DefenseType: core.DefenseTypeMagic,
		ProcMask:    core.ProcMaskSpellDamageProc,
		Flags:       core.SpellFlagPoison | core.SpellFlagPassiveSpell | SpellFlagRoguePoison,

		DamageMultiplier: rogue.getPoisonDamageMultiplier(),
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := sim.Roll(baseDamageByLevel, baseDamageByLevel+damageVariance)
			spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)
		},
	})
}

func (rogue *Rogue) makeDeadlyPoison() *core.Spell {
	return rogue.RegisterSpell(core.SpellConfig{
		ActionID: core.ActionID{SpellID: rogue.deadlyPoisonTick.SpellID},
		Flags:    core.SpellFlagPoison | core.SpellFlagPassiveSpell | SpellFlagRoguePoison,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcAndDealOutcome(sim, target, spell.OutcomeMagicHit)

			if !result.Landed() {
				return
			}

			dot := rogue.deadlyPoisonTick.Dot(target)

			dot.ApplyOrRefresh(sim)
			if dot.GetStacks() < dot.MaxStacks {
				dot.AddStack(sim)
				// snapshotting only takes place when adding a stack
				dot.TakeSnapshot(sim, true)
			}
		},
	})
}

// Make a source based variant of Wound Poison
func (rogue *Rogue) makeWoundPoison() *core.Spell {
	return rogue.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: 13219},
		SpellSchool: core.SpellSchoolNature,
		DefenseType: core.DefenseTypeMagic,
		ProcMask:    core.ProcMaskSpellDamageProc,
		Flags:       core.SpellFlagPoison | core.SpellFlagPassiveSpell | SpellFlagRoguePoison,

		DamageMultiplier: rogue.getPoisonDamageMultiplier(),
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcAndDealOutcome(sim, target, spell.OutcomeMagicHit)

			if !result.Landed() {
				return
			}

			aura := rogue.woundPoisonDebuffAuras.Get(target)
			if !aura.IsActive() {
				aura.Activate(sim)
				aura.SetStacks(sim, 1)
				return
			}

			if aura.GetStacks() < 5 {
				aura.Refresh(sim)
				aura.AddStack(sim)
				return
			}
			aura.Refresh(sim)
		},
	})
}
