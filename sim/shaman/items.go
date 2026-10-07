package shaman

import (
	"slices"
	"time"

	"github.com/wowsims/classic/sim/core"
)

const (
	// Keep these ordered by ID
	NaturalAlignmentCrystal  = 19344
	WushoolaysCharmOfSpirits = 19956
	TotemOfRage              = 22395
	TotemOfTheStorm          = 23199
	TotemOfThunder           = 228176
	TotemOfTheStormForever   = 272432
	BurningTotem             = 272433
)

// totemOfTheStormMaelstromReduction is the 50 in client spell 1291078
// ("Your Lightning Bolt ability can now also trigger the Maelstrom Weapon
// talent, but with a $s1% reduced chance"), as a fraction.
const totemOfTheStormMaelstromReduction = 0.5

// relicClassMasks says which engine spells each client spell family is, for
// the totems in relic_mods_auto_gen.go (SpellClassOptions masks of Lightning
// Bolt 403, Chain Lightning 421 and Flame Shock 8050). A totem whose family
// is missing here (healing spells, Grounding Totem) cannot be registered,
// and core.NewEquipModItemEffect refuses it.
var relicClassMasks = core.ClassMaskTable{
	{Client: core.ClientClassMask{1 << 0}, Engine: ShamanSpellMaskLightningBolt},
	{Client: core.ClientClassMask{1 << 1}, Engine: ShamanSpellMaskChainLightning},
	{Client: core.ClientClassMask{1 << 28}, Engine: ShamanSpellMaskFlameShock},
}

