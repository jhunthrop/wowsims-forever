package core

import (
	"strings"
	"testing"
	"time"
)

// Two spells must never share one dot aura on a target. The aura's gain
// and expire handlers are appended by every dot that wraps it, so a shared
// aura ticks every spell's dot whenever any one of them is applied (the
// Lightwell renews' three ranks, 2026-11-02). Registration refuses it.
func TestTwoSpellsCannotShareADotAura(t *testing.T) {
	_, caster, _ := newPeriodicCritFixture(t, 0)
	hot := func(id int32) SpellConfig {
		return SpellConfig{
			ActionID:         ActionID{SpellID: id},
			SpellSchool:      SpellSchoolHoly,
			DefenseType:      DefenseTypeMagic,
			Flags:            SpellFlagHelpful,
			ProcMask:         ProcMaskSpellHealing,
			DamageMultiplier: 1,
			ThreatMultiplier: 1,
			Hot: DotConfig{
				Aura:          Aura{Label: "Shared Test Renew"},
				NumberOfTicks: 5,
				TickLength:    2 * time.Second,
				OnTick:        func(*Simulation, *Unit, *Dot) {},
			},
		}
	}
	caster.RegisterSpell(hot(7001))
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("a second spell registered a dot on the first one's aura")
		}
		if msg, _ := r.(string); !strings.Contains(msg, "Shared Test Renew") {
			t.Errorf("panic %v does not name the aura", r)
		}
	}()
	caster.RegisterSpell(hot(27874))
}
