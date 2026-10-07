package paladin

import (
	"strconv"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

// HolyShieldValues: source 1.60.1.70009 client spell data
// (spellconst/paladin.json, ids 20925/20927/20928): effect 1 (aura 43, a
// proc-trigger damage) states the Holy damage a block deals, 110, 153 and
// 221 by rank, with an 0.08 spell-power coefficient. Rank 1's level is
// 40, not 30: the client moved it from Classic's 30. The vanilla 65/95/130
// the engine carried before is gone.
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
	{level: 40, spellID: 20925, procID: 20955, manaCost: 150, damage: 110},
	{level: 50, spellID: 20927, procID: 20956, manaCost: 195, damage: 153},
	{level: 60, spellID: 20928, procID: 20957, manaCost: 240, damage: 221},
}

const (
	// holyShieldCharges and holyShieldBlockChance are the Holy Shield
	// talent's text (node 105628): "Increases chance to block by 30% for
	// 10 sec, and deals 110 Holy damage for each attack blocked while
	// active. Damage caused by Holy Shield causes 20% additional threat.
	// Each block expends a charge. 4 charges." The rank spells' own
	// block effect reads 20; the live text says 30 and is the one used.
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
			ManaCost: core.ManaCostOptions{
				FlatCost: manaCost,
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
