package mage

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

const (
	// Fire Ruby (20036) has no item effect here. The engine previously
	// registered wowsims-classic's Season of Discovery rework of this item
	// (a mana restore plus a "Chaos Fire" +100 Fire Power aura consumed by
	// the next Fire spell, spell ID 24389). Forever's own Era client text
	// for 20036 is "Refreshes the cooldown of Fire Ward and causes Fire
	// damage absorbed by it to increase the damage of your next Fire Blast
	// cast within 1 min by 50% of the damage absorbed." -- an absorb-driven
	// effect with no DPS to model, so it is intentionally left unregistered
	// rather than crediting a rework Forever does not grant.
	MindQuickeningGem     = 19339
	HazzarahsCharmOfMagic = 19959
	JewelOfKajaro         = 19601
)

func init() {
	core.AddEffectsToTest = false

	// https://www.wowhead.com/classic/item=19959/hazzarahs-charm-of-magic
	// Increases the critical hit chance of your Arcane spells by 5%, and increases the critical hit damage of your Arcane spells by 50% for 20 sec.
	// (3 Min Cooldown)
	core.NewItemEffect(HazzarahsCharmOfMagic, func(agent core.Agent) {
		mage := agent.(MageAgent).GetMage()

		duration := time.Second * 20
		affectedSpells := []*core.Spell{}

		aura := mage.RegisterAura(core.Aura{
			ActionID: core.ActionID{SpellID: 24544},
			Label:    "Arcane Potency",
			Duration: duration,
			OnInit: func(aura *core.Aura, sim *core.Simulation) {
				for spellIdx := range mage.Spellbook {
					if spell := mage.Spellbook[spellIdx]; spell.SpellSchool == core.SpellSchoolArcane {
						affectedSpells = append(affectedSpells, spell)
					}
				}
			},
			OnGain: func(aura *core.Aura, sim *core.Simulation) {
				for _, spell := range affectedSpells {
					spell.BonusCritRating += 5 * core.CritRatingPerCritChance
					spell.CritDamageBonus += 0.50
				}
			},
			OnExpire: func(aura *core.Aura, sim *core.Simulation) {
				for _, spell := range affectedSpells {
					spell.BonusCritRating -= 5 * core.CritRatingPerCritChance
					spell.CritDamageBonus -= 0.50
				}
			},
		})

		spell := mage.RegisterSpell(core.SpellConfig{
			ActionID:    core.ActionID{ItemID: HazzarahsCharmOfMagic},
			SpellSchool: core.SpellSchoolArcane,
			Flags:       core.SpellFlagNoOnCastComplete | core.SpellFlagOffensiveEquipment,
			Cast: core.CastConfig{
				CD: core.Cooldown{
					Timer:    mage.NewTimer(),
					Duration: time.Minute * 3,
				},
				SharedCD: core.Cooldown{
					Timer:    mage.GetOffensiveTrinketCD(),
					Duration: duration,
				},
			},
			ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				aura.Activate(sim)
			},
		})

		mage.AddMajorCooldown(core.MajorCooldown{
			Spell:    spell,
			Priority: core.CooldownPriorityBloodlust,
			Type:     core.CooldownTypeDPS,
		})
	})

	// https://www.wowhead.com/classic/item=19601/jewel-of-kajaro
	// Equip: Reduces the cooldown of Counterspell by 2 sec.
	core.NewItemEffect(JewelOfKajaro, func(agent core.Agent) {
		mage := agent.(MageAgent).GetMage()

		mage.RegisterAura(core.Aura{
			Label: "Improved Counterspell",
			OnInit: func(aura *core.Aura, sim *core.Simulation) {
				mage.Counterspell.CD.Duration -= time.Second * 2
			},
		})
	})

	// https://www.wowhead.com/classic/item=19339/mind-quickening-gem
	// Use: Quickens the mind, increasing the Mage's casting speed of non-channeled spells by 33% for 20 sec. (2 Min Cooldown)
	core.NewItemEffect(MindQuickeningGem, func(agent core.Agent) {
		mage := agent.(MageAgent).GetMage()

		actionID := core.ActionID{ItemID: MindQuickeningGem}
		duration := time.Second * 20

		buffAura := mage.RegisterAura(core.Aura{
			ActionID: actionID,
			Label:    "Mind Quickening",
			Duration: duration,
		}).AttachMultiplyCastSpeed(&mage.Unit, 1.33)

		spell := mage.RegisterSpell(core.SpellConfig{
			ActionID: actionID,
			Flags:    core.SpellFlagNoOnCastComplete | core.SpellFlagOffensiveEquipment,
			Cast: core.CastConfig{
				CD: core.Cooldown{
					Timer:    mage.NewTimer(),
					Duration: time.Minute * 5,
				},
				SharedCD: core.Cooldown{
					Timer:    mage.GetOffensiveTrinketCD(),
					Duration: duration,
				},
			},
			ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				buffAura.Activate(sim)
			},
		})

		mage.AddMajorCooldown(core.MajorCooldown{
			Spell:    spell,
			Priority: core.CooldownPriorityBloodlust,
			Type:     core.CooldownTypeDPS,
		})
	})

	core.AddEffectsToTest = true
}
