package core

import (
	"fmt"
	"math"
	"strconv"
	"time"
)

type OnSnapshot func(sim *Simulation, target *Unit, dot *Dot, isRollover bool)
type OnTick func(sim *Simulation, target *Unit, dot *Dot)

type DotConfig struct {
	IsAOE    bool // Set to true for AOE dots (Blizzard, Hurricane, Consecrate, etc)
	SelfOnly bool // Set to true to only create the self-hot.

	// Optional, will default to the corresponding spell.
	Spell *Spell

	Aura Aura

	NumberOfTicks int32         // number of ticks over the whole duration
	TickLength    time.Duration // time between each tick

	AffectedByCastSpeed bool // If true, tick length will be shortened based on casting speed.

	OnSnapshot OnSnapshot
	OnTick     OnTick

	DamageMultiplier float64 // periodic damage multiplier
	BonusCoefficient float64 // EffectBonusCoefficient in SpellEffect client DB table, "SP mod" on Wowhead (not necessarily shown there even if > 0)

	// CanCrit forces this dot's ticks to roll crit under Dot.OutcomeTick.
	// Forever's rule (periodicDamageCanCrit) already lets every bleed,
	// every dot that is not channelled and every client-flagged channel
	// crit; set it for an unflagged channel a patch note says crits, or
	// one that pulses through a server tick row the client lets crit
	// (Blizzard, Hurricane, Rain of Fire, Volley: direct damage rows
	// without Can't Crit, and the logs show their ticks crit).
	// It does not gate the upstream snapshot-crit outcomes, whose OnTick
	// still chooses them.
	CanCrit bool

	// NeverCrits keeps this dot's ticks off the crit roll whatever
	// periodicCanCrit says: the client marks the periodic effect Can't
	// Crit (SPELL_ATTR2_CANT_CRIT, Attributes_2 0x20000000), which
	// overrides Forever's bleed and true-dot rules. Deep Wounds (the
	// warrior's 412613) and Ignite (the mage's 412545) carry it; the
	// launch logs agree for Deep Wounds: 0 of 206 Deep Wound ticks crit
	// where the same warriors' Rend ticks crit at their melee crit chance.
	NeverCrits bool

	// CritMultiplier is the multiplier a critical tick uses. Zero means
	// "the parent spell's", which is the safe default; Forever's
	// periodic figure is unpublished and may differ from the direct
	// one, so it gets its own field rather than borrowing.
	// unconfirmed
	CritMultiplier float64

	// Dynamic re-takes the dot's snapshot before every tick, so spell
	// power and the caster's damage multipliers are read when each tick
	// lands rather than when the dot was applied. Forever changed to
	// dynamic DoTs (forever-bugs #137, Blizzard 2026-10-01; #398,
	// 2026-10-26). Set it on a dot whose OnSnapshot is a plain
	// Dot.Snapshot of a fixed base; a dot that rewrites its snapshot
	// fields after that call (ramps, extra multipliers) needs its own
	// per-tick read instead.
	Dynamic bool

	// DynamicSnapshot re-runs the dot's own OnSnapshot before every tick,
	// for a dot whose OnSnapshot is a pure function of the caster's current
	// stats (every heal over time here: base tick plus its share of
	// healing power, times the caster's healing multipliers). Forever's
	// heals over time are dynamic like its DoTs (forever-bugs #137,
	// Blizzard 2026-10-01: "DoT/HoT boosting effects that apply
	// dynamically").
	DynamicSnapshot bool
}

