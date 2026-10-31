package elemental

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/shaman"
)

var nodeElementalMastery = elementalNode("elemental_mastery")

// TestElementalMasteryShortensBoltsAndFreesChainLightning guards
// shaman.registerElementalMasteryCD against the 1.60.1.70658 client (spell
// 16166): a 3 minute cooldown and a 15 second aura that takes 0.5 sec off
// Lightning Bolt's and Lava Burst's cast time and removes Chain
// Lightning's cooldown, all restored when it fades.
func TestElementalMasteryShortensBoltsAndFreesChainLightning(t *testing.T) {
	sim, built := newIsolatedTalentShaman(t, elementalTalentStringWithLavaBurst(nodeElementalMastery, 1))
	if built.ElementalMastery == nil {
		t.Fatal("Elemental Mastery is not registered with the talent")
	}
	if got, want := built.ElementalMastery.CD.Duration, 3*time.Minute; got != want {
		t.Errorf("Elemental Mastery cooldown = %v, want %v (client cooldown_ms 180000)", got, want)
	}

	bolt := built.LightningBolt[len(built.LightningBolt)-1]
	lava := built.LavaBurst[len(built.LavaBurst)-1]
	chain := built.ChainLightning[len(built.ChainLightning)-1]
	boltCast, lavaCast, chainCD := bolt.DefaultCast.CastTime, lava.DefaultCast.CastTime, chain.CD.Duration
	if chainCD == 0 {
		t.Fatal("Chain Lightning has no cooldown to remove")
	}

	built.ElementalMastery.Cast(sim, sim.Encounter.TargetUnits[0])
	aura := built.GetAura("Elemental Mastery")
	if aura == nil || !aura.IsActive() {
		t.Fatal("casting Elemental Mastery did not activate its aura")
	}
	if got, want := aura.Duration, 15*time.Second; got != want {
		t.Errorf("Elemental Mastery lasts %v, want %v (client duration_ms 15000)", got, want)
	}
	if got, want := bolt.DefaultCast.CastTime, boltCast-shaman.ElementalMasteryCastTimeReduction; got != want {
		t.Errorf("Lightning Bolt cast under Elemental Mastery = %v, want %v", got, want)
	}
	if got, want := lava.DefaultCast.CastTime, lavaCast-shaman.ElementalMasteryCastTimeReduction; got != want {
		t.Errorf("Lava Burst cast under Elemental Mastery = %v, want %v", got, want)
	}
	if chain.CD.Duration != 0 {
		t.Errorf("Chain Lightning cooldown under Elemental Mastery = %v, want 0", chain.CD.Duration)
	}

	aura.Deactivate(sim)
	if bolt.DefaultCast.CastTime != boltCast || lava.DefaultCast.CastTime != lavaCast || chain.CD.Duration != chainCD {
		t.Errorf("after Elemental Mastery fades: bolt %v lava %v chain cd %v, want %v %v %v",
			bolt.DefaultCast.CastTime, lava.DefaultCast.CastTime, chain.CD.Duration, boltCast, lavaCast, chainCD)
	}
}

// TestElementalMasteryNeedsTheTalent: without the point nothing registers.
func TestElementalMasteryNeedsTheTalent(t *testing.T) {
	_, built := newIsolatedTalentShaman(t, elementalTalentStringWithLavaBurst(nodeElementalMastery, 0))
	if built.ElementalMastery != nil {
		t.Error("Elemental Mastery registered without the talent")
	}
}
