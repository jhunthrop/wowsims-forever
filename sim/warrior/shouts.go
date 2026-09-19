package warrior

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

const ShoutExpirationThreshold = time.Second * 3

// battleShoutRageCost reads the client's cost column for a rank of
// Battle Shout. Rank 6 is the one rank whose column is a dedup
// artifact: 11551 (cost 100) and 27578 (cost 0) share spell_level 52
// and the generator kept the free one, so BattleShoutManaCost[6] is 0
// where every other rank says 100 tenths. That rank falls back to the
// 100 the client gives 11551; listed for the data lane.
func battleShoutRageCost(rank int32) float64 {
	tenths := BattleShoutManaCost[rank]
	if tenths == 0 {
		tenths = 100
	}
	return rageCost(tenths)
}

func (warrior *Warrior) newShoutSpellConfig(actionID core.ActionID, rank int32, allyAuras core.AuraArray) *WarriorSpell {
	// Use extra hits to simulate buffing your party for threat
	extraHits := 5 - len(allyAuras)

	return warrior.RegisterSpell(AnyStance, core.SpellConfig{
		ActionID: actionID,
		Flags:    core.SpellFlagNoOnCastComplete | core.SpellFlagAPL | core.SpellFlagHelpful,

		RageCost: core.RageCostOptions{
			Cost: battleShoutRageCost(rank),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			IgnoreHaste: true,
		},

		FlatThreatBonus: float64(core.BattleShoutLevel[rank]),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			for _, aura := range allyAuras {
				spell.CalcAndDealOutcome(sim, aura.Unit, spell.OutcomeAlwaysHit)
				aura.Activate(sim)
			}

			for i := 0; i < extraHits; i++ {
				spell.CalcAndDealOutcome(sim, target, spell.OutcomeAlwaysHit)
			}
		},

		RelatedAuras: []core.AuraArray{allyAuras},
	})
}

func (warrior *Warrior) registerBattleShout() {
	rank := core.TernaryInt32(core.IncludeAQ, 7, 6)
	actionId := core.BattleShoutSpellId[rank]
	has3pcWrath := warrior.HasSetBonus(ItemSetBattleGearOfWrath, 3)

	warrior.BattleShout = warrior.newShoutSpellConfig(core.ActionID{SpellID: actionId}, rank, warrior.NewPartyAuraArray(func(unit *core.Unit) *core.Aura {
		// FOREVER: Improved Battle Shout is not in the client's trees.
		// return core.BattleShoutAura(unit, warrior.Talents.ImprovedBattleShout, warrior.Talents.BoomingVoice, has3pcWrath)
		return core.BattleShoutAura(unit, 0, warrior.Talents.BoomingVoice, has3pcWrath)
	}))
}

func (warrior *Warrior) registerShouts() {
	warrior.registerBattleShout()
}
