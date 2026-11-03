package core

import (
	"fmt"
	"time"
)

// A cast corresponds to any action which causes the in-game castbar to be
// shown, and activates the GCD. Note that a cast can also be instant, i.e.
// the effects are applied immediately even though the GCD is still activated.

// Callback for when a cast is finished, i.e. when the in-game castbar reaches full.
type OnCastComplete func(aura *Aura, sim *Simulation, spell *Spell)

type Hardcast struct {
	Expires    time.Duration
	ActionID   ActionID
	OnComplete func(*Simulation, *Unit)
	Target     *Unit
	Pushback   float64
	// CastTime is the cast's full length; a pushback never leaves more
	// than this much of the cast to go.
	CastTime time.Duration
}

// Input for constructing the CastSpell function for a spell.
type CastConfig struct {
	// Default cast values with all static effects applied.
	DefaultCast Cast

	// Dynamic modifications for each cast.
	ModifyCast func(*Simulation, *Spell, *Cast)

	// Ignores haste when calculating the GCD and cast time for this cast.
	// Automatically set if GCD and cast times are all 0, e.g. for empty casts.
	IgnoreHaste bool

	CD       Cooldown
	SharedCD Cooldown

	CastTime func(spell *Spell) time.Duration
}

type Cast struct {
	// Amount of resource that will be consumed by this cast.
	Cost float64

	// The length of time the GCD will be on CD as a result of this cast.
	GCD time.Duration

	// The amount of time between the call to spell.Cast() and when the spell
	// effects are invoked.
	CastTime time.Duration
}

func (cast *Cast) EffectiveTime() time.Duration {
	gcd := cast.GCD
	if cast.GCD != 0 {
		// TODO: isn't this wrong for spells like shadowfury, that have a reduced GCD?
		gcd = max(GCDMin, gcd)
	}
	return max(gcd, cast.CastTime)
}

type CastFunc func(*Simulation, *Unit)
type CastSuccessFunc func(*Simulation, *Unit) bool

func (spell *Spell) castFailureHelper(sim *Simulation, message string, vals ...any) bool {
	if sim.CurrentTime < 0 && spell.Unit.Rotation != nil {
		spell.Unit.Rotation.ValidationWarning(fmt.Sprintf(spell.ActionID.String()+" failed to cast: "+message, vals...))
	} else {
		if sim.Log != nil && !spell.Flags.Matches(SpellFlagNoLogs) {
			spell.Unit.Log(sim, fmt.Sprintf(spell.ActionID.String()+" failed to cast: "+message, vals...))
		}
	}
	return false
}

