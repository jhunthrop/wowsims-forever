package priest

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

const (
	prayerOfMendingCooldown = 10 * time.Second
	// prayerOfMendingCharges is the client's "jumps up to 5 times", read
	// as five heals in all: the spell's second effect states 5.
	prayerOfMendingCharges   = 5
	prayerOfMendingAuraID    = 401877
	prayerOfMendingAuraLasts = 30 * time.Second
)

// prayerOfMending is one priest's Prayer of Mending: the charges sit as
// stacks of an aura on one ally at a time.
type prayerOfMending struct {
	priest *Priest
	auras  core.AuraArray
	// holder is the aura carrying the charges, nil when none is placed.
	holder *core.Aura
	// cast is the rank last cast, whose amount the charges spend.
	cast  *core.Spell
	entry healRank
}

// registerPrayerOfMending registers Prayer of Mending, which exists only
// with its talent. The cast places the charges on one ally; the next
// non-periodic heal that ally receives spends one, healing for the rank's
// amount, and the charges move to the most hurt other member of the raid.
//
// Assumption: the client also triggers it "the next time they take
// damage", but the fake raid's damage is plain health loss with no hook a
// healer can listen to (core.dealRaidDamage), so only the heal trigger is
// modelled. A holder nobody heals keeps its charges until they expire.
func (priest *Priest) registerPrayerOfMending() {
	if !priest.Talents.PrayerOfMending {
		return
	}
	pom := &prayerOfMending{priest: priest}
	pom.auras = priest.NewRaidAuraArray(func(target *core.Unit) *core.Aura {
		return target.GetOrRegisterAura(core.Aura{
			Label:     "Prayer of Mending",
			ActionID:  core.ActionID{SpellID: prayerOfMendingAuraID},
			Duration:  prayerOfMendingAuraLasts,
			MaxStacks: prayerOfMendingCharges,
			OnHealTaken: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, _ *core.SpellResult) {
				if spell != pom.cast {
					pom.spendCharge(sim, aura)
				}
			},
		})
	})
	cooldown := core.Cooldown{Timer: priest.NewTimer(), Duration: prayerOfMendingCooldown}
	priest.PrayerOfMending = priest.registerHealRanks(prayerOfMendingRanks, func(rank int, entry healRank) core.SpellConfig {
		config := priest.healSpellConfig(entry, rank, SpellCode_PriestPrayerOfMending, PriestSpellMaskPrayerOfMending)
		config.Cast.CD = cooldown
		config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			pom.cast, pom.entry = spell, entry
			pom.place(sim, pom.auras.Get(target), prayerOfMendingCharges)
		}
		return config
	})
}

// place puts charges on aura's unit, taking them off the previous holder:
// a priest keeps Prayer of Mending on one ally at a time.
func (pom *prayerOfMending) place(sim *core.Simulation, aura *core.Aura, charges int32) {
	if pom.holder != nil {
		pom.holder.Deactivate(sim)
	}
	pom.holder = aura
	aura.Activate(sim)
	aura.SetStacks(sim, charges)
}

// spendCharge heals the holder and passes the remaining charges on. The
// aura comes off first so the heal it deals cannot spend it again.
func (pom *prayerOfMending) spendCharge(sim *core.Simulation, aura *core.Aura) {
	remaining := aura.GetStacks() - 1
	aura.Deactivate(sim)
	pom.holder = nil
	healed := aura.Unit
	pom.priest.healTarget(sim, pom.cast, healed, pom.entry.effect.Center(int(pom.priest.Level)))
	if remaining > 0 {
		if next := pom.mostHurtOther(healed); next != nil {
			pom.place(sim, pom.auras.Get(next), remaining)
		}
	}
}

// mostHurtOther is the raid player other than from with the lowest health
// share, nil when no other player has a health bar.
func (pom *prayerOfMending) mostHurtOther(from *core.Unit) *core.Unit {
	var hurt *core.Unit
	for _, unit := range pom.priest.Env.Raid.AllPlayerUnits {
		if unit == from || !unit.HasHealthBar() {
			continue
		}
		if hurt == nil || unit.CurrentHealthPercent() < hurt.CurrentHealthPercent() {
			hurt = unit
		}
	}
	return hurt
}
