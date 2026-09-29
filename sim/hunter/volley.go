package hunter

import (
	"fmt"
	"time"

	"github.com/wowsims/classic/sim/core"
)

func (hunter *Hunter) registerVolleySpell() {
	ranks := 3

	for i := ranks; i >= 0; i-- {
		config := hunter.getVolleyConfig(i)

		if config.RequiredLevel <= int(hunter.Level) {
			hunter.Volley = hunter.GetOrRegisterSpell(config)
			break
		}
	}
}

func (hunter *Hunter) getVolleyConfig(rank int) core.SpellConfig {
	spellId := [4]int32{0, 1510, 14294, 14295}[rank]
	baseDamage := [4]float64{0, 50, 65, 80}[rank]
	manaCost := [4]float64{0, 350, 420, 490}[rank]
	level := [4]int{0, 40, 50, 58}[rank]

	manaCostModifer := 100 - 2*hunter.Talents.Efficiency

	return core.SpellConfig{
		SpellCode:   SpellCode_HunterVolley,
		ActionID:    core.ActionID{SpellID: spellId},
		SpellSchool: core.SpellSchoolArcane,
		ProcMask:    core.ProcMaskSpellDamage,
		Flags:       core.SpellFlagChanneled | core.SpellFlagAPL,

		RequiredLevel: level,
		Rank:          rank,

		ManaCost: core.ManaCostOptions{
			FlatCost:   manaCost,
			Multiplier: manaCostModifer,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			// spellconst/hunter.json carries cooldown_ms 0 and
			// category_cooldown_ms 0 for all three ranks (1510/14294/
			// 14295), and Wowhead's Forever tooltip for 1510 states
			// "n/a" for cooldown -- Volley is gated purely by its mana
			// cost and the 6s channel (below), the same as Classic. The
			// engine used to give it a fictitious 60s cooldown.
		},

		Dot: core.DotConfig{
			IsAOE: true,
			Aura: core.Aura{
				Label: fmt.Sprintf("Volley (Rank %d)", rank),
			},
			NumberOfTicks:    6,
			TickLength:       time.Second * 1,
			BonusCoefficient: .056,
			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, isRollover bool) {
				damage := baseDamage
				dot.Snapshot(target, damage, isRollover)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				for _, aoeTarget := range sim.Encounter.TargetUnits {
					dot.CalcAndDealPeriodicSnapshotDamage(sim, aoeTarget, dot.OutcomeTick)
				}
			},
		},

		CritDamageBonus:  (1 + hunter.mortalShots()) * (1 + (0.05 * float64(hunter.Talents.Barrage))),
		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			hunter.Unit.AutoAttacks.DelayRangedUntil(sim, sim.CurrentTime+(time.Second*6))
			spell.AOEDot().Apply(sim)
		},
	}
}
