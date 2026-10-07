package druid

import (
	"fmt"
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

const InsectSwarmRanks = 5

var InsectSwarmSpellId = [InsectSwarmRanks + 1]int32{0, 5570, 24974, 24975, 24976, 24977}

// InsectSwarmTickDamage is spellconst/druid.json's per-tick amount for the
// 2 s ticks of its 12 s DoT (rank 5: 31 at 0.158 a tick, 186 in all,
// against the Classic 324 this replaced).
var InsectSwarmTickDamage = [InsectSwarmRanks + 1]clientdamage.Effect{
	{},
	{Amount: 8, SpellLevel: 20},
	{Amount: 15, SpellLevel: 30},
	{Amount: 20, SpellLevel: 40},
	{Amount: 25, SpellLevel: 50},
	{Amount: 31, SpellLevel: 60},
}
var InsectSwarmTickSpellCoeff = [InsectSwarmRanks + 1]float64{0, .158, .158, .158, .158, .158}
var InsectSwarmManaCost = [InsectSwarmRanks + 1]float64{0, 45, 85, 100, 140, 160}
var InsectSwarmLevel = [InsectSwarmRanks + 1]int{0, 20, 30, 40, 50, 60}

func (druid *Druid) registerInsectSwarmSpell() {
	// Insect Swarm is a real, single-point Balance talent in this build
	// (data/builds/<build>/talents/druid.json, id 104930), not a
	// baseline spell every druid learns by level like Moonfire/Wrath -
	// this was the one thing the level-gate below did not check, so a
	// zero-talent character could already cast it fine (confirmed
	// against a level-60 zero-talent build: no unresolved warning
	// before this fix). Gated the same way every other real talent in
	// this file is (see talents.go's Talents.X == 0 early returns).
	if !druid.Talents.InsectSwarm {
		return
	}
	druid.InsectSwarm = make([]*DruidSpell, InsectSwarmRanks+1)

	druid.InsectSwarmAuras = druid.NewEnemyAuraArray(core.InsectSwarmAura)

	for rank := 1; rank <= InsectSwarmRanks; rank++ {
		level := InsectSwarmLevel[rank]
		if int32(level) <= druid.Level {
			numTicks := int32(6)
			tickLength := time.Second * 2

			spellID := InsectSwarmSpellId[rank]
			tickDamage := InsectSwarmTickDamage[rank]
			casterLevel := int(druid.Level)
			manaCost := InsectSwarmManaCost[rank]
			spellCoef := InsectSwarmTickSpellCoeff[rank]

			druid.InsectSwarm[rank] = druid.RegisterSpell(Humanoid|Moonkin, core.SpellConfig{
				SpellCode:      SpellCode_DruidInsectSwarm,
				ClassSpellMask: DruidSpellMaskInsectSwarm,
				ActionID:       core.ActionID{SpellID: spellID},
				SpellSchool:    core.SpellSchoolNature,
				DefenseType:    core.DefenseTypeMagic,
				ProcMask:       core.ProcMaskSpellDamage,
				Flags:          SpellFlagOmen | core.SpellFlagAPL | core.SpellFlagBinary,

				RequiredLevel: level,

				ManaCost: core.ManaCostOptions{
					FlatCost: manaCost,
				},
				Cast: core.CastConfig{
					DefaultCast: core.Cast{
						GCD: core.GCDDefault,
					},
				},

				DamageMultiplier: 1,
				ThreatMultiplier: 1,
				BonusCoefficient: spellCoef, // the report compares the spell's, which a pure DoT never reads
				ClientBaseDamage: tickDamage.Range(casterLevel),

				Dot: core.DotConfig{
					Aura: core.Aura{
						Label: fmt.Sprintf("Insect Swarm (Rank %d)", rank),
						OnGain: func(aura *core.Aura, sim *core.Simulation) {
							druid.InsectSwarmAuras.Get(aura.Unit).Activate(sim)
						},
						OnExpire: func(aura *core.Aura, sim *core.Simulation) {
							insectSwarmAura := druid.InsectSwarmAuras.Get(aura.Unit)
							if !insectSwarmAura.IsPermanent() {
								insectSwarmAura.Deactivate(sim)
							}
						},
					},

					NumberOfTicks:    numTicks,
					TickLength:       tickLength,
					BonusCoefficient: spellCoef,

					OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
						dot.Snapshot(target, tickDamage.Center(casterLevel), isRollover)
						if !druid.form.Matches(Moonkin) {
							dot.SnapshotCritChance = 0
						}
					},
					OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
						dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
					},
				},

				ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
					result := spell.CalcOutcome(sim, target, spell.OutcomeMagicHitNoHitCounter)
					if result.Landed() {
						spell.Dot(target).Apply(sim)
					}
					spell.DealOutcome(sim, result)
				},

				RelatedAuras: []core.AuraArray{druid.InsectSwarmAuras},
			})
		}
	}
}