type Dot struct {
	Spell *Spell

	// Embed Aura, so we can use IsActive/Refresh/etc directly.
	*Aura

	OriginalNumberOfTicks int32         // base number of ticks of the original dot
	NumberOfTicks         int32         // number of ticks over the whole duration
	TickLength            time.Duration // time between each tick

	// ModNumberOfTicks is the net ticks the spell's SpellMods have added.
	// A spell that assigns NumberOfTicks on every cast adds it back so a
	// duration modifier survives the assignment.
	ModNumberOfTicks int32

	// If true, tick length will be shortened based on casting speed.
	AffectedByCastSpeed bool

	OnSnapshot OnSnapshot
	OnTick     OnTick

	SnapshotBaseDamage         float64
	SnapshotCritChance         float64
	SnapshotAttackerMultiplier float64

	tickAction *PendingAction
	tickPeriod time.Duration

	// Number of ticks since last call to Apply().
	TickCount int32

	lastTickTime time.Duration
	isChanneled  bool

	DamageMultiplier float64 // periodic damage multiplier
	BonusCoefficient float64 // EffectBonusCoefficient in SpellEffect client DB table, "SP mod" on Wowhead (not necessarily shown there even if > 0)

	// CanCrit opts this dot into periodic critical strikes under
	// OutcomeMagicCritPerTick. See DotConfig.CanCrit; copied through from
	// there when the Dot is built.
	CanCrit bool

	// CritMultiplier is this dot's own critical multiplier. Zero means
	// "the parent spell's" - see dotCritMultiplier.
	CritMultiplier float64

	// dynamic and dynamicBase carry DotConfig.Dynamic: the base the last
	// Snapshot took, re-read with the caster's current stats each tick.
	dynamic     bool
	dynamicBase float64

	// dynamicSnapshot carries DotConfig.DynamicSnapshot.
	dynamicSnapshot bool

	// clockRate is how fast the dot's own clock runs against the sim's;
	// zero reads as 1. See SetClockRate.
	clockRate float64
}

// TickPeriod is how fast the snapshot dot ticks.
func (dot *Dot) TickPeriod() time.Duration {
	return dot.tickPeriod
}

func (dot *Dot) NextTickAt() time.Duration {
	return dot.lastTickTime + dot.tickPeriod
}

func (dot *Dot) TimeUntilNextTick(sim *Simulation) time.Duration {
	return dot.NextTickAt() - sim.CurrentTime
}

func (dot *Dot) MaxTicksRemaining() int32 {
	return dot.NumberOfTicks - dot.TickCount
}

func (dot *Dot) NumTicksRemaining(sim *Simulation) int {
	maxTicksRemaining := dot.MaxTicksRemaining()
	finalTickAt := dot.lastTickTime + dot.tickPeriod*time.Duration(maxTicksRemaining)
	return max(0, int((finalTickAt-sim.CurrentTime)/dot.tickPeriod)+1)
}

// Roll over = gets carried over with everlasting refresh and doesn't get applied if triggered when the spell is already up.
// - Example: critical strike rating, internal % damage modifiers: buffs or debuffs on player
// Nevermelting Ice, Shadow Mastery (ISB), Trick of the Trades, Deaths Embrace, Thaddius Polarity, Hera Spores, Crit on weapons from swapping

// Snapshot = calculation happens at refresh and application (stays up even if buff falls of, until new refresh or application)
// - Example: Spell power, Haste rating
// Blood Fury, Lightweave Embroid, Eradication, Bloodlust

// Dynamic = realtime update
// - Example: external % damage modifier debuffs on target
// Haunt, Curse of Shadow, Shadow Embrace

// Rollover is used to reset the duration of a dot from an external spell (not casting the dot itself)
// This keeps the snapshot crit and %dmg modifiers.
// However, sp and haste are recalculated.
func (dot *Dot) Rollover(sim *Simulation) {
	dot.TakeSnapshot(sim, true)

	dot.RecomputeAuraDuration() // recalculate haste
	dot.Aura.Refresh(sim)       // update aura's duration

	oldNextTick := dot.tickAction.NextActionAt
	dot.tickAction.Cancel(sim) // remove old PA ticker

	// recreate with new period, resetting the next tick.
	periodicOptions := dot.basePeriodicOptions()
	periodicOptions.Period = dot.tickPeriod
	dot.tickAction = NewPeriodicAction(sim, periodicOptions)
	dot.tickAction.NextActionAt = oldNextTick
	sim.AddPendingAction(dot.tickAction)
}

