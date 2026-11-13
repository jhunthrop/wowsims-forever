package mage

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// IgniteTicks is the burn's two 2 s ticks over 4 s (client 412538).
const IgniteTicks = 2

func (mage *Mage) applyIgnite() {
	if mage.Talents.Ignite == 0 {
		return
	}
	newIgniteDamage := 0.0
	igniteBurn := 0.0

	// burn adds a crit's share to Ignite on target.
	burn := func(sim *core.Simulation, target *core.Unit) {
		// Forever's Ignite rolls (client 412538: one aura,
		// CumulativeAura 1): a new crit adds its share to what the
		// burn has left and spreads the sum over a fresh 4 s.
		dot := mage.igniteTick.Dot(target)
		remaining := 0.0
		if dot.IsActive() {
			remaining = dot.SnapshotBaseDamage * float64(dot.MaxTicksRemaining())
		}
		igniteBurn = (remaining + newIgniteDamage*IgniteTicks) / IgniteTicks
		dot.Apply(sim)
	}

	mage.RegisterAura(core.Aura{
		Label:    "Ignite Talent",
		Duration: core.NeverExpires,
		OnReset: func(aura *core.Aura, sim *core.Simulation) {
			aura.Activate(sim)
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if !spell.ProcMask.Matches(core.ProcMaskSpellDamage) {
				return
			}
			if spell.SpellSchool.Matches(core.SpellSchoolFire) && result.DidCrit() {
				newIgniteDamage = result.Damage * 0.08 * float64(mage.Talents.Ignite) / IgniteTicks
				burn(sim, result.Target)
			}
		},
	})

	mage.igniteTick = mage.RegisterSpell(core.SpellConfig{
		SpellCode:      SpellCode_MageIgnite,
		ClassSpellMask: MageSpellMaskIgnite,
		ActionID:       core.ActionID{SpellID: 12654},
		SpellSchool:    core.SpellSchoolFire,
		DefenseType:    core.DefenseTypeMagic,
		ProcMask:       core.ProcMaskSpellProc,
		// The burn is a share of a crit that already took the target's
		// percentage modifiers (Improved Scorch, Curse of the Elements);
		// Blizzard's beta note "Ignite no longer double dips on % damage
		// increase modifiers" keeps the ticks from taking them again.
		Flags: core.SpellFlagNoOnCastComplete | core.SpellFlagPassiveSpell | SpellFlagMage | core.SpellFlagIgnoreTargetModifiers,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		Cast: core.CastConfig{
			IgnoreHaste: true,
		},

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label:    "Ignite",
				Tag:      mageFireEffectTag,
				Duration: time.Second * 4,
			},
			// 412538 is a dummy periodic aura (226); each pulse casts the
			// server tick 412545, which the client marks Can't Crit
			// (SPELL_ATTR2_CANT_CRIT, Attributes_2 0x20000004, 70888 and
			// 70938), as Deep Wound's 412613 is. The flag overrides the
			// owner's true-dot rule (crit-flags sweep, 2026-11-13).
			NeverCrits:    true,
			NumberOfTicks: IgniteTicks,
			TickLength:    time.Second * 2,

			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
				if isRollover {
					return
				}
				// igniteBurn is a per-tick share of crits' finished
				// damage, which already carries every attacker multiplier
				// (Fire Power, Arcane Power): Forever's 24 September 2026
				// beta note "Ignite: no longer counts damage bonuses
				// twice" keeps the ticks' own multiplier at 1.
				dot.SnapshotAttackerMultiplier = 1
				dot.SnapshotBaseDamage = igniteBurn
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
			},
		},
	})
}
