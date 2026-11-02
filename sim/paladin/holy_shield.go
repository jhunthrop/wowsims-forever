package paladin

import (
	"strconv"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

// HolyShieldValues: client 1.60.1.70707 moved Holy Shield from the
// Protection tree's 31-point row to its 21-point row, and its talent text
// (rank 1) now deals 75 Holy damage a block where it dealt 110. Wowhead's
// live Forever tooltips (2026-10-31) show every rank one bracket down to
// match: 20925 level 30 for 75, 20927 level 40 for 110, 20928 level 50 for
// 153, so a level 60 paladin's top rank deals 153. The mana costs moved
// down with them: the 70707 hotfix push of 31 October (wago.tools/hotfixes)
// sets SpellPower 185892, 186383 and 184913 to 100, 150 and 195, the levels
// and amounts to the ones below, and Wowhead's live tooltips read the same.
// The client tables (spellconst/paladin.json) carry the same since the
// site's pipeline layers the server's push (round 6; the shipped rows read
// levels 40/50/60, 110/153/221 and 150/195/240 mana), and the 0.08
// spell-power coefficient is theirs.
//
// procID is the engine's own id for the damage spell: the client states
// the damage on the aura rather than on a spell of its own.
var HolyShieldValues = []struct {
	level    int32
	spellID  int32
	procID   int32
	manaCost float64
	damage   float64
}{
	{level: 30, spellID: 20925, procID: 20955, manaCost: 100, damage: 75},
	{level: 40, spellID: 20927, procID: 20956, manaCost: 150, damage: 110},
	{level: 50, spellID: 20928, procID: 20957, manaCost: 195, damage: 153},
}

const (
	// holyShieldCharges and holyShieldBlockChance are the Holy Shield
	// talent's text (node 105628): "Increases chance to block by 30% for
	// 10 sec, and deals 75 Holy damage for each attack blocked while
	// active. Damage caused by Holy Shield causes 20% additional threat.
	// Each block expends a charge. 4 charges." The rank spells' own
	// block effect reads 30 since build 1.60.1.70291 (it read 20 before).
	holyShieldCharges     = 4
	holyShieldBlockChance = 30.0
	holyShieldThreat      = 1.2
	holyShieldDuration    = 10 * time.Second
	holyShieldCoefficient = 0.08
)

func (paladin *Paladin) registerHolyShield() {
	if !paladin.Talents.HolyShield {
		return
	}

	blockBonus := holyShieldBlockChance * core.BlockRatingPerBlockChance

	for i, values := range HolyShieldValues {
		rank := i + 1
		level := values.level
		spellID := values.spellID
		procID := values.procID
		manaCost := values.manaCost
		damage := values.damage

		if paladin.Level < level {
			break
		}

		paladin.holyShieldProc[i] = paladin.RegisterSpell(core.SpellConfig{
			ActionID:    core.ActionID{SpellID: procID},
			SpellCode:   SpellCode_PaladinHolyShieldProc,
			SpellSchool: core.SpellSchoolHoly,
			DefenseType: core.DefenseTypeMagic,
			ProcMask:    core.ProcMaskSpellDamage,

			RequiredLevel: int(level),
			Rank:          rank,

			DamageMultiplier: 1,
			ThreatMultiplier: holyShieldThreat,
			BonusCoefficient: holyShieldCoefficient,

			ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				// Spell damage from Holy Shield can crit, but does not miss.
				spell.CalcAndDealDamage(sim, target, damage, spell.OutcomeMagicCrit)
			},
		})

		paladin.holyShieldAura[i] = paladin.RegisterAura(core.Aura{
			Label:     "Holy Shield" + paladin.Label + strconv.Itoa(rank),
			ActionID:  core.ActionID{SpellID: spellID},
			Duration:  holyShieldDuration,
			MaxStacks: holyShieldCharges,
			OnGain: func(aura *core.Aura, sim *core.Simulation) {
				paladin.AddStatDynamic(sim, stats.Block, blockBonus)
			},
			OnExpire: func(aura *core.Aura, sim *core.Simulation) {
				paladin.AddStatDynamic(sim, stats.Block, -blockBonus)
			},
			OnSpellHitTaken: func(aura *core.Aura, sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
				if result.DidBlock() {
					paladin.holyShieldProc[i].Cast(sim, spell.Unit)
					aura.RemoveStack(sim)
				}
			},
		})

		paladin.RegisterSpell(core.SpellConfig{
			ActionID:      core.ActionID{SpellID: spellID},
			SpellCode:     SpellCode_PaladinHolyShield,
			Flags:         core.SpellFlagAPL,
			RequiredLevel: int(level),
			Rank:          rank,

			// paladin.holyShieldAura[i]'s own Duration (10s) matches
			// the client's duration_ms (10000) for this cast's own
			// SpellID at every rank (spellconst/paladin.json) - wiring
			// it through lets compare.go's engineDuration see it
			// instead of reporting 0 for a self-buff that does exist.
			RelatedSelfBuff: paladin.holyShieldAura[i],
			// Holy Shield needs a shield: without one nothing blocks.
			ExtraCastCondition: func(_ *core.Simulation, _ *core.Unit) bool {
				return paladin.PseudoStats.CanBlock
			},
			// Benediction's class mask (20101, mask_1 0x40) names
			// Holy Shield like the other instants.
			ManaCost: core.ManaCostOptions{
				FlatCost:   manaCost,
				Multiplier: paladin.benediction(),
			},
			Cast: core.CastConfig{
				DefaultCast: core.Cast{
					GCD: core.GCDDefault,
				},
				CD: core.Cooldown{
					Timer:    paladin.NewTimer(),
					Duration: holyShieldDuration,
				},
			},
			ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
				// A recast brings the charges back as well as the duration.
				paladin.holyShieldAura[i].Activate(sim)
				paladin.holyShieldAura[i].SetStacks(sim, holyShieldCharges)
			},
		})
	}
}

// activeHolyShield is the index of the Holy Shield rank that is up, or -1.
func (paladin *Paladin) activeHolyShield() int {
	for i, aura := range paladin.holyShieldAura {
		if aura != nil && aura.IsActive() {
			return i
		}
	}
	return -1
}