func (dot *Dot) RescheduleNextTick(sim *Simulation) {
	dot.RecomputeAuraDuration()

	dot.tickAction.Cancel(sim) // remove old PA ticker

	// recreate with new period, resetting the next tick.
	periodicOptions := dot.basePeriodicOptions()
	periodicOptions.Period = dot.tickPeriod
	dot.tickAction = NewPeriodicAction(sim, periodicOptions)
	dot.tickAction.NextActionAt = dot.lastTickTime + dot.tickPeriod
	sim.AddPendingAction(dot.tickAction)
}

func (dot *Dot) Apply(sim *Simulation) {
	dot.TakeSnapshot(sim, false)

	dot.Cancel(sim)
	dot.TickCount = 0
	dot.RecomputeAuraDuration()
	dot.Aura.Activate(sim)
}

// ApplyOrReset is used for rolling dots that reset the tick timer on reapplication.
// This is more efficient than Apply(), and works around tickAction.CleanUp() wrongly generating
// an extra ticks if (re-)application and tick happen at the same time.
func (dot *Dot) ApplyOrReset(sim *Simulation) {
	if !dot.IsActive() {
		dot.Apply(sim)
		return
	}

	dot.TakeSnapshot(sim, true)

	dot.RecomputeAuraDuration() // recalculate haste
	dot.Aura.Refresh(sim)       // update aura's duration

	dot.TickCount = 0

	oldTickAction := dot.tickAction
	dot.tickAction = nil      // prevent tickAction.CleanUp() from adding an extra tick
	oldTickAction.Cancel(sim) // remove old PA ticker

	// recreate with new period, resetting the next tick.
	periodicOptions := dot.basePeriodicOptions()
	periodicOptions.Period = dot.tickPeriod
	dot.tickAction = NewPeriodicAction(sim, periodicOptions)
	sim.AddPendingAction(dot.tickAction)
}

// Like Apply(), but does not reset the tick timer.
func (dot *Dot) ApplyOrRefresh(sim *Simulation) {
	dot.TakeSnapshot(sim, false)

	dot.TickCount = 0
	dot.RecomputeAuraDuration()
	dot.Aura.Activate(sim)
}

func (dot *Dot) Cancel(sim *Simulation) {
	if dot.Aura.IsActive() {
		dot.Aura.Deactivate(sim)
	}
}

// Call this after manually changing NumberOfTicks or TickLength.
func (dot *Dot) RecomputeAuraDuration() {
	if dot.AffectedByCastSpeed {
		dot.tickPeriod = dot.Spell.Unit.ApplyCastSpeedForSpell(dot.TickLength, dot.Spell)
	} else {
		dot.tickPeriod = dot.TickLength
	}
	if rate := dot.ClockRate(); rate != 1 {
		dot.tickPeriod = time.Duration(float64(dot.tickPeriod) / rate)
	}
	dot.Aura.Duration = dot.tickPeriod * time.Duration(dot.NumberOfTicks)
}

// ClockRate is how fast the dot's own clock runs against the sim's: 1
// unless SetClockRate changed it.
func (dot *Dot) ClockRate() float64 {
	if dot.clockRate == 0 {
		return 1
	}
	return dot.clockRate
}

