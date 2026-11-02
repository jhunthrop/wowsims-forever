package item_effects

import (
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

// Blackhand's Breadth (13965) and Eye of the Beast (13968) have no
// ItemSparse row in 1.60.1.70707, but the client links both to Forever's own
// effects. Their equip crit (1% melee and ranged, 1318954; 1% spell,
// 1318956) is simdb's. Their use effects are these: each lasts 20 sec
// (SpellDuration 18) on a 5 minute cooldown (ItemEffect CoolDownMSec
// 300000) and the 20 sec trinket category (CategoryCoolDownMSec 20000).
//
//   - Blackhand's Breadth, ItemEffect 231535 -> 1318944: aura 306, "Marks
//     your target, increasing your Critical Strike chance against them by
//     5% for 20 sec. This effect is doubled against Orcs and Ogres." Against
//     the one target, crit on every table; the type-neutral target is no
//     Orc or Ogre.
//   - Eye of the Beast, ItemEffect 231536 -> 1287840: aura 55, "Increases
//     your Spell Hit chance by 7% for 20 sec." Spell hit only, so it is the
//     magic schools' hit bonus.
const (
	BlackhandsBreadth = 13965
	EyeOfTheBeast     = 13968

	blackrockTrinketDuration = 20 * time.Second
	blackrockTrinketCooldown = 5 * time.Minute

	blackhandsBreadthCritPct = 5.0
	eyeOfTheBeastSpellHitPct = 7.0
)

var magicSchools = []stats.SchoolIndex{
	stats.SchoolIndexArcane,
	stats.SchoolIndexFire,
	stats.SchoolIndexFrost,
	stats.SchoolIndexHoly,
	stats.SchoolIndexNature,
	stats.SchoolIndexShadow,
}

func init() {
	core.NewSimpleStatOffensiveTrinketEffect(
		BlackhandsBreadth,
		stats.Stats{stats.Crit: blackhandsBreadthCritPct * core.CritRatingPerCritChance},
		blackrockTrinketDuration,
		blackrockTrinketCooldown,
	)

	core.NewItemEffect(EyeOfTheBeast, func(agent core.Agent) {
		character := agent.GetCharacter()
		actionID := core.ActionID{ItemID: EyeOfTheBeast}
		hit := eyeOfTheBeastSpellHitPct * core.HitRatingPerHitChance
		aura := character.GetOrRegisterAura(core.Aura{
			Label:    "Eye of the Beast",
			ActionID: actionID,
			Duration: blackrockTrinketDuration,
			OnGain: func(_ *core.Aura, _ *core.Simulation) {
				for _, school := range magicSchools {
					character.PseudoStats.SchoolBonusHitChance[school] += hit
				}
			},
			OnExpire: func(_ *core.Aura, _ *core.Simulation) {
				for _, school := range magicSchools {
					character.PseudoStats.SchoolBonusHitChance[school] -= hit
				}
			},
		})
		spell := character.RegisterSpell(core.SpellConfig{
			ActionID: actionID,
			Flags:    core.SpellFlagNoOnCastComplete | core.SpellFlagOffensiveEquipment,
			Cast: core.CastConfig{
				CD: core.Cooldown{
					Timer:    character.NewTimer(),
					Duration: blackrockTrinketCooldown,
				},
				SharedCD: core.Cooldown{
					Timer:    character.GetOffensiveTrinketCD(),
					Duration: blackrockTrinketDuration,
				},
			},
			ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
				aura.Activate(sim)
			},
		})
		character.AddMajorCooldown(core.MajorCooldown{
			Type:  core.CooldownTypeDPS,
			Spell: spell,
		})
	})
}
