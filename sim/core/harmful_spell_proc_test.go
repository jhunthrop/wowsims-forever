package core

import (
	"testing"
	"time"
)

func newHarmfulProcTestUnits() (*Simulation, *Unit, *Unit) {
	sim := &Simulation{rand: NewSplitMix(1)}
	caster := &Unit{Type: PlayerUnit, auraTracker: newAuraTracker()}
	target := &Unit{Type: EnemyUnit, auraTracker: newAuraTracker()}
	return sim, caster, target
}

func harmfulHit(aura *Aura, sim *Simulation, spell *Spell, target *Unit) {
	aura.OnSpellHitDealt(aura, sim, spell, &SpellResult{Outcome: OutcomeHit, Target: target})
}

// A "chance on harmful spell cast" proc at a rate per minute: each cast
// rolls ppm x its cast time (the GCD for an instant) / 60, the recovery is
// kept per spell so two different casts in one instant both roll, and a
// self-cast (Life Tap) does not roll.
func TestHarmfulSpellProcRollsPerMinuteAndKeepsItsRecoveryPerSpell(t *testing.T) {
	sim, caster, target := newHarmfulProcTestUnits()
	procs := 0
	aura := MakeHarmfulSpellProcAura(caster, HarmfulSpellProc{
		Name:    "Test Proc",
		PPM:     60, // a 1 s cast always procs at 60 a minute
		ICD:     10 * time.Millisecond,
		Handler: func(*Simulation, *Unit) { procs++ },
	})
	bolt := &Spell{ProcMask: ProcMaskSpellDamage, CurCast: Cast{CastTime: 3 * time.Second}}
	curse := &Spell{ProcMask: ProcMaskSpellDamage}
	tap := &Spell{ProcMask: ProcMaskSpellDamage}

	sim.CurrentTime = time.Second
	harmfulHit(aura, sim, bolt, target)
	harmfulHit(aura, sim, curse, target)
	if procs != 2 {
		t.Errorf("two different casts landing in one instant procced %d times, want 2", procs)
	}
	harmfulHit(aura, sim, bolt, target)
	if procs != 2 {
		t.Errorf("the same cast's second hit inside the recovery procced (%d), want it held", procs)
	}
	harmfulHit(aura, sim, tap, caster)
	if procs != 2 {
		t.Errorf("a self-cast procced (%d)", procs)
	}
	sim.CurrentTime = 0 // a new iteration restarts the clock
	harmfulHit(aura, sim, bolt, target)
	if procs != 3 {
		t.Errorf("the first cast of a new iteration was held by the last iteration's recovery")
	}

	if got, want := SpellPPMChance(1, bolt), 0.05; got != want {
		t.Errorf("1 PPM on a 3 s cast is %v a cast, want %v", got, want)
	}
	if got, want := SpellPPMChance(1, curse), 0.025; got != want {
		t.Errorf("1 PPM on an instant is %v a cast, want %v (the 1.5 s GCD)", got, want)
	}
}
