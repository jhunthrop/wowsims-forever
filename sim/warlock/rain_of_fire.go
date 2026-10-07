package warlock

import (
	"strconv"
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
)

const RainOfFireRanks = 4

// RainOfFireTickDamage is spellconst/warlock.json's own damage per tick for
// the damage spells (ids 1282380, 1282383, 1282384, 1282385) the cast ids
// 5740 through 11678 trigger: rank 4 is 220 at level 58 growing 0.6 a level
// to level 63 (a centre of 223 at level 60), where the table it replaces
// carried 42/92/155/226.
var RainOfFireTickDamage = [RainOfFireRanks + 1]clientdamage.Effect{
	{},
	{Amount: 40, PerLevel: 0.3, SpellLevel: 20, MaxLevel: 25},
	{Amount: 91, PerLevel: 0.4, SpellLevel: 34, MaxLevel: 39},
	{Amount: 149, PerLevel: 0.5, SpellLevel: 46, MaxLevel: 51},
	{Amount: 220, PerLevel: 0.6, SpellLevel: 58, MaxLevel: 63},
}

func (warlock *Warlock) getRainOfFireBaseConfig(rank int) core.SpellConfig {
	spellId := [RainOfFireRanks + 1]int32{0, 5740, 6219, 11677, 11678}[rank]
	spellCoeff := [RainOfFireRanks + 1]float64{0, 0.083, 0.083, 0.083, 0.083}[rank]
	damage := RainOfFireTickDamage[rank]
	casterLevel := int(warlock.Level)
	baseDamage := damage.Center(casterLevel)
	manaCost := [RainOfFireRanks + 1]float64{0, 295, 605, 885, 1185}[rank]
	level := [RainOfFireRanks + 1]int{0, 20, 34, 46, 58}[rank]

	flags := core.SpellFlagAPL | core.SpellFlagResetAttackSwing | WarlockFlagDestruction | core.SpellFlagChanneled

	config := core.SpellConfig{
		ActionID:         core.ActionID{SpellID: spellId},
		SpellSchool:      core.SpellSchoolFire,
		DefenseType:      core.DefenseTypeMagic,
		ProcMask:         core.ProcMaskSpellDamage,
		Flags:            flags,
		RequiredLevel:    level,
		Rank:             rank,
		ClientBaseDamage: damage.Range(casterLevel),

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		ManaCost: core.ManaCostOptions{
			FlatCost: manaCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
		},
		Dot: core.DotConfig{
			IsAOE: true,
			Aura: core.Aura{
				Label: "RainOfFire-" + warlock.Label + strconv.Itoa(rank),
			},
			NumberOfTicks:    4,
			TickLength:       time.Second * 2,
			BonusCoefficient: spellCoeff,

			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
				dot.Snapshot(target, baseDamage, isRollover)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				for _, aoeTarget := range sim.Encounter.TargetUnits {
					dot.CalcAndDealPeriodicSnapshotDamage(sim, aoeTarget, dot.OutcomeTick)
				}

			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.AOEDot().Apply(sim)
		},
	}

	return config
}

func (warlock *Warlock) registerRainOfFireSpell() {
	warlock.RainOfFire = make([]*core.Spell, 0)
	for rank := 1; rank <= RainOfFireRanks; rank++ {
		config := warlock.getRainOfFireBaseConfig(rank)

		if config.RequiredLevel <= int(warlock.Level) {
			warlock.RainOfFire = append(warlock.RainOfFire, warlock.GetOrRegisterSpell(config))
		}
	}
}
