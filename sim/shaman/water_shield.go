package shaman

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Water Shield is the Restoration tree's tier-2 bool talent
// (talents/shaman.json node 104732, max_rank 1): "The caster is
// surrounded by 3 globes of water. When a spell, melee, or ranged
// attack hits the caster or when one of the caster's healing spells
// gets a critical result, 2% of maximum mana is restored to the
// caster, expending one water globe. Only one globe will activate
// every few seconds... Lasts 10 min. Only one Elemental Shield can be
// active on the Shaman at any one time." Implemented because the
// instructions call it out explicitly as a mana sustain source for
// Elemental and Enhancement's rotation ("Water Shield if the rotation
// keeps it up"), mirroring lightning_shield.go's shield-swap pattern
// (shaman.ActiveShield / ActiveShieldAura) rather than duplicating it.
//
// The "few seconds" ICD and the 3-globe charge count are both
// datamined, not stated by the tooltip: spellconst/shaman.json spell
// 408510's SpellCooldowns.csv row gives CategoryRecoveryTime 15000ms,
// and its SpellAuraOptions.csv row gives ProcCharges 3. The constants
// generator's own "Water Shield"-named slot (WaterShieldSpellId,
// constants_auto_gen.go) resolved to 408511 - a second, differently
// numbered spell also named "Water Shield" that is the 2%-of-mana
// proc effect itself (its own single effect's amount is exactly 2),
// not the castable buff - so the buff's ActionID below uses the
// literal 408510 that talents_auto_gen.go's TalentSpellIDs already
// established as this talent's display id, and WaterShieldSpellId[0]
// is used for the mana-restore proc's own metrics id instead.
const (
	waterShieldBuffSpellId = 408510
	waterShieldMaxCharges  = int32(3)
	waterShieldManaPercent = 0.02
	waterShieldICD         = time.Second * 15
	waterShieldDuration    = time.Minute * 10
)

func (shaman *Shaman) registerWaterShieldSpell() {
	if !shaman.Talents.WaterShield {
		return
	}

	actionID := core.ActionID{SpellID: waterShieldBuffSpellId}
	manaMetrics := shaman.NewManaMetrics(core.ActionID{SpellID: WaterShieldSpellId[0]})
	icd := core.Cooldown{Timer: shaman.NewTimer(), Duration: waterShieldICD}

	restoreMana := func(sim *core.Simulation, aura *core.Aura) {
		if aura.GetStacks() == 0 || !icd.IsReady(sim) {
			return
		}
		icd.Use(sim)
		shaman.AddMana(sim, shaman.MaxMana()*waterShieldManaPercent, manaMetrics)
		aura.RemoveStack(sim)
	}

	waterShieldAura := shaman.RegisterAura(core.Aura{
		Label:     "Water Shield",
		ActionID:  actionID,
		Duration:  waterShieldDuration,
		MaxStacks: waterShieldMaxCharges,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			aura.SetStacks(sim, waterShieldMaxCharges)
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			if shaman.ActiveShieldAura == aura {
				shaman.ActiveShieldAura = nil
				shaman.ActiveShield = nil
			}
		},
		// Unlike Lightning Shield's own OnSpellHitTaken (ProcMaskMelee
		// only), the tooltip names "a spell, melee, or ranged attack" -
		// every incoming attack type - so this has no ProcMask filter.
		OnSpellHitTaken: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if result.Landed() {
				restoreMana(sim, aura)
			}
		},
		OnSpellHitDealt: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			if spell.ProcMask.Matches(core.ProcMaskSpellHealing) && result.Outcome.Matches(core.OutcomeCrit) {
				restoreMana(sim, aura)
			}
		},
	})

	shaman.WaterShield = shaman.RegisterSpell(core.SpellConfig{
		ActionID:        actionID,
		Flags:           SpellFlagShaman | core.SpellFlagAPL,
		RelatedSelfBuff: waterShieldAura,
		Cast: core.CastConfig{
			DefaultCast: core.Cast{GCD: core.GCDDefault},
		},
		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			if shaman.ActiveShieldAura != nil {
				shaman.ActiveShieldAura.Deactivate(sim)
			}
			shaman.ActiveShield = shaman.WaterShield
			shaman.ActiveShieldAura = waterShieldAura
			waterShieldAura.Activate(sim)
		},
	})
}
