package warrior

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// bloodCrazeHealthPercent is Blood Craze: "Regenerates 1%/2%/3% of your
// total Health over 6 sec" at ranks 1-3 (Fury node 105934), so 1% a
// point. It is a survival talent with no damage, crit, hit, rage or
// cooldown term of its own - it changes nothing on this package's
// Patchwerk goldens - but is implemented rather than named-with-reason
// because the heal-over-time and its three triggers are all ordinary
// aura machinery this package already has.
var bloodCrazeHealthPercent = [4]float64{0, 0.01, 0.02, 0.03}

// bloodCrazeTicks splits the 6 sec heal into 2s ticks, matching Rend's
// own 3-second tick period (rend.go); the client's rank text gives a
// total and a duration, not a tick rate, so there is nothing to read a
// rate from.
const (
	bloodCrazeDuration = time.Second * 6
	bloodCrazeTicks    = 3
)

// applyBloodCraze registers Blood Craze's three independent triggers -
// taking a critical strike, landing Bloodthirst, or taking a single hit
// over 20% of max health - each of which starts its own 6s heal. The
// client states no internal cooldown between them, so two triggers 1s
// apart start two overlapping HoTs rather than one trigger being
// swallowed.
func (warrior *Warrior) applyBloodCraze() {
	if warrior.Talents.BloodCraze == 0 {
		return
	}

	healPercent := bloodCrazeHealthPercent[rankIndex(warrior.Talents.BloodCraze, bloodCrazeHealthPercent[:])]
	healthMetrics := warrior.NewHealthMetrics(core.ActionID{SpellID: TalentSpellIDs["blood_craze"][0]})

	startHeal := func(sim *core.Simulation) {
		perTick := (warrior.MaxHealth() * healPercent) / bloodCrazeTicks
		core.StartPeriodicAction(sim, core.PeriodicActionOptions{
			Period:   bloodCrazeDuration / bloodCrazeTicks,
			NumTicks: bloodCrazeTicks,
			OnAction: func(sim *core.Simulation) {
				warrior.GainHealth(sim, perTick, healthMetrics)
			},
		})
	}

	warrior.RegisterAura(core.Aura{
		Label:    "Blood Craze Trigger",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitTaken: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if !result.Landed() {
				return
			}
			if result.Outcome.Matches(core.OutcomeCrit) || result.Damage > warrior.MaxHealth()*0.20 {
				startHeal(sim)
			}
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if result.Landed() && spell.SpellCode == SpellCode_WarriorBloodthirst {
				startHeal(sim)
			}
		},
	})
}