// SetClockRate makes the dot's own clock run rate times as fast as the
// sim's from now on: Forever's Wrack makes the caster's Corruption, Siphon
// Life and Bane of Agony "deal their periodic damage 100% faster" while it
// is channelled, and "the time it skips is kept" once it ends. An active
// dot keeps the time it has run: its next tick and its expiry move to where
// its clock reaches them at the new rate, on the same tick grid, and every
// later tick comes at the new period. An inactive dot only stores the rate,
// so an application while it holds runs at that rate too.
func (dot *Dot) SetClockRate(sim *Simulation, rate float64) {
	if rate <= 0 {
		panic(fmt.Sprintf("dot %v: clock rate %v must be positive", dot.Spell.ActionID, rate))
	}
	old := dot.ClockRate()
	if rate == old {
		return
	}
	dot.clockRate = rate
	if !dot.IsActive() || dot.tickAction == nil {
		return
	}

	scale := func(d time.Duration) time.Duration { return time.Duration(float64(d) * old / rate) }
	nextTickAt := sim.CurrentTime + scale(dot.tickAction.NextActionAt-sim.CurrentTime)
	remaining := scale(dot.Aura.RemainingDuration(sim))
	dot.tickPeriod = scale(dot.tickPeriod)
	dot.Aura.Duration = dot.tickPeriod * time.Duration(dot.NumberOfTicks)

	dot.tickAction.Cancel(sim)
	periodicOptions := dot.basePeriodicOptions()
	periodicOptions.Period = dot.tickPeriod
	dot.tickAction = NewPeriodicAction(sim, periodicOptions)
	dot.tickAction.NextActionAt = nextTickAt
	dot.lastTickTime = nextTickAt - dot.tickPeriod
	sim.AddPendingAction(dot.tickAction)

	// The expiry stays on the tick grid, so the last tick and the fade still
	// land together after the rounding a scaled duration can carry.
	ticksAfterNext := math.Round(float64(remaining-(nextTickAt-sim.CurrentTime)) / float64(dot.tickPeriod))
	dot.Aura.UpdateExpires(sim, nextTickAt+dot.tickPeriod*time.Duration(max(ticksAfterNext, 0)))
}

// Takes a new snapshot of this Dot's effects.
//
// In most cases this will be called automatically, and should only be called
// to force a new snapshot to be taken.
//
//	doRollover will apply previously snapshotted crit/%dmg instead of recalculating.
func (dot *Dot) TakeSnapshot(sim *Simulation, doRollover bool) {
	if dot.OnSnapshot != nil {
		dot.OnSnapshot(sim, dot.Unit, dot, doRollover)
	}
}

// Forces an instant tick. Does not reset the tick timer or aura duration,
// the tick is simply an extra tick.
func (dot *Dot) TickOnce(sim *Simulation) {
	dot.lastTickTime = sim.CurrentTime
	if dot.dynamic {
		dot.Snapshot(dot.Unit, dot.dynamicBase, false)
	}
	if dot.dynamicSnapshot {
		dot.TakeSnapshot(sim, false)
	}
	dot.OnTick(sim, dot.Unit, dot)

	if dot.isChanneled {
		// Note: even if the clip delay is 0ms, need a WaitUntil so that APL is called after the channel aura fully fades.
		if dot.MaxTicksRemaining() == 0 {
			if dot.Spell.Unit.GCD.IsReady(sim) {
				dot.Spell.Unit.WaitUntil(sim, sim.CurrentTime+dot.Spell.Unit.ChannelClipDelay)
			}
		} else if dot.Spell.Unit.Rotation.shouldInterruptChannel(sim) {
			dot.Cancel(sim)
			if dot.Spell.Unit.GCD.IsReady(sim) {
				dot.Spell.Unit.WaitUntil(sim, sim.CurrentTime+dot.Spell.Unit.ChannelClipDelay)
			}
		}
	}
}

// ManualTick forces the dot forward one tick
// Will cancel the dot if it is out of ticks.
func (dot *Dot) ManualTick(sim *Simulation) {
	if dot.lastTickTime != sim.CurrentTime {
		dot.TickCount++
		if dot.NumTicksRemaining(sim) <= 0 {
			dot.Cancel(sim)
		} else {
			dot.TickOnce(sim)
		}
	}
}

