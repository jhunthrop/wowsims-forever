package core

import (
	"slices"

	"github.com/wowsims/classic/sim/core/proto"
)

// padTargetsForTimeline grows options.Targets to the largest count the
// timeline asks for by repeating the last one. The site sends one target
// and a timeline, not five copies of the target, and growing the proto
// here means the pooled targets are constructed, initialized and given
// attack tables by exactly the same code as a plain five-target fight.
// NewEncounter already mutates options (the execute proportions), so this
// is the file's established shape.
func padTargetsForTimeline(options *proto.Encounter) {
	if len(options.Targets) == 0 || len(options.TargetsOverTime) == 0 {
		return
	}
	maxCount := 0
	for _, entry := range options.TargetsOverTime {
		maxCount = max(maxCount, int(entry.Count))
	}
	for len(options.Targets) < maxCount {
		options.Targets = append(options.Targets, options.Targets[len(options.Targets)-1])
	}
}

// newTargetTimeline converts the proto entries to sim time and sorts them,
// so the request may list them in any order.
func newTargetTimeline(entries []*proto.TargetCountAt) []TargetCount {
	if len(entries) == 0 {
		return nil
	}
	timeline := make([]TargetCount, 0, len(entries))
	for _, entry := range entries {
		timeline = append(timeline, TargetCount{
			At:    DurationFromSeconds(entry.AtSeconds),
			Count: entry.Count,
		})
	}
	slices.SortStableFunc(timeline, func(a, b TargetCount) int {
		return int(a.At - b.At)
	})
	return timeline
}

// initialActiveCount is how many targets are up when the pull starts: the
// last timeline entry at or before zero, or the whole pool when there is
// no timeline. Clamped the same way SetActiveTargetCount clamps at run
// time: a bad count in the request (missing, zero, negative) must not
// crash NewEncounter itself, before the run-time self-healing in
// SetActiveTargetCount ever gets a chance to run.
func (encounter *Encounter) initialActiveCount() int32 {
	poolSize := int32(len(encounter.AllTargetUnits))
	count := poolSize
	if len(encounter.TargetsOverTime) == 0 {
		return count
	}
	count = encounter.TargetsOverTime[0].Count
	for _, entry := range encounter.TargetsOverTime {
		if entry.At > 0 {
			break
		}
		count = entry.Count
	}
	return min(max(count, 1), poolSize)
}

// SetActiveTargetCount makes the first `count` pooled targets active and
// every other one inactive. The count is clamped rather than panicking:
// the request is validated on the site, and a bad number must not crash a
// run that a player paid for.
func (encounter *Encounter) SetActiveTargetCount(sim *Simulation, count int32) {
	count = min(max(count, 1), int32(len(encounter.AllTargetUnits)))

	for i, target := range encounter.Targets {
		if int32(i) < count {
			target.activate(sim)
		} else {
			target.deactivate(sim)
		}
	}

	encounter.TargetUnits = encounter.AllTargetUnits[:count]
	encounter.updateAOECapMultiplier()
}

// activate brings a pooled target into the fight. It is a no-op on a
// target that is already in, which is what makes SetActiveTargetCount
// safe to call every step of the timeline.
func (target *Target) activate(sim *Simulation) {
	if target.IsEnabled() {
		return
	}
	target.enabled = true
	target.SetGCDTimer(sim, max(0, sim.CurrentTime))
	target.AutoAttacks.EnableAutoSwing(sim)
	if sim.Log != nil {
		target.Log(sim, "Target activated")
	}
}

// deactivate takes a pooled target out. Its auras expire, so the player's
// DoTs on it stop ticking, and anyone still pointed at it is retargeted
// at the first active target.
func (target *Target) deactivate(sim *Simulation) {
	if !target.IsEnabled() {
		return
	}
	target.enabled = false
	target.auraTracker.expireAll(sim)
	target.AutoAttacks.CancelAutoSwing(sim)
	// gcdAction only exists for targets with a TargetAI (see target_ai.go);
	// a plain dummy target never gets one, so guard the same way
	// SetGCDTimer already does.
	if target.gcdAction != nil {
		target.CancelGCDTimer(sim)
	}
	target.Hardcast = Hardcast{Expires: startingCDTime}

	first := target.Env.Encounter.AllTargetUnits[0]
	for _, unit := range target.Env.Raid.AllUnits {
		if unit.CurrentTarget == &target.Unit {
			unit.CurrentTarget = first
		}
	}

	if sim.Log != nil {
		target.Log(sim, "Target deactivated")
	}
}

// initEncounterTargets applies the timeline for one iteration.
// Simulation.reset calls it after Environment.reset has re-enabled every
// target, so the first thing it does is take the later ones back out.
func (sim *Simulation) initEncounterTargets() {
	encounter := &sim.Encounter
	encounter.SetActiveTargetCount(sim, encounter.initialActiveCount())

	for _, entry := range encounter.TargetsOverTime {
		if entry.At <= 0 {
			continue
		}
		count := entry.Count
		StartDelayedAction(sim, DelayedActionOptions{
			DoAt:     entry.At,
			Priority: ActionPriorityDOT,
			OnAction: func(sim *Simulation) {
				encounter.SetActiveTargetCount(sim, count)
			},
		})
	}
}
