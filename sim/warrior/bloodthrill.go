package warrior

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// bloodthrillProcChancePct is Bloodthrill's per-rank chance (Arms node
// 110524, 2 ranks since 1.60.1.70755): "a 10%/20% chance to allow an
// activation of Overpower". The client's trait curve 109366 on 1289682
// e0 gives 1: 10, 2: 20; the spell row's own base points (15) is the
// untraited default the curve replaces, not a rank's value.
var bloodthrillProcChancePct = [3]float64{0, 10, 20}

// bloodthrillProcChance is the chance at a rank. Kept separate from the
// trigger below so the per-rank number is checkable without rolling RNG.
func bloodthrillProcChance(rank int32) float64 {
	return bloodthrillProcChancePct[rankIndex(rank, bloodthrillProcChancePct[:])] / 100
}

// bloodthrillWindow is the Overpower window 1282733 opens: DurationIndex 28,
// 5000 ms in SpellDuration (the talent text's "6 sec" is not the client's
// duration). It is its own Aura rather than the dodge-proc Overpower window
// (OverpowerAura in overpower.go) with a mutated Duration.
const bloodthrillWindow = time.Second * 5

// applyBloodthrill is Bloodthrill: "Grants your Main Hand melee attacks
// against enemies afflicted by your Rend a 10%/20% chance to allow an
// activation of Overpower."
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
