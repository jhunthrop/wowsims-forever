package feral

import (
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/druid"
)

type FeralDruidRotation struct {
	PrimaryBuilder     *druid.DruidSpell
	MinCombosForRip    int32
	MaxWaitTime        time.Duration
	MaintainFaerieFire bool
	UseShredTrick      bool
	UseRipTrick        bool
}

func (cat *FeralDruid) OnGCDReady(sim *core.Simulation) {
	if !cat.GCD.IsReady(sim) {
		return
	}

	cat.missChance = cat.MissChance()
	cat.bleedAura = cat.CurrentTarget.GetExclusiveEffectCategory(core.BleedEffectCategory).GetActiveAura()

	if cat.preRotationCleanup(sim) {
		waiting := false
		nextAction := time.Duration(0)
		// if cat.Rotation.RotationType == proto.FeralDruid_Rotation_SingleTarget {
		// 	valid, nextAction = cat.doRotation(sim)
		// } else {
		// 	valid, nextAction = cat.doAoeRotation(sim)
		// }
		waiting, nextAction = cat.doRotation(sim)
		if waiting {
			cat.postRotation(sim, nextAction)
		}
	}

	// Replace gcd event with our own if we casted a spell
	if !cat.GCD.IsReady(sim) {
		nextGcd := cat.NextGCDAt()
		cat.CancelGCDTimer(sim)

		cat.NextRotationAction(sim, nextGcd)
	}
}

func (cat *FeralDruid) NextRotationAction(sim *core.Simulation, kickAt time.Duration) {
	if cat.rotationAction != nil {
		cat.rotationAction.Cancel(sim)
	}

	cat.rotationAction = &core.PendingAction{
		Priority:     core.ActionPriorityGCD,
		OnAction:     cat.OnGCDReady,
		NextActionAt: kickAt,
	}

	sim.AddPendingAction(cat.rotationAction)
}

func (cat *FeralDruid) tryPowershift(sim *core.Simulation) {
	if cat.CatForm == nil { // below Cat Form's trainer level
		return
	}
	if cat.InForm(druid.Cat) {
		cat.CancelShapeshift(sim)
		cat.TryUseCooldowns(sim)
	}

	if cat.Rotation.MaintainFaerieFire && cat.ShouldFaerieFire(sim, cat.CurrentTarget) && (cat.CurrentMana() >= cat.CatForm.DefaultCast.Cost+cat.FaerieFire.DefaultCast.Cost) {
		cat.FaerieFire.Cast(sim, cat.CurrentTarget)
	} else {
		cat.CatForm.Cast(sim, nil)
		cat.readyToShift = false
		cat.lastShift = sim.CurrentTime
	}
}

// shiftingPowerFits is whether casting Shifting Power now would take its full
// energy: the spell is learned, ready and affordable, and the bar has room.
func (cat *FeralDruid) shiftingPowerFits(sim *core.Simulation) bool {
	return cat.ShiftingPower != nil &&
		cat.ShiftingPower.CanCast(sim, cat.CurrentTarget) &&
		cat.CurrentEnergy()+druid.ShiftingPowerEnergy <= cat.MaxEnergy()
}

func (cat *FeralDruid) maxShifts() int32 {
	if cat.CatForm == nil {
		return 0
	}
	return max(int32(cat.MaxMana()/cat.CatForm.DefaultCast.Cost), 0)
}

func (cat *FeralDruid) numShiftsRemaining() int32 {
	if cat.CatForm == nil {
		return 0
	}
	return max(int32(cat.CurrentMana()/cat.CatForm.DefaultCast.Cost), 0)
}

func (cat *FeralDruid) timeToCast(numSpecials int32) time.Duration {
	// Rough calculation, won't be exact! Intended to skew conservatively.
	totalCasts := float64(numSpecials)
	numPowershiftedSpecials := min(totalCasts, float64(cat.numShiftsRemaining()*2*cat.Talents.Furor)/5.0)
	numOomSpecials := totalCasts - numPowershiftedSpecials
	return core.DurationFromSeconds(numPowershiftedSpecials*2.0 + numOomSpecials*4.0)
}

func (cat *FeralDruid) canRip(sim *core.Simulation, isTrick bool) bool {
	if cat.Rip.CurDot().IsActive() {
		return false
	}

	fightDur := sim.GetRemainingDuration()

	if cat.ComboPoints() == 5 {
		// 5CP rip is worth it with 4 ticks
		if fightDur <= time.Second*8 {
			return false
		} else {
			return true
		}
	} else {
		// Otherwise require 5 ticks
		if fightDur <= time.Second*10 {
			return false
		}
	}

	// Caller decided that we should "rip trick".
	if isTrick {
		return true
	}

	return false
}

func (cat *FeralDruid) preRotationCleanup(sim *core.Simulation) bool {
	// If we previously decided to shift, then execute the shift now once
	// the input delay is over.
	if cat.readyToShift {
		cat.tryPowershift(sim)
		return false
	}

	return true
}

func (cat *FeralDruid) postRotation(sim *core.Simulation, nextAction time.Duration) {
	nextAction += cat.latency

	if nextAction <= sim.CurrentTime {
		panic("nextaction in the past")
	} else {
		cat.NextRotationAction(sim, nextAction)
	}
}

