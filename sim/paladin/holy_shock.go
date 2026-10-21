package paladin

import (
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

// holyShockRank is one rank of Holy Shock. The client splits a rank into
// three spells: the cast the player learns (castID: cost, cooldown and a
// dummy effect), the damage it deals an enemy (index 1 of HolyShockDamage)
// and the heal it gives an ally (healID).
type holyShockRank struct {
	castID   int32
	healID   int32
	level    int
	manaCost float64
	heal     clientdamage.Effect
	// number is the Rank the spell reports. The ladder the damage specs
	// have always registered starts at 20473 as rank 1 (the client calls it
	// rank 2), and renumbering it would change the damage specs' conformance
	// rows, so the first rank only the healing spec registers is rank 0.
	number int
	// healerOnly ranks are registered for the healing spec only.
	healerOnly bool
}

// holyShockRanks are the four ranks trainables "Holy Shock" teaches
// (levels 30, 40, 48, 56).
const holyShockRankCount = 4

var holyShockRanks = [holyShockRankCount]holyShockRank{
	{castID: 1311606, healID: 1311605, level: 30, manaCost: 160, number: 0, healerOnly: true,
		heal: clientdamage.Effect{Amount: 114, Variance: 0.075472, SpellLevel: 30, MaxLevel: 37}},
	{castID: 20473, healID: 25914, level: 40, manaCost: 225, number: 1,
		heal: clientdamage.Effect{Amount: 156, Variance: 0.075472, SpellLevel: 40, MaxLevel: 47}},
	{castID: 20929, healID: 25913, level: 48, manaCost: 275, number: 2,
		heal: clientdamage.Effect{Amount: 230, Variance: 0.075862, SpellLevel: 48, MaxLevel: 55}},
	{castID: 20930, healID: 25903, level: 56, manaCost: 325, number: 3,
		heal: clientdamage.Effect{Amount: 320, Variance: 0.078947, SpellLevel: 56, MaxLevel: 60}},
}

// HolyShockDamage is spellconst/paladin.json's own roll for the damage
// spells ids 1311604, 25912, 25911 and 25902 that the cast ids above
// trigger: 134, 182, 258 and 348 at their own levels (30, 40, 48, 56) and
// 7.5-7.9% wide, with no per-level growth. The Classic tooltip rolls these
// replaced (204-220, 279-301, 365-395) sat about 12% above them. Index i
// is holyShockRanks[i-1].
var HolyShockDamage = [holyShockRankCount + 1]clientdamage.Effect{
	{},
	{Amount: 134, Variance: 0.075472, SpellLevel: 30},
	{Amount: 182, Variance: 0.075472, SpellLevel: 40},
	{Amount: 258, Variance: 0.075862, SpellLevel: 48},
	{Amount: 348, Variance: 0.078947, SpellLevel: 56},
}

const (
	holyShockCooldown = 10 * time.Second
	// holyShockHealCoefficient is the sp_coefficient of every rank's heal.
	holyShockHealCoefficient = 0.429
)

func (paladin *Paladin) registerHolyShock() {
	if !paladin.Talents.HolyShock {
		return
	}

	// One cooldown for every rank: the ranks are one ability.
	cd := core.Cooldown{
		Timer: paladin.NewTimer(),
		// category_cooldown_ms is 10000 for every rank's actual cast spell
		// (spellconst/paladin.json ids 1311606/20473/20929/20930): source
		// 1.60.1.70009 client spell data. 30s was Classic's original Holy
		// Shock cooldown; Forever's client shortened it to 10s. Flagged by
		// sim/core/testdata/conformance/paladin.golden.md's "cooldown_ms
		// 10000->30000" row.
		Duration: holyShockCooldown,
	}

	for i, rank := range holyShockRanks {
		if paladin.Level < int32(rank.level) {
			break
		}
		if rank.healerOnly && !paladin.isHealer() {
			continue
		}
		paladin.registerHolyShockRank(rank, HolyShockDamage[i+1], cd)
	}
}

func (paladin *Paladin) registerHolyShockRank(rank holyShockRank, damage clientdamage.Effect, cd core.Cooldown) {
	casterLevel := int(paladin.Level)
	healSpell := paladin.registerHolyShockHeal(rank)

	paladin.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: rank.castID},
		SpellSchool: core.SpellSchoolHoly,
		DefenseType: core.DefenseTypeMagic,
		ProcMask:    core.ProcMaskSpellDamage,
		Flags:       core.SpellFlagMeleeMetrics | core.SpellFlagAPL,

		RequiredLevel: rank.level,
		Rank:          rank.number,

		SpellCode:      SpellCode_PaladinHolyShock,
		ClassSpellMask: PaladinSpellMaskHolyShock,

		ManaCost: core.ManaCostOptions{FlatCost: rank.manaCost},

		Cast: core.CastConfig{
			DefaultCast: core.Cast{GCD: core.GCDDefault},
			CD:          cd,
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: 0.429,
		ClientBaseDamage: damage.Range(casterLevel),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			if healSpell != nil && !target.IsOpponent(&paladin.Unit) {
				paladin.healWithHolyShock(sim, target, spell, healSpell)
				return
			}
			spell.CalcAndDealDamage(sim, target, damage.Roll(sim, casterLevel), spell.OutcomeMagicHitAndCrit)
		},
	})
}

// registerHolyShockHeal registers the heal half of a rank, for the
// healing spec only. It is a spell of its own, as it is in the client, so
// its healing is reported apart from the cast and talents can match it.
func (paladin *Paladin) registerHolyShockHeal(rank holyShockRank) *core.Spell {
	if !paladin.isHealer() {
		return nil
	}
	casterLevel := int(paladin.Level)
	spell := paladin.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: rank.healID},
		SpellSchool: core.SpellSchoolHoly,
		DefenseType: core.DefenseTypeMagic,
		ProcMask:    core.ProcMaskSpellHealing,
		Flags:       core.SpellFlagHelpful | core.SpellFlagNoOnCastComplete,

		RequiredLevel: rank.level,
		Rank:          rank.number,

		SpellCode:      SpellCode_PaladinHolyShockHeal,
		ClassSpellMask: PaladinSpellMaskHolyShockHeal,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: holyShockHealCoefficient,
		ClientBaseDamage: rank.heal.Range(casterLevel),
		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.CalcAndDealHealing(sim, target, rank.heal.Roll(sim, casterLevel), spell.OutcomeHealingCrit)
		},
	})
	paladin.markIlluminating(spell, rank.manaCost)
	return spell
}

// healWithHolyShock is the ally half of a Holy Shock cast: the heal, and
// the party heal a Light's Vigil on the target turns it into.
func (paladin *Paladin) healWithHolyShock(sim *core.Simulation, target *core.Unit, cast, heal *core.Spell) {
	heal.Cast(sim, target)
	paladin.triggerLightsVigil(sim, target, cast)
}
