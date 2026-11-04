package mage

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Chill effects and what rides on them. The client marks a spell as a
// Chill effect with SpellClassMask_0 bit 20 (1048576): Frostbolt, Cone of
// Cold, Frostfire Bolt and the Chilled aura Improved Blizzard applies.
// Two talents trigger off it: Fingers of Frost (fingers_of_frost.go) and
// Frostbite, below.
//
// Frostbite 11071, max rank 3: "Gives your Chill effects a 5/10/15%
// chance to Freeze the target for 5 sec." The spell row is an aura 109
// (proc trigger) with base points 15 for the top rank, class mask 1048576
// and trigger spell 12494; 12494 is the "Frozen" root, aura 26, duration
// index 28 = 5000 ms.
//
// Model decisions the rows do not carry:
//   - The root is the Frozen state Shatter and Ice Lance read, like Frost
//     Nova's root, and it shares Frost Nova's immunity rule: a target
//     above the player level cap (a raid boss) cannot be frozen
//     (canFreeze), so Frostbite changes nothing against a boss.
//   - The root does not break on damage; neither does Frost Nova's.
//   - Every Chill application rolls on its own, as Fingers of Frost does.
const (
	frostbiteSpellId       int32 = 12494
	frostbiteDuration            = 5 * time.Second
	frostbiteChancePerRank       = 0.05
)

func (mage *Mage) applyFrostbite() {
	if mage.Talents.Frostbite == 0 {
		return
	}
	mage.FrostbiteFrozenAuras = mage.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
		return target.GetOrRegisterAura(core.Aura{
			Label:    "Frostbite",
			ActionID: core.ActionID{SpellID: frostbiteSpellId},
			Duration: frostbiteDuration,
		})
	})
}

// applyChillTriggers wires the talents that trigger off a landed Chill
// effect: every damage spell flagged SpellFlagChillSpell that lands rolls
// them. Blizzard's chill is a proc spell with no hit of its own and rolls
// from its own effect (blizzard.go).
func (mage *Mage) applyChillTriggers() {
	if mage.FingersOfFrostAura == nil && mage.FrostbiteFrozenAuras == nil {
		return
	}
	mage.RegisterAura(core.Aura{
		Label:    "Chill Triggers",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(_ *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if spell.ProcMask.Matches(core.ProcMaskSpellDamage) && spell.Flags.Matches(SpellFlagChillSpell) && result.Landed() {
				mage.chillApplied(sim, spell, result.Target)
			}
		},
	})
}

// chillApplied is one Chill application on target by spell.
func (mage *Mage) chillApplied(sim *core.Simulation, spell *core.Spell, target *core.Unit) {
	mage.rollFingersOfFrost(sim, mage.fingersOfFrostChance(spell))
	mage.rollFrostbite(sim, target)
}

// rollFrostbite is one Chill application's chance to freeze target.
func (mage *Mage) rollFrostbite(sim *core.Simulation, target *core.Unit) {
	if mage.FrostbiteFrozenAuras == nil || !mage.canFreeze(target) {
		return
	}
	chance := frostbiteChancePerRank * float64(mage.Talents.Frostbite)
	if sim.Proc(chance, "Frostbite") {
		mage.FrostbiteFrozenAuras.Get(target).Activate(sim)
	}
}