func (unit *Unit) applySpellPushback() {
	unit.RegisterAura(Aura{
		Label:    "Spell Pushback",
		Duration: NeverExpires,
		OnReset: func(aura *Aura, sim *Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitTaken: func(aura *Aura, sim *Simulation, spell *Spell, result *SpellResult) {
			//No pushback for bosses/NPCs
			if unit.Type == EnemyUnit {
				return
			}

			if !result.Landed() {
				return
			}

			if result.Damage <= 0 {
				return
			}

			if !spell.ProcMask.Matches(ProcMaskDirect) {
				return
			}

			aura.Unit.PushBackCast(sim)
		},
	})
}

// pushBackChannel shortens the channel the unit is in: the ticks that
// would land after the new end are lost, and a channel left with no tick
// ends now and hands the unit back to its rotation.
func (unit *Unit) pushBackChannel(sim *Simulation, dot *Dot) {
	if sim.Roll(0, 1.0) <= dot.Spell.PushbackReduction {
		return
	}
	pushback := DurationFromSeconds(max(0.2, unit.channelPushback))
	unit.channelPushback -= 0.2

	newExpires := max(sim.CurrentTime, dot.ExpiresAt()-pushback)
	// A tick due inside this server batch lands before the hit does.
	if nextTick := dot.lastTickTime + dot.tickPeriod; dot.tickPeriod > 0 && nextTick > newExpires &&
		dot.MaxTicksRemaining() > 0 && sim.sameServerBatch(nextTick-sim.CurrentTime) {
		newExpires = nextTick
	}
	ticksLeft := int32(0)
	if dot.tickPeriod > 0 {
		ticksLeft = int32((newExpires - dot.lastTickTime) / dot.tickPeriod)
	}
	ticksLeft = min(ticksLeft, dot.MaxTicksRemaining())
	if sim.Log != nil {
		unit.Log(sim, "Channel shortened by %s due to damage taken, %d tick(s) left", pushback, ticksLeft)
	}
	if ticksLeft <= 0 {
		dot.Cancel(sim)
		unit.resumeAfterChannel(sim)
		return
	}
	newEnd := dot.lastTickTime + dot.tickPeriod*time.Duration(ticksLeft)
	dot.UpdateExpires(sim, newEnd)
	// The channel's own last tick only hands back to the rotation when it
	// is the channel's planned last, so the cut channel does it here.
	StartDelayedAction(sim, DelayedActionOptions{
		DoAt: newEnd,
		OnAction: func(sim *Simulation) {
			if unit.ChanneledDot != dot || !dot.IsActive() {
				unit.resumeAfterChannel(sim)
			}
		},
	})
}

// resumeAfterChannel asks the rotation for its next action once a
// channel has ended early.
func (unit *Unit) resumeAfterChannel(sim *Simulation) {
	if unit.GCD.IsReady(sim) {
		unit.WaitUntil(sim, sim.CurrentTime+unit.ChannelClipDelay)
	}
}

// PushBackCast costs the unit's current cast its casting time for taking
// damage: a cast is delayed and a channel shortened, by 1 s for the first
// hit and 0.2 s less for each hit after, never under 0.2 s. The chance to
// keep the casting time (Nature's Focus, Healing Focus, Spiritual Focus,
// Twilight Focus, Fel Concentration) belongs to the spell being cast, so
// it is read from that spell and not from whatever dealt the damage.
func (unit *Unit) PushBackCast(sim *Simulation) {
	if dot := unit.ChanneledDot; dot != nil && dot.IsActive() && !unit.IsCasting(sim) {
		unit.pushBackChannel(sim, dot)
		return
	}
	if !unit.IsCasting(sim) {
		return
	}
	hc := &unit.Hardcast
	hcSpell := unit.GetSpell(hc.ActionID)
	if hcSpell == nil {
		return
	}
	// A cast that ends inside this server batch completes before the hit.
	if !hcSpell.Flags.Matches(SpellFlagChanneled) && sim.sameServerBatch(hc.Expires-sim.CurrentTime) {
		return
	}
	if sim.Roll(0, 1.0) <= hcSpell.PushbackReduction {
		return
	}
	pushback := DurationFromSeconds(max(0.2, hc.Pushback))
	hc.Pushback -= 0.2

	metricsTarget := unit.CurrentTarget
	if hc.Target != nil {
		metricsTarget = hc.Target
	}
	metrics := &hcSpell.SpellMetrics[metricsTarget.UnitIndex]

	if hcSpell.Flags.Matches(SpellFlagChanneled) {
		newExpires := max(sim.CurrentTime, hc.Expires-pushback)
		if sim.Log != nil {
			unit.Log(sim, "Unit Hardcast shortened by %s due to damage taken, will now occur at %s", pushback, newExpires)
		}

		// Update Dot if present
		if hcDot := hcSpell.CurDot(); hcDot != nil {
			hcDot.UpdateExpires(sim, newExpires)
		}

		metrics.TotalCastTime -= hc.Expires - newExpires
		hc.Expires = newExpires
	} else {
		// The vanilla server never leaves more than the full cast to go
		// (cmangos-classic Spell::Delayed).
		if hc.CastTime > 0 {
			pushback = max(0, min(pushback, sim.CurrentTime+hc.CastTime-hc.Expires))
		}
		if sim.Log != nil {
			unit.Log(sim, "Unit Hardcast extended by %s due to damage taken, will now occur at %s", pushback, hc.Expires+pushback)
		}

		hc.Expires += pushback
		metrics.TotalCastTime += pushback
	}

	// Update GCDTimer
	unit.SetGCDTimer(sim, hc.Expires)

	// Update Swing timer
	unit.AutoAttacks.StopMeleeUntil(sim, hc.Expires, false)
}

func (spell *Spell) makeCastFunc(config CastConfig) CastSuccessFunc {
	return func(sim *Simulation, target *Unit) bool {
		spell.CurCast = spell.DefaultCast

		if config.ModifyCast != nil {
			config.ModifyCast(sim, spell, &spell.CurCast)
			if spell.CurCast.Cost != spell.DefaultCast.Cost {
				// Costs need to be modified using the unit and spell multipliers, so that
				// their affects are also visible in the spell.CanCast() function, which
				// does not invoke ModifyCast.
				panic("May not modify cost in ModifyCast!")
			}
		}

		if spell.ExtraCastCondition != nil {
			if !spell.ExtraCastCondition(sim, target) {
				return spell.castFailureHelper(sim, "extra spell condition")
			}
		}

		if spell.Cost != nil {
			if !spell.Cost.MeetsRequirement(sim, spell) {
				return spell.castFailureHelper(sim, spell.Cost.CostFailureReason(sim, spell))
			}
		}

		if !config.IgnoreHaste {
			// Vanilla has no natural GCD reduction besides abilities with 1s GCDs
			// spell.CurCast.GCD = spell.Unit.ApplyFlatCastSpeed(spell.CurCast.GCD)
			spell.CurCast.CastTime = config.CastTime(spell)
		}

		if config.CD.Timer != nil {
			// By panicking if spell is on CD, we force each sim to properly check for their own CDs.
			if !spell.CD.IsReady(sim) {
				return spell.castFailureHelper(sim, "still on cooldown for %s, curTime = %s", spell.CD.TimeToReady(sim), sim.CurrentTime)
			}

			spell.CD.Set(sim.CurrentTime + spell.CurCast.CastTime + spell.CD.Duration)
		}

		if config.SharedCD.Timer != nil {
			// By panicking if spell is on CD, we force each sim to properly check for their own CDs.
			if !spell.SharedCD.IsReady(sim) {
				return spell.castFailureHelper(sim, "still on shared cooldown for %s, curTime = %s", spell.SharedCD.TimeToReady(sim), sim.CurrentTime)
			}

			spell.SharedCD.Set(sim.CurrentTime + spell.CurCast.CastTime + spell.SharedCD.Duration)
		}

		// By panicking if spell is on CD, we force each sim to properly check for their own CDs.
		if spell.CurCast.GCD != 0 && !spell.Unit.GCD.IsReady(sim) {
			return spell.castFailureHelper(sim, "GCD on cooldown for %s, curTime = %s", spell.Unit.GCD.TimeToReady(sim), sim.CurrentTime)
		}

		if hc := spell.Unit.Hardcast; spell.Unit.IsCasting(sim) {
			return spell.castFailureHelper(sim, "casting/channeling %v for %s, curTime = %s", hc.ActionID, hc.Expires-sim.CurrentTime, sim.CurrentTime)
		}

		if dot := spell.Unit.ChanneledDot; spell.Unit.IsChanneling(sim) {
			return spell.castFailureHelper(sim, "channeling %v for %s, curTime = %s", dot.ActionID, dot.expires-sim.CurrentTime, sim.CurrentTime)
		}

		if effectiveTime := spell.CurCast.EffectiveTime(); effectiveTime != 0 {
			if spell.Flags.Matches(SpellFlagCastTimeNoGCD) {
				effectiveTime = max(effectiveTime, spell.Unit.GCD.TimeToReady(sim))
			}
			// do not add channeled time here as they have variable cast length
			// cast time for channels is handled in dot.OnExpire
			if !spell.Flags.Matches(SpellFlagChanneled) {
				spell.SpellMetrics[target.UnitIndex].TotalCastTime += effectiveTime
			}
			spell.Unit.SetGCDTimer(sim, sim.CurrentTime+effectiveTime)
		}

		if (spell.CurCast.CastTime > 0) && spell.Unit.IsCastingBlocked() {
			return spell.castFailureHelper(sim, "casting/channeling while moving not allowed!")
		}

		// Non melee casts
		if spell.Flags.Matches(SpellFlagResetAttackSwing) && spell.Unit.AutoAttacks.enabled {
			restartMeleeAt := sim.CurrentTime + spell.CurCast.CastTime
			spell.Unit.AutoAttacks.StopMeleeUntil(sim, restartMeleeAt, false)
		}

		// Hardcasts
		if spell.CurCast.CastTime > 0 {
			if sim.Log != nil && !spell.Flags.Matches(SpellFlagNoLogs) {
				spell.Unit.Log(sim, "Casting %s (Cost = %0.03f, Cast Time = %s, Effective Time = %s)",
					spell.ActionID, max(0, spell.CurCast.Cost), spell.CurCast.CastTime, spell.CurCast.EffectiveTime())
			}

			spell.Unit.Hardcast = Hardcast{
				Expires:  sim.CurrentTime + spell.CurCast.CastTime,
				ActionID: spell.ActionID,
				Pushback: 1.0,
				CastTime: spell.CurCast.CastTime,
				OnComplete: func(sim *Simulation, target *Unit) {
					spell.LastCastAt = sim.CurrentTime

					if sim.Log != nil && !spell.Flags.Matches(SpellFlagNoLogs) {
						spell.Unit.Log(sim, "Completed cast %s", spell.ActionID)
					}

					if spell.Cost != nil {
						if !spell.Cost.MeetsRequirement(sim, spell) {
							spell.castFailureHelper(sim, spell.Cost.CostFailureReason(sim, spell))
							return
						}
						spell.Cost.SpendCost(sim, spell)
					}

					spell.applyEffects(sim, target)

					if !spell.Flags.Matches(SpellFlagNoOnCastComplete) {
						spell.Unit.OnCastComplete(sim, spell)
					}

					if !sim.Options.Interactive {
						spell.Unit.Rotation.DoNextAction(sim)
					}
				},
				Target: target,
			}

			if spell.Unit.Hardcast.Expires != spell.Unit.NextGCDAt() {
				spell.Unit.newHardcastAction(sim)
			}

			return true
		}

		spell.LastCastAt = sim.CurrentTime

		if sim.Log != nil && !spell.Flags.Matches(SpellFlagNoLogs) {
			spell.Unit.Log(sim, "Casting %s (Cost = %0.03f, Cast Time = %s, Effective Time = %s)",
				spell.ActionID, max(0, spell.CurCast.Cost), spell.CurCast.CastTime, spell.CurCast.EffectiveTime())
			spell.Unit.Log(sim, "Completed cast %s", spell.ActionID)
		}

		if spell.Cost != nil {
			spell.Cost.SpendCost(sim, spell)
		}

		spell.applyEffects(sim, target)

		if !spell.Flags.Matches(SpellFlagNoOnCastComplete) {
			spell.Unit.OnCastComplete(sim, spell)
		}

		return true
	}
}

func (spell *Spell) makeCastFuncSimple() CastSuccessFunc {
	return func(sim *Simulation, target *Unit) bool {
		if spell.ExtraCastCondition != nil {
			if !spell.ExtraCastCondition(sim, target) {
				return spell.castFailureHelper(sim, "extra spell condition")
			}
		}

		if spell.CD.Timer != nil {
			// By panicking if spell is on CD, we force each sim to properly check for their own CDs.
			if !spell.CD.IsReady(sim) {
				return spell.castFailureHelper(sim, "still on cooldown for %s, curTime = %s", spell.CD.TimeToReady(sim), sim.CurrentTime)
			}

			spell.CD.Set(sim.CurrentTime + spell.CD.Duration)
		}

		if spell.SharedCD.Timer != nil {
			// By panicking if spell is on CD, we force each sim to properly check for their own CDs.
			if !spell.SharedCD.IsReady(sim) {
				return spell.castFailureHelper(sim, "still on shared cooldown for %s, curTime = %s", spell.SharedCD.TimeToReady(sim), sim.CurrentTime)
			}

			spell.SharedCD.Set(sim.CurrentTime + spell.SharedCD.Duration)
		}

		if sim.Log != nil && !spell.Flags.Matches(SpellFlagNoLogs) {
			spell.Unit.Log(sim, "Casting %s (Cost = %0.03f, Cast Time = %s, Effective Time = %s)",
				spell.ActionID, 0.0, "0s", "0s")
			spell.Unit.Log(sim, "Completed cast %s", spell.ActionID)
		}

		spell.applyEffects(sim, target)

		if !spell.Flags.Matches(SpellFlagNoOnCastComplete) {
			spell.Unit.OnCastComplete(sim, spell)
		}

		return true
	}
}

func (spell *Spell) makeCastFuncAutosOrProcs() CastSuccessFunc {
	return func(sim *Simulation, target *Unit) bool {
		if sim.Log != nil && !spell.Flags.Matches(SpellFlagNoLogs) {
			spell.Unit.Log(sim, "Casting %s (Cost = %0.03f, Cast Time = %s, Effective Time = %s)",
				spell.ActionID, 0.0, "0s", "0s")
			spell.Unit.Log(sim, "Completed cast %s", spell.ActionID)
		}

		spell.applyEffects(sim, target)

		if !spell.Flags.Matches(SpellFlagNoOnCastComplete) {
			spell.Unit.OnCastComplete(sim, spell)
		}

		return true
	}
}
