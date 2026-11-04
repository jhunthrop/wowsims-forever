package hunter

import (
	"strconv"
	"time"

	"github.com/wowsims/classic/sim/core"
)

const RaptorStrikeRanks = 8

var RaptorStrikeSpellId = [RaptorStrikeRanks + 1]int32{0, 2973, 14260, 14261, 14262, 14263, 14264, 14265, 14266}
var RaptorStrikeSpellIdMeleeSpecialist = [RaptorStrikeRanks + 1]int32{0, 415335, 415336, 415337, 415338, 415340, 415341, 415342, 415343}

// RaptorStrikeBaseDamage and RaptorStrikeManaCost are ranks 1-8's flat
// bonus damage and mana cost; source: 1.60.1.70009 client spell data
// ("Raptor Strike", spells 2973/14260-14266). Forever's own numbers are
// lower than vanilla Classic's from rank 4 on (rank 8 is +70, not the
// +140 vanilla gave), and rank 1's cost is 10, not 15; both corrected
// here against the client rather than carrying vanilla's table forward.
var RaptorStrikeManaCost = [RaptorStrikeRanks + 1]float64{0, 10, 25, 35, 45, 55, 70, 85, 100}
var RaptorStrikeLevel = [RaptorStrikeRanks + 1]int{0, 1, 8, 16, 24, 32, 40, 48, 56}

// Returns true if the regular melee swing should be used, false otherwise.
func (hunter *Hunter) TryRaptorStrike(sim *core.Simulation, mhSwingSpell *core.Spell) *core.Spell {
	if hunter.curQueuedAutoSpell != nil && hunter.curQueuedAutoSpell.CanCast(sim, hunter.CurrentTarget) {
		return hunter.curQueuedAutoSpell
	}
	return mhSwingSpell
}

// getRaptorStrikeConfig is rank's on-next-swing cast. cdTimer is the one
// cooldown every rank shares.
func (hunter *Hunter) getRaptorStrikeConfig(rank int, cdTimer *core.Timer) core.SpellConfig {
	spellID := RaptorStrikeSpellId[rank]
	manaCost := RaptorStrikeManaCost[rank]
	level := RaptorStrikeLevel[rank]

	hit := hunter.newRaptorStrikeHitSpell(rank)
	hunter.RaptorStrikeHit = hit

	spellConfig := core.SpellConfig{
		SpellCode:     SpellCode_HunterRaptorStrike,
		ActionID:      core.ActionID{SpellID: spellID},
		SpellSchool:   core.SpellSchoolPhysical,
		DefenseType:   core.DefenseTypeMelee,
		ProcMask:      core.ProcMaskMeleeMHSpecial | core.ProcMaskMeleeMHAuto,
		Flags:         core.SpellFlagMeleeMetrics | SpellFlagStrike,
		Rank:          rank,
		RequiredLevel: level,

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},

		Cast: core.CastConfig{
			CD: core.Cooldown{
				Timer:    cdTimer,
				Duration: time.Second * 6,
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return hunter.DistanceFromTarget <= core.MaxMeleeAttackDistance
		},
		ClientBaseDamage: RaptorStrikeDamage[rank].Range(int(hunter.Level)),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			hit.Cast(sim, target)

			if hunter.curQueueAura != nil {
				hunter.curQueueAura.Deactivate(sim)
			}
		},
	}

	return spellConfig
}

