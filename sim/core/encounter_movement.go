package core

// EncounterMovementDistance is how far an away window puts a player from
// the target: past melee range (5) and at the minimum ranged distance
// (12), so the window costs melee and casting without also making ranged
// attacks illegal for a reason the fight style never asked for.
const EncounterMovementDistance = 12.0

// initEncounterMovement schedules the encounter's movement windows for
// every player, once per iteration. Simulation.reset calls it, beside
// initManaTickAction, because that is where per-iteration pending actions
// belong: reset has just emptied the queue and set sim.Duration.
func (sim *Simulation) initEncounterMovement() {
	pattern := sim.Encounter.Movement
	if pattern == nil {
		return
	}

	numTicks := int(sim.Duration / pattern.Interval)
	if numTicks <= 0 {
		return
	}

	for _, party := range sim.Raid.Parties {
		for _, player := range party.Players {
			unit := &player.GetCharacter().Unit
			sim.AddPendingAction(NewPeriodicAction(sim, PeriodicActionOptions{
				Period:          pattern.Interval,
				NumTicks:        numTicks,
				TickImmediately: false,
				Priority:        ActionPriorityAuto,
				OnAction: func(sim *Simulation) {
					unit.startMovementWindow(sim, pattern)
				},
			}))
		}
	}
}

// startMovementWindow opens one window on one unit and schedules its
// close. An away window activates the existing movement aura, which is
// what cancels auto attacks and channels; a casting-only window only
// interrupts the cast in progress and refuses new ones for the duration.
func (unit *Unit) startMovementWindow(sim *Simulation, pattern *MovementPattern) {
	unit.InterruptCast(sim)

	if pattern.CastingOnly {
		unit.MovementHandler.CastingBlocked = true
		if sim.Log != nil {
			unit.Log(sim, "Casting interrupted for %s", pattern.Duration)
		}
		StartDelayedAction(sim, DelayedActionOptions{
			DoAt:     sim.CurrentTime + pattern.Duration,
			Priority: ActionPriorityAuto,
			OnAction: func(sim *Simulation) {
				unit.MovementHandler.CastingBlocked = false
				if unit.Rotation != nil {
					unit.Rotation.DoNextAction(sim)
				}
			},
		})
		return
	}

	unit.MovementHandler.moveAura.Activate(sim)
	unit.DistanceFromTarget = EncounterMovementDistance
	unit.MovementHandler.moveAura.SetStacks(sim, int32(EncounterMovementDistance))
	if sim.Log != nil {
		unit.Log(sim, "Moving out of range for %s", pattern.Duration)
	}
	StartDelayedAction(sim, DelayedActionOptions{
		DoAt:     sim.CurrentTime + pattern.Duration,
		Priority: ActionPriorityAuto,
		OnAction: func(sim *Simulation) {
			unit.DistanceFromTarget = unit.StartDistanceFromTarget
			unit.MovementHandler.moveAura.Deactivate(sim)
			if unit.Rotation != nil {
				unit.Rotation.DoNextAction(sim)
			}
		},
	})
}
