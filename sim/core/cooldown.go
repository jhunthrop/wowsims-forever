package core

import (
	"time"
)

const startingCDTime = -60 * time.Minute

// Stored value is the time at which the cooldown will be available again.
type Timer time.Duration

type Cooldown struct {
	*Timer

	// Default amount of time after activation before this CD can be used again.
	// Note that some CDs won't use this, e.g. the GCD.
	Duration time.Duration
}

func (unit *Unit) NewTimer() *Timer {
	if len(unit.cdTimers) > 100 {
		panic("Over 100 timers! There is probably one being registered every iteration.")
	}

	newTimer := new(Timer)
	unit.cdTimers = append(unit.cdTimers, newTimer)
	return newTimer
}

func (unit *Unit) resetCDs(_ *Simulation) {
	for _, timer := range unit.cdTimers {
		timer.Reset()
	}
}

func (timer *Timer) ReadyAt() time.Duration {
	return time.Duration(*timer)
}

func (timer *Timer) Set(t time.Duration) {
	*timer = Timer(t)
}

func (timer *Timer) Reset() {
	*timer = Timer(startingCDTime)
}

func (timer *Timer) TimeToReady(sim *Simulation) time.Duration {
	return max(0, time.Duration(*timer)-sim.CurrentTime)
}

func (timer *Timer) IsReady(sim *Simulation) bool {
	return time.Duration(*timer) <= sim.CurrentTime
}

// Puts this CD on cooldown, using the default duration.
func (cd *Cooldown) Use(sim *Simulation) {
	*cd.Timer = Timer(sim.CurrentTime + cd.Duration)
}

func BothTimersReadyAt(t1 *Timer, t2 *Timer) time.Duration {
	readyAt := time.Duration(0)
	if t1 != nil {
		readyAt = t1.ReadyAt()
	}
	if t2 != nil {
		readyAt = max(readyAt, t2.ReadyAt())
	}
	return readyAt
}

func BothTimersReady(t1 *Timer, t2 *Timer, sim *Simulation) bool {
	return (t1 == nil || t1.IsReady(sim)) && (t2 == nil || t2.IsReady(sim))
}

func MaxTimeToReady(t1 *Timer, t2 *Timer, sim *Simulation) time.Duration {
	remaining := time.Duration(0)
	if t1 != nil {
		remaining = t1.TimeToReady(sim)
	}
	if t2 != nil {
		remaining = max(remaining, t2.TimeToReady(sim))
	}
	return remaining
}

// Helper for shared timers that are not always needed, so it is only
// allocated if necessary.
func (unit *Unit) GetOrInitTimer(timer **Timer) *Timer {
	if *timer == nil {
		*timer = unit.NewTimer()
	}
	return *timer
}

type CooldownArray []*Cooldown

func (cooldowns CooldownArray) Get(target *Unit) *Cooldown {
	return cooldowns[target.UnitIndex]
}

func (caster *Unit) NewEnemyICDArray(makeCooldown func(*Unit) *Cooldown) CooldownArray {
	cooldowns := make([]*Cooldown, len(caster.Env.AllUnits))
	for _, target := range caster.Env.AllUnits {
		if target.Type == EnemyUnit {
			cooldowns[target.UnitIndex] = makeCooldown(target)
		}
	}
	return cooldowns
}

// ApplyFlatCooldownMod adds a duration to this cooldown, clamped at zero.
//
// Season of Discovery wraps cooldowns in a SpellCooldown type that keeps
// the flat and percentage modifiers separately and applies them lazily in
// a fixed order. Classic's Cooldown is a timer and a duration, and
// introducing a second cooldown type across the engine to preserve that
// ordering buys nothing: no shipped talent applies a percentage and a
// flat modifier to the same cooldown, so applying each in call order is
// indistinguishable. If one ever does, this is the place to revisit.
func (cd *Cooldown) ApplyFlatCooldownMod(duration time.Duration) {
	cd.Duration = max(0, cd.Duration+duration)
}

// ApplyFlatPercentCooldownMod scales this cooldown. `percent` is an
// offset from zero: -50 means minus fifty percent.
func (cd *Cooldown) ApplyFlatPercentCooldownMod(percent int64) {
	cd.Duration = max(0, time.Duration(float64(cd.Duration)*float64(100+percent)/100))
}
