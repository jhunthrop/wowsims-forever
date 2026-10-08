package warlock

import (
	"github.com/wowsims/classic/sim/common/clientsetbonus"
	"github.com/wowsims/classic/sim/core"
)

// Plagueheart Raiment is the Tier 1 warlock set (client ItemSet 529).
const (
	plagueheartSetID           int32 = 529
	plagueheartVampirismBonus  int32 = 28831
	plagueheartCorruptionBonus int32 = 28829
	plagueheartThreatBonus     int32 = 28746
	plagueheartLifeTapBonus    int32 = 28830

	plagueheartVampirismReason = "heals the wearer on a Shadow Bolt; the sim measures no healing taken"
)

var ItemSetPlagueheartRaiment = core.NewClientItemSet(core.ClientSetModel{
	ID: plagueheartSetID,
	Effects: map[int32]core.ApplyEffect{
		// Increases damage caused by your Corruption by 12%.
		plagueheartCorruptionBonus: applyPlagueheartCorruption,
		// Corruption, Immolate, Curse of Agony and Siphon Life generate
		// 25% less threat. The row's second effect, 25% less threat from
		// spell critical hits, is not modelled: the engine has no
		// per-crit threat.
		plagueheartThreatBonus: applyPlagueheartThreat,
		// Reduces the health cost of your Life Tap by 12%.
		plagueheartLifeTapBonus: applyPlagueheartLifeTap,
	},
	NoSim: map[int32]string{plagueheartVampirismBonus: plagueheartVampirismReason},
})

func applyPlagueheartCorruption(agent core.Agent) {
	warlock := agent.(WarlockAgent).GetWarlock()
	bonus := clientsetbonus.PercentModifier(plagueheartCorruptionBonus, clientsetbonus.ModOpDot)
	warlock.RegisterAura(core.Aura{
		Label: "Corruption (Plagueheart Raiment)",
		OnInit: func(aura *core.Aura, sim *core.Simulation) {
			for _, spell := range warlock.Corruption {
				spell.DamageMultiplierAdditive += bonus
			}
		},
	})
}

func applyPlagueheartThreat(agent core.Agent) {
	warlock := agent.(WarlockAgent).GetWarlock()
	threat := 1 + clientsetbonus.PercentModifier(plagueheartThreatBonus, clientsetbonus.ModOpThreat)
	warlock.RegisterAura(core.Aura{
		Label: "Plagueheart",
		OnInit: func(aura *core.Aura, sim *core.Simulation) {
			for _, spells := range [][]*core.Spell{warlock.Corruption, warlock.Immolate, warlock.CurseOfAgony, warlock.SiphonLife} {
				for _, spell := range spells {
					spell.ThreatMultiplier *= threat
				}
			}
		},
	})
}

func applyPlagueheartLifeTap(agent core.Agent) {
	warlock := agent.(WarlockAgent).GetWarlock()
	warlock.lifeTapHealthDiscount = -clientsetbonus.PercentModifier(plagueheartLifeTapBonus, core.ClientModOpCost)
}
