package enhancement

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/shaman"
)

// Flametongue Totem, the shaman's cast (client spells 8227, 8249, 10526 and
// 16387): a fire totem, 1000 ms GCD, 5 minutes, 90 / 140 / 200 / 275 mana.
func TestFlametongueTotemIsRegisteredAtEveryLearnedRank(t *testing.T) {
	_, built := newIsolatedTalentShaman(t, shamanTalentString(nil, nil, nil))
	wantIDs := []int32{8227, 8249, 10526, 16387}
	wantCost := []float64{90, 140, 200, 275}
	if len(built.FlametongueTotem) != len(wantIDs) {
		t.Fatalf("a level 60 shaman has %d Flametongue Totem ranks, want %d", len(built.FlametongueTotem), len(wantIDs))
	}
	for i, spell := range built.FlametongueTotem {
		if spell.ActionID.SpellID != wantIDs[i] {
			t.Errorf("rank %d spell id = %d, want %d", i+1, spell.ActionID.SpellID, wantIDs[i])
		}
		if spell.DefaultCast.Cost != wantCost[i] {
			t.Errorf("rank %d cost = %v, want %v", i+1, spell.DefaultCast.Cost, wantCost[i])
		}
	}
}

func TestFlametongueTotemCastActivatesItsAuraForFiveMinutes(t *testing.T) {
	sim, built := newIsolatedTalentShaman(t, shamanTalentString(nil, nil, nil))
	aura := built.GetAura(core.FlametongueTotemAuraLabel)
	if aura == nil {
		t.Fatal("the shaman has no Flametongue Totem aura")
	}
	if aura.IsActive() {
		t.Fatal("the totem's aura is up before the totem is dropped")
	}
	top := built.FlametongueTotem[len(built.FlametongueTotem)-1]
	top.ApplyEffects(sim, &built.Unit, top)
	if !aura.IsActive() {
		t.Fatal("casting Flametongue Totem did not activate its aura")
	}
	if got := built.TotemExpirations[shaman.FireTotem] - sim.CurrentTime; got != 5*time.Minute {
		t.Errorf("the totem stands for %v, want 5m", got)
	}
}

func TestSearingTotemEndsAStandingFlametongueTotem(t *testing.T) {
	sim, built := newIsolatedTalentShaman(t, shamanTalentString(nil, nil, nil))
	aura := built.GetAura(core.FlametongueTotemAuraLabel)
	flametongue := built.FlametongueTotem[len(built.FlametongueTotem)-1]
	searing := built.SearingTotem[len(built.SearingTotem)-1]
	flametongue.ApplyEffects(sim, &built.Unit, flametongue)
	searing.ApplyEffects(sim, built.CurrentTarget, searing)
	if aura.IsActive() {
		t.Error("Flametongue Totem still stands after a Searing Totem took the fire slot")
	}
	flametongue.ApplyEffects(sim, &built.Unit, flametongue)
	if !aura.IsActive() {
		t.Error("Flametongue Totem did not stand after replacing a Searing Totem")
	}
}