func (dot *Dot) basePeriodicOptions() PeriodicActionOptions {
	return PeriodicActionOptions{
		//Priority: ActionPriorityDOT,
		OnAction: func(sim *Simulation) {
			if dot.lastTickTime != sim.CurrentTime {
				dot.TickCount++
				dot.TickOnce(sim)
			}
		},
		CleanUp: func(sim *Simulation) {
			// In certain cases, the last tick and the dot aura expiration can happen in
			// different orders, so we might need to apply the last tick.
			if dot.tickAction != nil && dot.tickAction.NextActionAt == sim.CurrentTime {
				if dot.lastTickTime != sim.CurrentTime {
					dot.TickCount++
					dot.TickOnce(sim)
				}
			}
		},
	}
}

func newDot(config Dot) *Dot {
	dot := &Dot{}
	*dot = config

	dot.tickPeriod = dot.TickLength
	dot.Aura.Duration = dot.TickLength * time.Duration(dot.NumberOfTicks)

	dot.Aura.ApplyOnGain(func(aura *Aura, sim *Simulation) {
		dot.lastTickTime = sim.CurrentTime
		periodicOptions := dot.basePeriodicOptions()
		periodicOptions.Period = dot.tickPeriod
		dot.tickAction = NewPeriodicAction(sim, periodicOptions)
		sim.AddPendingAction(dot.tickAction)
		if dot.isChanneled {
			dot.Spell.Unit.ChanneledDot = dot
			dot.Spell.Unit.channelPushback = 1.0
		}
	})
	dot.Aura.ApplyOnExpire(func(aura *Aura, sim *Simulation) {
		if dot.tickAction != nil {
			dot.tickAction.Cancel(sim)
			dot.tickAction = nil
		}
		if dot.isChanneled {
			dot.Spell.Unit.ChanneledDot = nil
			dot.Spell.Unit.Rotation.interruptChannelIf = nil
			dot.Spell.Unit.Rotation.allowChannelRecastOnInterrupt = false
			// track time metrics for channels
			dot.Spell.SpellMetrics[aura.Unit.UnitIndex].TotalCastTime += dot.fadeTime - dot.StartedAt()
		}
	})

	return dot
}

type DotArray []*Dot

func (dots DotArray) Get(target *Unit) *Dot {
	return dots[target.UnitIndex]
}

func (spell *Spell) createDots(config DotConfig, isHot bool) {
	if config.NumberOfTicks == 0 && config.TickLength == 0 {
		return
	}

	if config.Spell == nil {
		config.Spell = spell
	}

	if config.DamageMultiplier == 0 {
		config.DamageMultiplier = 1
	}

	dot := Dot{
		Spell: config.Spell,

		OriginalNumberOfTicks: config.NumberOfTicks,
		NumberOfTicks:         config.NumberOfTicks,
		TickLength:            config.TickLength,
		AffectedByCastSpeed:   config.AffectedByCastSpeed,

		OnSnapshot: config.OnSnapshot,
		OnTick:     config.OnTick,

		isChanneled: config.Spell.Flags.Matches(SpellFlagChanneled),

		DamageMultiplier: config.DamageMultiplier,
		BonusCoefficient: config.BonusCoefficient,

		CanCrit:        !config.NeverCrits && (config.CanCrit || periodicCanCrit(config.Spell, isHot)),
		CritMultiplier: config.CritMultiplier,

		dynamic:         config.Dynamic,
		dynamicSnapshot: config.DynamicSnapshot,
	}

	auraConfig := config.Aura
	if auraConfig.ActionID.IsEmptyAction() {
		auraConfig.ActionID = dot.Spell.ActionID
	}

	if auraConfig.Label == "" {
		panic("DoT aura label (Dot.Aura.Label) is empty for spell " + config.Spell.ActionID.String())
	}

	caster := dot.Spell.Unit
	if config.IsAOE || config.SelfOnly {
		dot.Aura = caster.GetOrRegisterAura(auraConfig)
		caster.claimDotAura(dot.Aura, spell)
		spell.aoeDot = newDot(dot)
	} else {
		auraConfig.Label += "-" + strconv.Itoa(int(caster.UnitIndex))
		if spell.dots == nil {
			spell.dots = make([]*Dot, len(caster.Env.AllUnits))
		}
		for _, target := range caster.Env.AllUnits {
			if isHot != caster.IsOpponent(target) {
				dot.Aura = target.GetOrRegisterAura(auraConfig)
				target.claimDotAura(dot.Aura, spell)
				spell.dots[target.UnitIndex] = newDot(dot)
			}
		}
	}
}

