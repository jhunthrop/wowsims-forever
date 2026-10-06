package warrior

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// bloodthrillProcChance is Bloodthrill's own per-rank chance: "a
// 4%/8%/12%/16%/20% chance to allow the use of your Overpower ability"
// at ranks 1-5 (Arms node 110524), so a flat 4% a point. Kept separate
// from the trigger below so the per-rank number is checkable without
// rolling RNG.
func bloodthrillProcChance(rank int32) float64 {
	return 0.04 * float64(rank)
}

// bloodthrillWindow is "Lasts 6 sec" - distinct from the vanilla
// dodge-proc Overpower window (OverpowerAura's 5s in overpower.go), so
// Bloodthrill gets its own Aura rather than reusing that one with a
// mutated Duration.
const bloodthrillWindow = time.Second * 6

// applyBloodthrill is Bloodthrill: "Your Main Hand melee attacks
// against enemies afflicted by your Rend have a 4%/8%/12%/16%/20%
// chance to allow the use of your Overpower ability on the target."
//
// warrior.Rend.Dot(target).IsActive() is the "afflicted by your Rend"
// check: Rend is a snapshot Dot (rend.go), and Dot embeds *Aura, so
// IsActive reads the same ticking state the bleed itself runs on rather
// than a second, separately-tracked flag that could drift from it.
func (warrior *Warrior) applyBloodthrill() {
	if warrior.Talents.Bloodthrill == 0 {
		return
	}

	procChance := bloodthrillProcChance(warrior.Talents.Bloodthrill)

	warrior.BloodthrillAura = warrior.RegisterAura(core.Aura{
		Label:    "Bloodthrill Aura",
		ActionID: core.ActionID{SpellID: TalentSpellIDs["bloodthrill"][0]},
		Duration: bloodthrillWindow,
	})

	warrior.RegisterAura(core.Aura{
		Label:    "Bloodthrill Trigger",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if !result.Landed() || !spell.ProcMask.Matches(core.ProcMaskMeleeMH) {
				return
			}
			if !warrior.Rend.Dot(result.Target).IsActive() {
				return
			}
			if sim.Proc(procChance, "Bloodthrill") {
				warrior.BloodthrillAura.Activate(sim)
			}
		},
	})
}
