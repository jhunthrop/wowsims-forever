package druid

import (
	"fmt"
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

// healingEffect is one effect of a healing spell: the client's roll for it
// and the share of the caster's healing power added to it.
type healingEffect struct {
	roll        clientdamage.Effect
	coefficient float64
}

// healingRank is one learnable rank of a healing spell as the client states
// it (the tables are in healing_tables.go).
type healingRank struct {
	spellID  int32
	manaCost float64
	castTime time.Duration
	// heal is the direct heal; the zero value for a pure heal over time.
	heal healingEffect
	// tick is the periodic heal of one tick; the zero value for a direct heal.
	tick healingEffect
}

// level is the level the rank is learned at, the level of whichever effect
// the rank has.
func (rank healingRank) level() int {
	if rank.heal != (healingEffect{}) {
		return rank.heal.roll.SpellLevel
	}
	return rank.tick.roll.SpellLevel
}

// reported is the effect the conformance report compares for the rank: the
// direct heal when it has one, otherwise its periodic heal.
func (rank healingRank) reported() healingEffect {
	if rank.heal != (healingEffect{}) {
		return rank.heal
	}
	return rank.tick
}

// registerHealingRanks registers the ranks of table the druid's level has
// learned. The slice it returns is indexed by rank (index 0 is unused), like
// Wrath and Starfire, with nil for a rank not learned yet.
func (druid *Druid) registerHealingRanks(table []healingRank, build func(rank int, spec healingRank) core.SpellConfig) []*DruidSpell {
	spells := make([]*DruidSpell, len(table)+1)
	for index, spec := range table {
		if spec.level() > int(druid.Level) {
			break
		}
		spells[index+1] = druid.RegisterSpell(Humanoid, build(index+1, spec))
	}
	return spells
}

// healingSpellConfig is what every rank of a healing spell shares: a Nature
// heal that costs its flat mana and takes the global cooldown.
func (druid *Druid) healingSpellConfig(rank int, spec healingRank, code int32, mask uint64) core.SpellConfig {
	reported := spec.reported()
	return core.SpellConfig{
		ActionID:       core.ActionID{SpellID: spec.spellID},
		SpellCode:      code,
		ClassSpellMask: mask,
		SpellSchool:    core.SpellSchoolNature,
		DefenseType:    core.DefenseTypeMagic,
		ProcMask:       core.ProcMaskSpellHealing,
		Flags:          core.SpellFlagHelpful | core.SpellFlagAPL,

		RequiredLevel: spec.level(),
		Rank:          rank,

		ManaCost: core.ManaCostOptions{FlatCost: spec.manaCost},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD:      core.GCDDefault,
				CastTime: spec.castTime,
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: reported.coefficient,
		ClientBaseDamage: reported.roll.Range(int(druid.Level)),
	}
}

// directHealEffects heals the target for the rank's roll plus its share of
// healing power. The heal can critically strike.
func (druid *Druid) directHealEffects(spec healingRank) core.ApplySpellResults {
	casterLevel := int(druid.Level)
	return func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		spell.CalcAndDealHealing(sim, target, spec.heal.roll.Roll(sim, casterLevel), spell.OutcomeHealingCrit)
	}
}

// healOverTimeConfig is the periodic half of a heal: numTicks ticks of the
// rank's tick effect, tickLength apart, each landing as one heal.
//
// Periodic heals do not critically strike: no source states that Forever's
// do (research/08-stats.md section 1.1), and core.DotConfig.CanCrit is the
// switch for when the beta says they do.
func (druid *Druid) healOverTimeConfig(name string, rank int, spec healingRank, numTicks int32, tickLength time.Duration) core.DotConfig {
	tickCenter := spec.tick.roll.Center(int(druid.Level))
	return core.DotConfig{
		Aura:             core.Aura{Label: fmt.Sprintf("%s (Rank %d)", name, rank)},
		NumberOfTicks:    numTicks,
		TickLength:       tickLength,
		BonusCoefficient: spec.tick.coefficient,
		OnSnapshot: func(_ *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
			snapshotHealing(dot, target, tickCenter, isRollover)
		},
		OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
			dot.CalcAndDealPeriodicSnapshotHealing(sim, target, dot.OutcomeTick)
		},
	}
}

// snapshotHealing fixes what each tick of a heal over time lands for: the
// tick's roll plus its share of healing power, and the caster's healing
// multiplier together with the periodic bonus (Genesis).
//
// core.Dot.SnapshotHeal is not used: it multiplies by the caster's damage
// multipliers (Moonfury's school bonus, Naturalist's damage), which are not a
// healer's, and leaves the healing multiplier (Gift of Nature) out.
func snapshotHealing(dot *core.Dot, target *core.Unit, tickRoll float64, isRollover bool) {
	if isRollover {
		return
	}
	spell := dot.Spell
	dot.SnapshotBaseDamage = tickRoll + dot.BonusCoefficient*spell.HealingPower(target)
	dot.SnapshotAttackerMultiplier = spell.CasterHealingMultiplier() * spell.PeriodicDamageMultiplierAdditive * dot.DamageMultiplier
}

// eachPartyMember calls visit for every member of the target's party, the
// target included. A target that is in no party (a lone unit) is its own.
func (druid *Druid) eachPartyMember(target *core.Unit, visit func(member *core.Unit)) {
	members := druid.Env.Raid.GetPlayerParty(target).Players
	if len(members) == 0 {
		visit(target)
		return
	}
	for _, member := range members {
		visit(&member.GetCharacter().Unit)
	}
}

// RegisterHealingSpells registers the restoration druid's healing kit.
// Swiftmend and Wild Growth come with their talents.
func (druid *Druid) RegisterHealingSpells() {
	druid.registerHealingTouchSpell()
	druid.registerRegrowthSpell()
	druid.registerRejuvenationSpell()
	druid.registerSwiftmendSpell()
	druid.registerTranquilitySpell()
	druid.registerWildGrowthSpell()
}
