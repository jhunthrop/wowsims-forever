package core

import (
	"math"
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core/proto"
)

// Read Leyline (1259705, the Alliance Skyborne's second active) casts
// Energized 1270842: aura 110 (power regeneration percent, mana) at 50 and
// aura 88 (health regeneration percent) at 50 for 15 s. SpellCooldowns:
// 120000 ms recovery, 1500 ms global cooldown (1.60.1.70753).
func TestReadLeylineIsFiftyPercentSpiritRegenForFifteenSeconds(t *testing.T) {
	skyborne, env := racialPlayer(t, proto.Race_RaceHighOrderSkyborne, proto.MobType_MobTypeUnknown)
	spell := skyborne.GetSpell(ActionID{SpellID: readLeylineSpellID})
	if spell == nil {
		t.Fatal("Read Leyline is not registered under the client id 1259705")
	}
	if spell.CD.Duration != 2*time.Minute {
		t.Errorf("Read Leyline cooldown is %v, want 2m", spell.CD.Duration)
	}
	if spell.DefaultCast.GCD != GCDDefault {
		t.Errorf("Read Leyline global cooldown is %v, want %v", spell.DefaultCast.GCD, GCDDefault)
	}
	aura := skyborne.GetAura("Energized")
	if aura == nil || aura.Duration != 15*time.Second {
		t.Fatalf("Energized aura = %v, want a 15 s aura", aura)
	}
	sim := &Simulation{Environment: env}
	before := skyborne.PseudoStats.SpiritRegenMultiplier
	aura.Activate(sim)
	if got := skyborne.PseudoStats.SpiritRegenMultiplier - before; math.Abs(got-0.5) > 1e-9 {
		t.Errorf("Energized raises the spirit regen multiplier by %v, want 0.5", got)
	}
	aura.Deactivate(sim)
	if skyborne.PseudoStats.SpiritRegenMultiplier != before {
		t.Errorf("Energized leaves the multiplier at %v, want %v", skyborne.PseudoStats.SpiritRegenMultiplier, before)
	}
}

// The client names every racial: Tauren's Cultivation (20552, an active)
// and Plainsrunning (1259918, a passive), and Undead's fourth, Underwater
// Breathing (5227). None is unconfirmed any more.
func TestTheClientNamesEveryRacial(t *testing.T) {
	if got := UnconfirmedRacials(); len(got) != 0 {
		t.Errorf("UnconfirmedRacials() = %v, want none", got)
	}
	breathing := racialNamed(t, proto.Race_RaceUndead, "Underwater Breathing")
	if breathing.Kind != RacialPassive || breathing.SpellID != 5227 {
		t.Errorf("Underwater Breathing = %v %d, want a passive with id 5227", breathing.Kind, breathing.SpellID)
	}
}
