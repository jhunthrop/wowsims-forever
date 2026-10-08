package warrior

import (
	"math"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

const ShoutExpirationThreshold = time.Second * 3

// battleShoutDuration is the client's duration for every Battle Shout rank
// (spell 6673 and its ranks: "Lasts $d" at 180000 ms); vanilla's 2 minutes
// is gone, and Booming Voice no longer lengthens it in Forever's tree.
const battleShoutDuration = 3 * time.Minute

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
		ActionID:       actionID,
		ClassSpellMask: WarriorSpellMaskBattleShout,
		Flags:          core.SpellFlagNoOnCastComplete | core.SpellFlagAPL | core.SpellFlagHelpful,

		RequiredLevel: core.BattleShoutLevel[rank],

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

// battleShoutAllyAura is core.BattleShoutAura (sim/core/buffs.go),
// keyed to the RANK this warrior's own level has actually learned
// instead of that function's own hardcoded top rank. Without this, a
// levelling warrior's Battle Shout cast under its correct rank id
// (registerBattleShout, above) while the buff it grants stayed
// registered under core.BattleShoutAura's permanent top-rank id, so
// the rotation's own "is Battle Shout up" check (auraIsActive against
// the rank it just cast, since spellranks.json's Battle Shout chain
// rewrites that condition's id the same way it rewrites the cast)
// could never find a match - reported as an unresolved id at every
// level below 60. Duplicated here rather than editing
// core.BattleShoutAura because this lane's brief does not permit
// sim/core changes beyond the energy-bar guard; Improved Battle Shout
// (impBattleShout) is dropped because the Forever client's talent
// trees don't carry it (core.BattleShoutAura's own call site already
// passes 0 for it, per the FOREVER comment below).
func battleShoutAllyAura(unit *core.Unit, actionID core.ActionID, baseAP float64, has3pcWrath bool) *core.Aura {
	return unit.GetOrRegisterAura(core.Aura{
		Label:      "Battle Shout",
		ActionID:   actionID,
		Duration:   battleShoutDuration,
		BuildPhase: core.CharacterBuildPhaseBuffs,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.AddStatsDynamic(sim, stats.Stats{
				stats.AttackPower: math.Floor(baseAP + core.TernaryFloat64(has3pcWrath, 30, 0)),
			})
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			aura.Unit.AddStatsDynamic(sim, stats.Stats{
				stats.AttackPower: -1 * math.Floor(baseAP+core.TernaryFloat64(has3pcWrath, 30, 0)),
			})
		},
	})
}

// battleShoutGrant is the rank a warrior of this level has learned (the
// rank id its rotation finds, which the ladder rewrites to the learned
// rank; the AQ phase flag still caps the ceiling at 60), and the attack
// power it grants: the client's amount for that rank, growing by the
// client's points per level inside the rank (core.BattleShoutRankTable),
// not the vanilla AQ-era 232.
func battleShoutGrant(level int32) (rank int32, spellID int32, attackPower float64) {
	rank = min(int32(rankAtLevel(core.BattleShoutLevel[:], level)), core.TernaryInt32(core.IncludeAQ, 7, 6))
	return rank, core.BattleShoutSpellId[rank], core.BattleShoutAttackPower(int(level))
}

func (warrior *Warrior) registerBattleShout() {
	rank, actionId, baseAP := battleShoutGrant(warrior.Level)
	has3pcWrath := warrior.HasSetBonus(ItemSetBattleGearOfWrath, 3)

	warrior.BattleShout = warrior.newShoutSpellConfig(core.ActionID{SpellID: actionId}, rank, warrior.NewPartyAuraArray(func(unit *core.Unit) *core.Aura {
		// FOREVER: Improved Battle Shout is not in the client's trees.
		return battleShoutAllyAura(unit, core.ActionID{SpellID: actionId}, baseAP, has3pcWrath)
	}))
}

func (warrior *Warrior) registerShouts() {
	warrior.registerBattleShout()
}
