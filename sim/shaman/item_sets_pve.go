package shaman

import (
	"slices"
	"time"

	"github.com/wowsims/classic/sim/common/clientsetbonus"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

// The Tier 2.5 and Tier 3 bonuses read from the client's rows by hand.
const (
	stormcallerChainHealBonus     int32 = 26122
	earthshattererManaSpringBonus int32 = 29171
	chainHealClientFamilyBit            = 1 << 8
)

// chainHealClassMasks says which engine spell the client family of
// Stormcaller's Chain Heal bonus is.
var chainHealClassMasks = core.ClassMaskTable{
	{Client: core.ClientClassMask{chainHealClientFamilyBit}, Engine: ShamanSpellMaskChainHeal},
}

var ItemSetTheEarthfury = core.NewItemSet(core.ItemSet{
	Name: "The Earthfury",
	Bonuses: map[int32]core.ApplyEffect{
		// The radius of your totems that affect friendly targets is increased to 30 yd.
		3: func(agent core.Agent) {
			// Nothing to do
		},
		// After casting your Healing Wave or Lesser Healing Wave spell, gives you a 25% chance to gain Mana equal to 35% of the base cost of the spell.
		5: func(agent core.Agent) {
			// Nothing to do
		},
		// Your Healing Wave will now jump to additional nearby targets. Each jump reduces the effectiveness of the heal by 80%, and the spell will jump to up to two additional targets.
		8: func(agent core.Agent) {
			// Nothing to do
		},
	},
})

var ItemSetTheTenStorms = core.NewItemSet(core.ItemSet{
	Name: "The Ten Storms",
	Bonuses: map[int32]core.ApplyEffect{
		// Increases the amount healed by Chain Heal to targets beyond the first by 30%.
		3: func(agent core.Agent) {
			// Nothing to do
		},
		// Improves your chance to get a critical strike with Nature spells by 3%.
		5: func(agent core.Agent) {
			shaman := agent.(ShamanAgent).GetShaman()
			shaman.PseudoStats.SchoolBonusCritChance[stats.SchoolIndexNature] += 3 * core.CritRatingPerCritChance
		},
		// When you cast a Healing Wave or Lesser Healing Wave, there is a 25% chance the target also receives a free Lightning Shield that causes 50 Nature damage to attacker on hit.
		8: func(agent core.Agent) {
			// Nothing to do
		},
	},
})

var ItemSetStormcallersGarb = core.NewItemSet(core.ItemSet{
	Name: "Stormcaller's Garb",
	Bonuses: map[int32]core.ApplyEffect{
		// Your Lightning Bolt, Chain Lightning, and Shock spells have a 20% chance to grant up to 50 Nature damage to spells for 8 sec. (Proc chance: 20%)
		3: func(agent core.Agent) {
			shaman := agent.(ShamanAgent).GetShaman()

			buffAura := shaman.RegisterAura(core.Aura{
				ActionID: core.ActionID{SpellID: 26121},
				Label:    "Stormcaller's Wrath",
				Duration: time.Second * 8,
				OnGain: func(aura *core.Aura, sim *core.Simulation) {
					shaman.AddStatDynamic(sim, stats.NaturePower, 50)
				},
				OnExpire: func(aura *core.Aura, sim *core.Simulation) {
					shaman.AddStatDynamic(sim, stats.NaturePower, -50)
				},
			})

			affectedSpellCodes := []int32{SpellCode_ShamanLightningBolt, SpellCode_ShamanChainLightning, SpellCode_ShamanEarthShock, SpellCode_ShamanFlameShock, SpellCode_ShamanFrostShock}

			core.MakeProcTriggerAura(&shaman.Unit, core.ProcTrigger{
				Name:     "Stormcaller Spelldamage Bonus",
				Callback: core.CallbackOnSpellHitDealt,
				Outcome:  core.OutcomeLanded,
				Handler: func(sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
					if slices.Contains(affectedSpellCodes, spell.SpellCode) && sim.Proc(0.20, "Stormcaller's Wrath") {
						buffAura.Activate(sim)
					}
				},
			})
		},
		// -0.4 seconds on the casting time of your Chain Heal spell.
		5: func(agent core.Agent) {
			agent.GetCharacter().AddStaticMod(clientsetbonus.TableMod(stormcallerChainHealBonus, chainHealClassMasks))
		},
	},
})

var ItemSetGiftOfTheGatheringStorm = core.NewItemSet(core.ItemSet{
	Name: "Gift of the Gathering Storm",
	Bonuses: map[int32]core.ApplyEffect{
		// Increases the chain target damage multiplier of your Chain Lightning spell by 5%.
		3: func(agent core.Agent) {
			shaman := agent.(ShamanAgent).GetShaman()
			shaman.RegisterAura(core.Aura{
				Label: "Gift of the Gathering Storm Chain Lightning Bonus",
				OnInit: func(aura *core.Aura, sim *core.Simulation) {
					shaman.ChainLightningBounceCoefficient += 0.05
				},
			})
		},
	},
})

var ItemSetTheEarthshatterer = core.NewItemSet(core.ItemSet{
	Name: "The Earthshatterer",
	Bonuses: map[int32]core.ApplyEffect{
		// Reduces the mana cost of your totem spells by 12%.
		2: func(agent core.Agent) {
			shaman := agent.(ShamanAgent).GetShaman()
			shaman.OnSpellRegistered(func(spell *core.Spell) {
				if spell.Flags.Matches(SpellFlagTotem) {
					spell.Cost.Multiplier -= 12
				}
			})
		},
		// Increases the mana gained from your Mana Spring totems by 25%.
		4: func(agent core.Agent) {
			agent.(ShamanAgent).GetShaman().manaSpringSetBonus = clientsetbonus.PercentModifier(earthshattererManaSpringBonus, clientsetbonus.ModOpAllEffects)
		},
		// Your Healing Wave and Lesser Healing Wave spells have a chance to imbue your target with Totemic Power.
		6: func(agent core.Agent) {
		},
		// Your Lightning Shield spell also grants you 15 mana per 5 sec. while active.
		8: func(agent core.Agent) {
			shaman := agent.(ShamanAgent).GetShaman()
			shaman.RegisterAura(core.Aura{
				Label: "Lightning Shield",
				OnInit: func(_ *core.Aura, _ *core.Simulation) {
					for _, lsAura := range shaman.LightningShieldAuras {
						if lsAura == nil {
							return
						}

						oldOnGain := lsAura.OnGain
						lsAura.OnGain = func(aura *core.Aura, sim *core.Simulation) {
							oldOnGain(aura, sim)
							shaman.AddStatDynamic(sim, stats.MP5, 15)
						}

						oldOnExpire := lsAura.OnExpire
						lsAura.OnExpire = func(aura *core.Aura, sim *core.Simulation) {
							oldOnExpire(aura, sim)
							shaman.AddStatDynamic(sim, stats.MP5, -15)
						}
					}
				},
			})
		},
	},
})
