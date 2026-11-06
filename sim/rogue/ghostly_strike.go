package rogue

import (
	"time"

	"github.com/wowsims/classic/sim/core/stats"

	"github.com/wowsims/classic/sim/core"
)

// Ghostly Strike (14278, 70755): e0 125% weapon damage (e3 180% with a
// main-hand dagger), normalized by e4 (effect 121), e1 +15% dodge for 7 sec, e2 one combo point, and e5
// (effect 64) triggers 1329326, "Next Ambush or Garrote does not require
// Stealth": aura 275 on mask 768 (Garrote 256 + Ambush 512), DurationIndex
// 165 (7 sec).
const (
	ghostlyStrikeOpenerSpellID  = 1329326
	ghostlyStrikeOpenerDuration = 7 * time.Second
)

func (rogue *Rogue) registerGhostlyStrikeSpell() {
	if !rogue.Talents.GhostlyStrike {
		return
	}

	ghostlyStrikeAura := rogue.RegisterAura(core.Aura{
		Label:    "Ghostly Strike Buff",
		ActionID: core.ActionID{SpellID: 14278},
		Duration: time.Second * 7,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			rogue.AddStatDynamic(sim, stats.Dodge, 15*core.DodgeRatingPerDodgeChance)
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			rogue.AddStatDynamic(sim, stats.Dodge, -15*core.DodgeRatingPerDodgeChance)
		},
	})

	rogue.GhostlyStrikeOpenerAura = rogue.RegisterAura(core.Aura{
		Label:    "Ghostly Strike (Opener)",
		ActionID: core.ActionID{SpellID: ghostlyStrikeOpenerSpellID},
		Duration: ghostlyStrikeOpenerDuration,
	})

	rogue.GhostlyStrike = rogue.RegisterSpell(core.SpellConfig{
		SpellCode:      SpellCode_RogueGhostlyStrike,
		ClassSpellMask: RogueSpellMaskGhostlyStrike,
		ActionID:       ghostlyStrikeAura.ActionID,
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          rogue.builderFlags(),

		// See talents.go's Adrenaline Rush RelatedSelfBuff comment.
		RelatedSelfBuff: ghostlyStrikeAura,

		EnergyCost: core.EnergyCostOptions{
			Cost:   40.0,
			Refund: 0.8,
		},

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: time.Second,
			},
			CD: core.Cooldown{
				Timer:    rogue.NewTimer(),
				Duration: time.Second * 20,
			},
			IgnoreHaste: true,
		},

		CritDamageBonus: rogue.lethality(),

		DamageMultiplier: rogue.mainHandStrikePct(ghostlyStrikeWeaponDamagePct, ghostlyStrikeDaggerWeaponDamagePct),
		ThreatMultiplier: 1,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			rogue.BreakStealth(sim)
			baseDamage := rogue.ghostlyStrikeWeaponDamage(sim, spell.MeleeAttackPower(target))

			result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)

			ghostlyStrikeAura.Activate(sim)

			if result.Landed() {
				rogue.AddComboPoints(sim, 1, target, spell.ComboPointMetrics())
				rogue.GhostlyStrikeOpenerAura.Activate(sim)
			} else {
				spell.IssueRefund(sim)
			}
		},
	})
}

// canOpen reports whether Ambush or Garrote may be cast now: in Stealth,
// or with Ghostly Strike's opener buff up.
func (rogue *Rogue) canOpen() bool {
	return rogue.IsStealthed() || rogue.hasGhostlyStrikeOpener()
}

func (rogue *Rogue) hasGhostlyStrikeOpener() bool {
	return rogue.GhostlyStrikeOpenerAura != nil && rogue.GhostlyStrikeOpenerAura.IsActive()
}

// spendOpener breaks Stealth for an Ambush or Garrote; one cast outside
// Stealth spends Ghostly Strike's buff. It is checked before BreakStealth,
// which would otherwise make "not stealthed" ambiguous between "never
// stealthed" and "was stealthed, now broken".
func (rogue *Rogue) spendOpener(sim *core.Simulation) {
	if !rogue.IsStealthed() && rogue.hasGhostlyStrikeOpener() {
		rogue.GhostlyStrikeOpenerAura.Deactivate(sim)
	}
	rogue.BreakStealth(sim)
}

// ghostlyStrikeWeaponDamage is Ghostly Strike's weapon damage before its 125%
// (180% with a dagger) multiplier: normalized. The 70755 client added effect
// 121 (normalized weapon damage, SpellEffect 1368764, amount 0) to 14278 in
// the same change that added the opener trigger (e5), the shape Hemorrhage
// has (hemorrhageWeaponDamage).
func (rogue *Rogue) ghostlyStrikeWeaponDamage(sim *core.Simulation, attackPower float64) float64 {
	return rogue.MHNormalizedWeaponDamage(sim, attackPower)
}