func init() {
	core.AddEffectsToTest = false

	// Keep these ordered by name

	// https://www.wowhead.com/classic/item=19344/natural-alignment-crystal
	// Use: Aligns the Shaman with nature, increasing the damage done by spells by 20%, improving heal effects by 20%, and increasing mana cost of spells by 20% for 20 sec.
	// (5 Min Cooldown)
	core.NewItemEffect(NaturalAlignmentCrystal, func(agent core.Agent) {
		shaman := agent.(ShamanAgent).GetShaman()

		duration := time.Second * 20

		aura := shaman.RegisterAura(core.Aura{
			ActionID: core.ActionID{ItemID: NaturalAlignmentCrystal},
			Label:    "Nature Aligned",
			Duration: duration,
			OnGain: func(aura *core.Aura, sim *core.Simulation) {
				shaman.PseudoStats.SchoolDamageDealtMultiplier.MultiplyMagicSchools(1.20)
				shaman.PseudoStats.HealingDealtMultiplier *= 1.20
				shaman.PseudoStats.SchoolCostMultiplier.AddToAllSchools(20)
			},
			OnExpire: func(aura *core.Aura, sim *core.Simulation) {
				shaman.PseudoStats.SchoolDamageDealtMultiplier.MultiplyMagicSchools(1 / 1.20)
				shaman.PseudoStats.HealingDealtMultiplier /= 1.20
				shaman.PseudoStats.SchoolCostMultiplier.AddToAllSchools(-20)
			},
		})

		spell := shaman.RegisterSpell(core.SpellConfig{
			ActionID: core.ActionID{ItemID: NaturalAlignmentCrystal},
			ProcMask: core.ProcMaskEmpty,
			Flags:    core.SpellFlagNoOnCastComplete | core.SpellFlagOffensiveEquipment,
			Cast: core.CastConfig{
				CD: core.Cooldown{
					Timer:    shaman.NewTimer(),
					Duration: time.Minute * 5,
				},
				SharedCD: core.Cooldown{
					Timer:    shaman.GetOffensiveTrinketCD(),
					Duration: duration,
				},
			},
			ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				aura.Activate(sim)
			},
		})

		shaman.AddMajorCooldown(core.MajorCooldown{
			Spell:    spell,
			Priority: core.CooldownPriorityBloodlust,
			Type:     core.CooldownTypeDPS,
		})
	})

	// https://www.wowhead.com/classic/item=23199/totem-of-the-storm
	// Equip: Increases damage done by Chain Lightning and Lightning Bolt by up to 33.
	core.NewItemEffect(TotemOfTheStorm, func(agent core.Agent) {
		shaman := agent.(ShamanAgent).GetShaman()
		shaman.OnSpellRegistered(func(spell *core.Spell) {
			if spell.SpellCode == SpellCode_ShamanLightningBolt || spell.SpellCode == SpellCode_ShamanChainLightning {
				spell.BonusDamage += 33
			}
		})
	})

	// Totem of Rage
	// Equip: Increases damage done by Earth Shock, Flame Shock, and Frost Shock by up to 30.
	// Acts as extra 30 spellpower for shocks.
	core.NewItemEffect(TotemOfRage, func(agent core.Agent) {
		shaman := agent.(ShamanAgent).GetShaman()
		affectedSpellCodes := []int32{SpellCode_ShamanEarthShock, SpellCode_ShamanFlameShock, SpellCode_ShamanFrostShock}
		shaman.OnSpellRegistered(func(spell *core.Spell) {
			if slices.Contains(affectedSpellCodes, spell.SpellCode) {
				spell.BonusDamage += 30
			}
		})
	})

	// Client spell 461295. Equip: Increases the critical strike chance of
	// Lightning Bolt by 1%.
	core.NewEquipModItemEffect(TotemOfThunder, relicEquipMods[TotemOfThunder], relicClassMasks)

	// Client spell 1291077. Equip: Increases the duration of your Flame Shock
	// ability by 3 sec.
	core.NewEquipModItemEffect(BurningTotem, relicEquipMods[BurningTotem], relicClassMasks)

	// Client spell 1291078. Equip: Your Lightning Bolt ability can now also
	// trigger the Maelstrom Weapon talent, but with a 50% reduced chance.
	core.NewItemEffect(TotemOfTheStormForever, func(agent core.Agent) {
		agent.(ShamanAgent).GetShaman().LightningBoltMaelstromChance = maelstromWeaponProcChance * (1 - totemOfTheStormMaelstromReduction)
	})

	// https://www.wowhead.com/classic/item=19956/wushoolays-charm-of-spirits
	// Use: Increases the damage dealt by your Lightning Shield spell by 100% for 20 sec. (3 Min Cooldown)
	core.NewItemEffect(WushoolaysCharmOfSpirits, func(agent core.Agent) {
		shaman := agent.(ShamanAgent).GetShaman()

		duration := time.Second * 20
		actionID := core.ActionID{ItemID: WushoolaysCharmOfSpirits}

		var affectedSpells []*core.Spell

		aura := shaman.RegisterAura(core.Aura{
			ActionID: actionID,
			Label:    "Wushoolay's Charm of Spirits",
			Duration: time.Second * 20,
			OnInit: func(aura *core.Aura, sim *core.Simulation) {
				affectedSpells = core.FilterSlice(
					shaman.LightningShieldProcs,
					func(spell *core.Spell) bool { return spell != nil },
				)
			},
			OnGain: func(aura *core.Aura, sim *core.Simulation) {
				for _, spell := range affectedSpells {
					spell.DamageMultiplier *= 2
				}
			},
			OnExpire: func(aura *core.Aura, sim *core.Simulation) {
				for _, spell := range affectedSpells {
					spell.DamageMultiplier /= 2
				}
			},
		})

		spell := shaman.RegisterSpell(core.SpellConfig{
			ActionID:    actionID,
			SpellSchool: core.SpellSchoolNature,
			ProcMask:    core.ProcMaskEmpty,
			Flags:       core.SpellFlagNoOnCastComplete | core.SpellFlagOffensiveEquipment,
			Cast: core.CastConfig{
				CD: core.Cooldown{
					Timer:    shaman.NewTimer(),
					Duration: time.Minute * 3,
				},
				SharedCD: core.Cooldown{
					Timer:    shaman.GetOffensiveTrinketCD(),
					Duration: duration,
				},
			},
			ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				aura.Activate(sim)
			},
		})

		shaman.AddMajorCooldown(core.MajorCooldown{
			Spell:    spell,
			Priority: core.CooldownPriorityDefault,
			Type:     core.CooldownTypeDPS,
		})
	})

	core.AddEffectsToTest = true
}