// periodicCanCrit is whether a dot's or heal over time's ticks may crit. A
// heal over time follows the client flag alone: every trained Rejuvenation,
// Regrowth, Tranquility, Wild Growth, Renew, Riptide and Healing Stream rank
// carries PERIODIC_CAN_CRIT in 1.60.1.70707, and Blizzard's beta notes of 24
// September 2026 say "now Rejuvenation can land critical hits".
func periodicCanCrit(spell *Spell, isHot bool) bool {
	if isHot {
		return clientPeriodicCanCritSpell(spell.SpellID)
	}
	return periodicDamageCanCrit(spell)
}

// periodicDamageCanCrit is Forever's periodic crit rule (owner,
// 2026-10-31): a periodic damage effect crits if it is a bleed (Deep
// Wounds included), OR a dot that is not channelled (Ignite included), OR
// the client marks its spell PERIODIC_CAN_CRIT. Only a channel depends on
// the client flag, so Mind Flay crits while an unflagged channel ticks
// flat. A spell's DotConfig can still override the rule both ways:
// NeverCrits for a Can't Crit server tick (Deep Wounds, Ignite), CanCrit
// for a ground channel that pulses through a direct-damage row (Blizzard,
// Hurricane, Rain of Fire, Volley).
func periodicDamageCanCrit(spell *Spell) bool {
	return spell.periodicIsBleed() ||
		!spell.Flags.Matches(SpellFlagChanneled) ||
		clientPeriodicCanCritSpell(spell.SpellID)
}

// periodicIsBleed is the engine's bleed tag: periodic physical damage
// (Rend, Deep Wounds, Rip, Rupture, Garrote), which ignores armor and
// takes the target's bleed damage multiplier.
func (spell *Spell) periodicIsBleed() bool {
	return spell.SpellSchool.Matches(SpellSchoolPhysical)
}

// clientPeriodicCanCritSpell reports whether the client marks spellID's
// periodic damage or healing PERIODIC_CAN_CRIT (periodic_crit_auto_gen.go).
func clientPeriodicCanCritSpell(spellID int32) bool {
	_, ok := clientPeriodicCanCrit[spellID]
	return ok
}

// dotCritMultiplier resolves a dot's critical multiplier. Zero means the
// dot did not choose one, and it inherits the parent spell's rather than
// silently becoming a 1.0.
func dotCritMultiplier(dotMult, spellMult float64) float64 {
	if dotMult == 0 {
		return spellMult
	}
	return dotMult
}

// claimDotAura records that spell's dot wraps aura on unit, and refuses a
// second spell: every dot appends its own gain and expire handlers to the
// aura it wraps, so two spells sharing one aura would tick both dots
// whenever either is applied. Give each spell (each rank) its own label.
func (unit *Unit) claimDotAura(aura *Aura, spell *Spell) {
	if unit.dotAuraSpells == nil {
		unit.dotAuraSpells = make(map[*Aura]*Spell)
	}
	if owner, ok := unit.dotAuraSpells[aura]; ok && owner != spell {
		panic(fmt.Sprintf("Dot aura %s on %s already belongs to %s; %s needs its own label", aura.Label, unit.Label, owner.ActionID, spell.ActionID))
	}
	unit.dotAuraSpells[aura] = spell
}
