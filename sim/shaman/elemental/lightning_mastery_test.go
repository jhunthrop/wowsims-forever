package elemental

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
)

var (
	nodeLightningMastery = elementalNode("lightning_mastery")
	nodeCallOfThunder    = elementalNode("call_of_thunder")
)

// TestLightningMasteryShortensBoltsAndFreesChainLightning guards
// shaman.registerLightningMasteryCD against the 1.60.1.70755 client
// (spell 16166): a 3 minute cooldown and a 15 second aura whose effect 0
// (aura 107 misc 10, -500, mask word 0 bit 0) takes 0.5 sec off Lightning
// Bolt's cast time and whose effect 1 (aura 108 misc 11, -100%, mask word
// 0 bit 1) removes Chain Lightning's cooldown, all restored when it fades.
// Chain Lightning's own cast time is untouched.
func TestLightningMasteryShortensBoltsAndFreesChainLightning(t *testing.T) {
	sim, built := newIsolatedTalentShaman(t, elementalTalentString(nodeLightningMastery, 1))
	if built.LightningMastery == nil {
		t.Fatal("Lightning Mastery is not registered with the talent")
	}
	if got, want := built.LightningMastery.ActionID.SpellID, int32(16166); got != want {
		t.Errorf("Lightning Mastery spell = %d, want %d", got, want)
	}
	if got, want := built.LightningMastery.CD.Duration, 3*time.Minute; got != want {
		t.Errorf("Lightning Mastery cooldown = %v, want %v (client cooldown_ms 180000)", got, want)
	}

	bolt := built.LightningBolt[len(built.LightningBolt)-1]
	chain := built.ChainLightning[len(built.ChainLightning)-1]
	boltCast, chainCast, chainCD := bolt.DefaultCast.CastTime, chain.DefaultCast.CastTime, chain.CD.Duration
	if chainCD == 0 {
		t.Fatal("Chain Lightning has no cooldown to remove")
	}

	built.LightningMastery.Cast(sim, sim.Encounter.TargetUnits[0])
	aura := built.GetAura("Lightning Mastery")
	if aura == nil || !aura.IsActive() {
		t.Fatal("casting Lightning Mastery did not activate its aura")
	}
	if got, want := aura.Duration, 15*time.Second; got != want {
		t.Errorf("Lightning Mastery lasts %v, want %v (client DurationIndex 8, 15000 ms)", got, want)
	}
	if got, want := bolt.DefaultCast.CastTime, boltCast-500*time.Millisecond; got != want {
		t.Errorf("Lightning Bolt cast under Lightning Mastery = %v, want %v", got, want)
	}
	if got := chain.DefaultCast.CastTime; got != chainCast {
		t.Errorf("Chain Lightning cast under Lightning Mastery = %v, want it untouched at %v", got, chainCast)
	}
	if chain.CD.Duration != 0 {
		t.Errorf("Chain Lightning cooldown under Lightning Mastery = %v, want 0", chain.CD.Duration)
	}

	aura.Deactivate(sim)
	if bolt.DefaultCast.CastTime != boltCast || chain.CD.Duration != chainCD {
		t.Errorf("after Lightning Mastery fades: bolt %v chain cd %v, want %v %v",
			bolt.DefaultCast.CastTime, chain.CD.Duration, boltCast, chainCD)
	}
}

// TestLightningMasteryNeedsTheTalent: without the point nothing registers.
func TestLightningMasteryNeedsTheTalent(t *testing.T) {
	_, built := newIsolatedTalentShaman(t, elementalTalentString(nodeLightningMastery, 0))
	if built.LightningMastery != nil {
		t.Error("Lightning Mastery registered without the talent")
	}
}

// TestCallOfThunderRaisesBoltAndChainCrit guards shaman.applyCallOfThunder
// (node 104760, spell 16120 effect 0: aura 107 misc 7, +5, class mask word
// 0 = 3): +5% crit chance on every rank of Lightning Bolt and Chain
// Lightning, and nothing on Earth Shock.
func TestCallOfThunderRaisesBoltAndChainCrit(t *testing.T) {
	_, without := newIsolatedTalentShaman(t, elementalTalentString(nodeCallOfThunder, 0))
	_, with := newIsolatedTalentShaman(t, elementalTalentString(nodeCallOfThunder, 1))
	if !with.Talents.CallOfThunder {
		t.Fatal("elementalTalentString(nodeCallOfThunder, 1) did not set Call of Thunder")
	}
	want := 5.0 * core.CritRatingPerCritChance
	check := func(name string, withSpells, withoutSpells []*core.Spell, want float64) {
		t.Helper()
		for i, spell := range withSpells {
			if spell == nil {
				continue
			}
			if got := spell.BonusCritRating - withoutSpells[i].BonusCritRating; got != want {
				t.Errorf("%s rank %d: Call of Thunder adds %v crit rating, want %v", name, spell.Rank, got, want)
			}
		}
	}
	check("Lightning Bolt", with.LightningBolt, without.LightningBolt, want)
	check("Chain Lightning", with.ChainLightning, without.ChainLightning, want)
	check("Earth Shock", with.EarthShock, without.EarthShock, 0)
}
