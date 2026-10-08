package druid

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Item IDs
const (
	WolfsheadHelm       = 8345
	IdolOfFerocity      = 22397
	IdolOfTheMoon       = 23197
	IdolOfBrutality     = 23198
	RuneOfMetamorphosis = 19340
	TalonsOfWrath       = 249441
	IdolOfTheDream      = 220606
	HowlingIdol         = 272427
)

// Talons of Wrath: client spell 1248996 "Improved Wrath" procs at its
// SpellAuraOptions.ProcChance (50) and triggers spell 1302521, whose energize
// effect restores 35 mana.
const (
	talonsOfWrathProcChance = 0.5
	talonsOfWrathMana       = 35.0
	talonsOfWrathManaSpell  = 1302521
)

// relicClassMasks says which engine spells each client spell family is, for
// the idols in relic_mods_auto_gen.go (SpellClassOptions masks of Rip 9896
// and Tiger's Fury 5217). An idol whose family is missing here (Swiftmend,
// Enrage, Healing Touch) cannot be registered, and core.NewEquipModItemEffect
// refuses it. Swarming Idol (272430) is deliberately absent: its text names
// Insect Swarm but its client mask is Rip's, so it is left unmodelled until
// the client settles which spell it reaches.
var relicClassMasks = core.ClassMaskTable{
	{Client: core.ClientClassMask{0, 0, 1 << 21}, Engine: DruidSpellMaskRip},
	{Client: core.ClientClassMask{0, 0, 1 << 11}, Engine: DruidSpellMaskTigersFury},
}

func init() {
	core.AddEffectsToTest = false

	// https://www.wowhead.com/classic/item=22397/idol-of-ferocity
	// Equip: Reduces the energy cost of Claw and Rake by 3.
	core.NewItemEffect(IdolOfFerocity, func(agent core.Agent) {
		druid := agent.(DruidAgent).GetDruid()

		druid.OnSpellRegistered(func(spell *core.Spell) {
			if spell.SpellCode == SpellCode_DruidRake || spell.SpellCode == SpellCode_DruidClaw {
				spell.Cost.FlatModifier -= 3
			}
		})
	})

	// https://www.wowhead.com/classic/item=23197/idol-of-the-moon
	// Equip: Increases the damage of your Moonfire spell by up to 33.
	core.NewItemEffect(IdolOfTheMoon, func(agent core.Agent) {
		druid := agent.(DruidAgent).GetDruid()
		druid.OnSpellRegistered(func(spell *core.Spell) {
			if spell.SpellCode == SpellCode_DruidMoonfire {
				spell.BonusDamage += 33
			}
		})
	})

	// Client spell 446212. Equip: Increases the duration of Rip by 2 sec.
	core.NewEquipModItemEffect(IdolOfTheDream, relicEquipMods[IdolOfTheDream], relicClassMasks)

	// Client spell 1291059. Equip: Reduces the cooldown of your Tiger's Fury
	// ability by 3 sec.
	core.NewEquipModItemEffect(HowlingIdol, relicEquipMods[HowlingIdol], relicClassMasks)

	// Client spell 1248996. Equip: Causes Wrath to have a 50% chance to
	// restore 35 Mana.
	core.NewItemEffect(TalonsOfWrath, func(agent core.Agent) {
		druid := agent.(DruidAgent).GetDruid()
		manaMetrics := druid.NewManaMetrics(core.ActionID{SpellID: talonsOfWrathManaSpell})
		core.MakePermanent(druid.RegisterAura(core.Aura{
			Label: "Talons of Wrath",
			OnSpellHitDealt: func(_ *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
				if !spell.Matches(DruidSpellMaskWrath) || !result.Landed() {
					return
				}
				if sim.Proc(talonsOfWrathProcChance, "Talons of Wrath") {
					druid.AddMana(sim, talonsOfWrathMana, manaMetrics)
				}
			},
		}))
	})

	// https://www.wowhead.com/classic/item=23198/idol-of-brutality
	// Equip: Reduces the rage cost of Maul and Swipe by 3.
	core.NewItemEffect(IdolOfBrutality, func(agent core.Agent) {
		druid := agent.(DruidAgent).GetDruid()
		druid.OnSpellRegistered(func(spell *core.Spell) {
			if spell.SpellCode == SpellCode_DruidMaul || spell.SpellCode == SpellCode_DruidSwipe {
				spell.Cost.FlatModifier -= 3
			}
		})
	})

	// https://www.wowhead.com/classic/item=19340/rune-of-metamorphosis
	// Use: Decreases the mana cost of all Druid shapeshifting forms by 100% for 20 sec. (5 Min Cooldown)
	core.NewItemEffect(RuneOfMetamorphosis, func(agent core.Agent) {
		druid := agent.(DruidAgent).GetDruid()

		actionID := core.ActionID{SpellID: 23724}
		duration := time.Second * 20
		cooldown := time.Minute * 5

		buffAura := druid.GetOrRegisterAura(core.Aura{
			ActionID: actionID,
			Label:    "Metamorphosis Rune",
			Duration: duration,
			OnGain: func(aura *core.Aura, sim *core.Simulation) {
				druid.CatForm.Cost.Multiplier -= 100
				//druid.BearForm.Cost.Multiplier -= 100
				//druid.MoonkinForm.Cost.Multiplier -= 100
			},
			OnExpire: func(aura *core.Aura, sim *core.Simulation) {
				druid.CatForm.Cost.Multiplier += 100
				//druid.BearForm.Cost.Multiplier += 100
				//druid.MoonkinForm.Cost.Multiplier += 100
			},
		})

		spell := druid.GetOrRegisterSpell(core.SpellConfig{
			ActionID: actionID,
			Flags:    core.SpellFlagNoOnCastComplete | core.SpellFlagOffensiveEquipment,
			Cast: core.CastConfig{
				CD: core.Cooldown{
					Timer:    druid.NewTimer(),
					Duration: cooldown,
				},
			},
			ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				buffAura.Activate(sim)
			},
		})

		druid.AddMajorCooldown(core.MajorCooldown{
			Spell: spell,
			Type:  core.CooldownTypeDPS,
		})
	})

	core.AddEffectsToTest = true
}
