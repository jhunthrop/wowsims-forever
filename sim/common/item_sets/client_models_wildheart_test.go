package item_sets_test

import (
	"math"
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

const (
	wildheartSet         int32 = 185
	wildheartBonus       int32 = 450608
	wildheartPieces            = 5
	wildheartName              = "S03 - Druid Energize Trigger - Wildheart Raiment"
	wildheartManaSpell   int32 = 27782
	wildheartRageSpell   int32 = 27783
	wildheartEnergySpell int32 = 27784
	rageClientTenthsWild       = 10.0
	energyTrials               = 20_000
	wholeProcTolerance         = 1e-6
)

// The chance is the bonus's dummy effect, in percent.
func wildheartChance() float64 {
	for _, effect := range core.MustClientSpellRow(wildheartBonus).Effects {
		return effect.Points / 100
	}
	panic("the Wildheart bonus has no effect row")
}

// The three returns, as the spells the bonus's tooltip names state them.
func wildheartMana() float64 { return triggerPoints(wildheartManaSpell) }

func wildheartRage() float64 { return triggerPoints(wildheartRageSpell) / rageClientTenthsWild }

func wildheartEnergy() (perTick float64, ticks int) {
	row := core.MustClientSpellRow(wildheartEnergySpell)
	return row.Effects[0].Points, int(row.DurationMS / row.Effects[0].PeriodMS)
}

// energyWindow is long enough for a whole proc's ticks to land.
func energyWindow() time.Duration {
	_, ticks := wildheartEnergy()
	return time.Duration(ticks+1) * time.Second
}

func TestWildheartRestoresManaOnSpellcastAndRageWhenStruck(t *testing.T) {
	cases := []struct {
		name   string
		host   hostClass
		label  string
		power  proto.ResourceType
		amount float64
		fire   func(wornSet, *core.Aura)
	}{
		{"mana", mageHost, wildheartName + " (spellcast)", proto.ResourceType_ResourceTypeMana, wildheartMana(), spellcast},
		{"rage", warriorHost, wildheartName + " (damage taken)", proto.ResourceType_ResourceTypeRage, wildheartRage(),
			func(w wornSet, aura *core.Aura) { w.struck(aura) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if wear(t, tc.host, wildheartSet, wildheartPieces-1).character.GetAura(tc.label) != nil {
				t.Fatal("four pieces already carry the bonus")
			}
			w := wear(t, tc.host, wildheartSet, wildheartPieces)
			aura := w.character.GetAura(tc.label)
			if aura == nil {
				t.Fatalf("five pieces do not carry %q", tc.label)
			}
			events, perEvent := resourceOf(w.character, tc.power).gains(w, procTrials, func() { tc.fire(w, aura) })
			if perEvent != tc.amount {
				t.Errorf("each proc restores %v, want %v", perEvent, tc.amount)
			}
			if want := wildheartChance() * procTrials; !withinTolerance(events, want) {
				t.Errorf("%d procs in %d triggers, want about %.0f", events, procTrials, want)
			}
		})
	}
}

// energyDuring steps the sim through the window, emptying energy after
// every event, and returns everything energy gained: the rogue's own
// regeneration plus whatever the bonus restored.
func energyDuring(w wornSet, probe resource) float64 {
	var total float64
	probe.drain(w.sim)
	for w.sim.CurrentTime < energyWindow() && !w.sim.Step() {
		total += probe.current()
		probe.drain(w.sim)
	}
	return total
}

func TestWildheartRestoresEnergyOverTimeOnMeleeHits(t *testing.T) {
	label := wildheartName + " (melee autoattack)"
	bare := wear(t, rogueHost, wildheartSet, wildheartPieces-1)
	baseline := energyDuring(bare, resourceOf(bare.character, proto.ResourceType_ResourceTypeEnergy))

	w := wear(t, rogueHost, wildheartSet, wildheartPieces)
	aura := w.character.GetAura(label)
	if aura == nil {
		t.Fatalf("five pieces do not carry %q", label)
	}
	probe := resourceOf(w.character, proto.ResourceType_ResourceTypeEnergy)
	for i := 0; i < energyTrials; i++ {
		w.swing(aura, core.ProcMaskMeleeMHAuto)
	}
	restored := energyDuring(w, probe) - baseline

	perTick, ticks := wildheartEnergy()
	perProc := perTick * float64(ticks)
	procs := restored / perProc
	if math.Abs(procs-math.Round(procs)) > wholeProcTolerance {
		t.Fatalf("the bonus restored %v energy, not a whole number of %v-energy procs", restored, perProc)
	}
	if want := wildheartChance() * energyTrials; !withinTolerance(int(procs), want) {
		t.Errorf("%v procs in %d swings, want about %.0f", procs, energyTrials, want)
	}
}