func (hunter *Hunter) newRaptorStrikeHitSpell(rank int) *core.Spell {
	spellID := RaptorStrikeSpellId[rank]
	flatDamage := RaptorStrikeDamage[rank]
	casterLevel := int(hunter.Level)

	return hunter.RegisterSpell(core.SpellConfig{
		SpellCode:   SpellCode_HunterRaptorStrikeHit,
		ActionID:    core.ActionID{SpellID: spellID}.WithTag(1),
		SpellSchool: core.SpellSchoolPhysical,
		DefenseType: core.DefenseTypeMelee,
		ProcMask:    core.ProcMaskMeleeMHSpecial,
		// SpellFlagPassiveSpell: this inner hit shares its ActionID's
		// SpellID with the real Raptor Strike cast above (only the Tag
		// differs), the exact shape compare.go's rowFor doc comment
		// warns about (Death Coil's self-heal sub-spell). Without this
		// flag, rowFor kept a second, all-zero-valued row for this
		// SpellID alongside the real one -- the client-facing numbers
		// (cost, cooldown_ms, required_level) were never actually
		// missing, they were just shadowed by this bogus row.
		Flags: core.SpellFlagMeleeMetrics | core.SpellFlagNoOnCastComplete | core.SpellFlagPassiveSpell,

		BonusCritRating:  hunter.savageStrikesCritRating(),
		CritDamageBonus:  hunter.predatorsEdgeCritDamage(),
		DamageMultiplier: 1,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			damage := flatDamage.Roll(sim, casterLevel) + hunter.MHWeaponDamage(sim, spell.MeleeAttackPower(target))
			spell.CalcAndDealDamage(sim, target, damage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)
		},
	})
}

// makeQueueSpellsAndAura is raptorStrike's queue: the APL cast (tag 3)
// that arms it on the next main-hand swing.
func (hunter *Hunter) makeQueueSpellsAndAura(raptorStrike *core.Spell) *core.Spell {
	queueAura := hunter.RegisterAura(core.Aura{
		Label:    "Raptor Strike Queued " + strconv.Itoa(raptorStrike.Rank),
		ActionID: raptorStrike.ActionID,
		Duration: core.NeverExpires,
		OnGain: func(aura *core.Aura, sim *core.Simulation) {
			if hunter.curQueueAura != nil {
				hunter.curQueueAura.Deactivate(sim)
			}
			hunter.PseudoStats.DisableDWMissPenalty = true
			hunter.curQueueAura = aura
			hunter.curQueuedAutoSpell = raptorStrike
		},
		OnExpire: func(aura *core.Aura, sim *core.Simulation) {
			hunter.PseudoStats.DisableDWMissPenalty = false
			hunter.curQueueAura = nil
			hunter.curQueuedAutoSpell = nil
		},
	})

	queueSpell := hunter.RegisterSpell(core.SpellConfig{
		SpellCode: SpellCode_HunterRaptorStrike,
		ActionID:  raptorStrike.WithTag(3),
		// SpellFlagPassiveSpell: like RaptorStrikeHit above, this
		// queue-trigger shares RaptorStrike's SpellID (WithTag only
		// changes the Tag), the same shape compare.go's rowFor doc
		// comment describes. It deals no damage and sets no Cost or
		// RequiredLevel of its own -- the real cast it queues up is
		// hunter.RaptorStrike (tag 0), tracked under its own ActionID
		// -- so without this flag it left a second bogus all-zero row
		// under this SpellID even after the Hit sub-spell's fix.
		Flags: core.SpellFlagMeleeMetrics | core.SpellFlagAPL | core.SpellFlagPassiveSpell,

		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return hunter.curQueueAura != queueAura &&
				hunter.CurrentMana() >= raptorStrike.Cost.GetCurrentCost() &&
				!hunter.IsCasting(sim) &&
				hunter.DistanceFromTarget <= core.MaxMeleeAttackDistance &&
				raptorStrike.IsReady(sim)
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			queueAura.Activate(sim)
		},
	})
	queueSpell.CdSpell = raptorStrike

	return queueSpell
}

func (hunter *Hunter) registerRaptorStrikeSpell() {
	// RaptorStrikeLevel[1:] is already the rank -> learn-level table (source:
	// 1.60.1.70009 client spell data, "Raptor Strike", ranks 1-8); reuse it
	// instead of a second copy.
	//
	// Every learned rank is registered, as the spellbook keeps them, so a
	// rotation can queue a lower one; all share one cooldown, and
	// hunter.RaptorStrike (and RaptorStrikeHit) is the highest.
	top := core.HighestRankAtLevel(RaptorStrikeLevel[1:], hunter.Level)
	if top == 0 {
		return
	}

	cdTimer := hunter.NewTimer()
	for rank := 1; rank <= top; rank++ {
		spell := hunter.GetOrRegisterSpell(hunter.getRaptorStrikeConfig(rank, cdTimer))
		hunter.makeQueueSpellsAndAura(spell)
		hunter.RaptorStrike = spell
	}
}