func (cat *FeralDruid) shouldPoolMana(sim *core.Simulation, numShiftsToOom int32) bool {
	maxShiftsPossible := cat.maxShifts()

	if (maxShiftsPossible == 0) || (cat.Talents.Furor == 0) {
		return true
	}

	effectiveFightDur := sim.GetRemainingDuration() - core.DurationFromSeconds(3.0) - cat.latency
	numShiftsToFightEnd := int32(effectiveFightDur / (time.Second * 4))
	canPoolMana := (numShiftsToOom < maxShiftsPossible) && (numShiftsToOom < numShiftsToFightEnd-1) && (sim.CurrentTime-cat.lastShift > time.Second*5)

	if !canPoolMana {
		return false
	}

	for _, cd := range cat.GetMajorCooldowns() {
		if cd.IsEnabled() && cd.Type.Matches(core.CooldownTypeMana) && cd.IsReady(sim) {
			return false
		}
	}

	return true
}

func (cat *FeralDruid) doRotation(sim *core.Simulation) (bool, time.Duration) {
	// Store state variables that will be used in calculations
	rotation := &cat.Rotation
	curCp := cat.ComboPoints()
	curEnergy := cat.CurrentEnergy()
	// Energy regenerates continuously; the rotation plans one poll step ahead.
	timeToNextTick := core.EnergyPollInterval
	nextEnergy := curEnergy + core.EnergyForTime(timeToNextTick)
	nextTick := sim.CurrentTime + timeToNextTick
	isClearcast := cat.ClearcastingAura.IsActive()
	numShiftsToOom := cat.numShiftsRemaining()
	fightDur := sim.GetRemainingDuration()
	shredCost := cat.Shred.DefaultCast.Cost
	mangleCost := rotation.PrimaryBuilder.DefaultCast.Cost
	ripCost := cat.Rip.DefaultCast.Cost

	// First determine the next special ability we want to cast
	poolMana := cat.shouldPoolMana(sim, numShiftsToOom)
	canShredTrick := rotation.UseShredTrick && cat.bleedAura.IsActive() && (curEnergy >= shredCost) && (timeToNextTick > time.Second) && ((nextEnergy-shredCost >= mangleCost) || (timeToNextTick > core.GCDDefault)) && (numShiftsToOom > 1) && !poolMana
	canRipTrick := rotation.UseRipTrick && (curCp >= 1) && ((curEnergy >= ripCost && curEnergy < mangleCost) || (nextEnergy >= ripCost && nextEnergy < mangleCost)) && (numShiftsToOom > 0) && !poolMana

	var nextAbility *druid.DruidSpell

	if isClearcast {
		nextAbility = cat.Shred
	} else if (curCp >= rotation.MinCombosForRip || canRipTrick) && cat.canRip(sim, canRipTrick) {
		nextAbility = cat.Rip
	} else if canShredTrick {
		nextAbility = cat.Shred
	} else {
		nextAbility = rotation.PrimaryBuilder
	}

	// Then determine whether to cast vs. wait vs. shift
	poolEnergy := poolMana && curCp == 5 && (nextEnergy < 100) && (nextAbility == rotation.PrimaryBuilder)
	nextAction := sim.CurrentTime

	if nextAbility.CanCast(sim, cat.CurrentTarget) && !poolEnergy {
		nextAbility.Cast(sim, cat.CurrentTarget)
		return false, nextAction
	}

	shiftNow := ((nextEnergy < nextAbility.DefaultCast.Cost) || (timeToNextTick > rotation.MaxWaitTime)) && (fightDur > core.GCDDefault+cat.latency)

	// Out of energy: Shifting Power is a better refill than a power-shift
	// while it is ready and the whole 40 energy fits under the cap.
	if shiftNow && cat.shiftingPowerFits(sim) {
		cat.ShiftingPower.Cast(sim, cat.CurrentTarget)
		return false, nextAction
	}

	if shiftNow && (poolMana || (numShiftsToOom == 0)) {
		shiftNow = false
		cat.Metrics.MarkOOM(sim)

		if !cat.poolingMana {
			cat.poolingMana = true
			cat.poolStartTime = sim.CurrentTime
		}
	}

	if shiftNow {
		cat.readyToShift = true

		if cat.poolingMana {
			cat.poolingMana = false
			cat.Metrics.AddOOMTime(sim, sim.CurrentTime-cat.poolStartTime)
		}
	} else {
		nextAction = nextTick
	}

	return true, nextAction
}

func (cat *FeralDruid) setupRotation(config *proto.APLActionCatOptimalRotationAction) {
	primaryBuilder := cat.Shred

	cat.Rotation = FeralDruidRotation{
		PrimaryBuilder:     primaryBuilder,
		MinCombosForRip:    config.MinCombosForRip,
		MaxWaitTime:        core.DurationFromSeconds(float64(config.MaxWaitTime)),
		MaintainFaerieFire: config.MaintainFaerieFire,
		UseShredTrick:      config.UseShredTrick,
		UseRipTrick:        false,
	}
}
