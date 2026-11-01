package balance

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
	"github.com/wowsims/classic/sim/druid"
)

// Forever's DoTs read spell power and the caster's multipliers at every
// tick (forever-bugs #137, Blizzard 2026-10-01: "Since we've changed to
// dynamic DoTs"; #398, 2026-10-26).
func TestBalanceDotsReadSpellPowerEachTick(t *testing.T) {
	for _, tc := range []struct {
		name  string
		spell func(*BalanceDruid) *druid.DruidSpell
	}{
		{"Moonfire", func(d *BalanceDruid) *druid.DruidSpell { return d.Moonfire[len(d.Moonfire)-1] }},
		{"Insect Swarm", func(d *BalanceDruid) *druid.DruidSpell { return d.InsectSwarm[len(d.InsectSwarm)-1] }},
		{"Hurricane", func(d *BalanceDruid) *druid.DruidSpell { return d.Hurricane[len(d.Hurricane)-1] }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Insect Swarm and Hurricane are talents; a full Balance build.
			built, sim, target := newBalanceDruidSimWithTalents(t, "5232220115501351-00000000000000000000-5050030000000000")
			spell := tc.spell(built)
			if spell == nil {
				t.Fatal("spell not registered")
			}
			var dot *core.Dot
			for attempt := 0; attempt < 20; attempt++ {
				spell.ApplyEffects(sim, target, spell.Spell)
				d := spell.AOEDot()
				if !spell.Flags.Matches(core.SpellFlagChanneled) {
					d = spell.Dot(target)
				}
				if d.IsActive() {
					dot = d
					break
				}
			}
			if dot == nil {
				t.Fatal("dot never landed")
			}
			dot.TickCount = 1
			dot.TickOnce(sim)
			before := dot.SnapshotBaseDamage
			crit := dot.SnapshotCritChance
			built.AddStatDynamic(sim, stats.SpellPower, 100)
			dot.TickCount = 2
			dot.TickOnce(sim)
			want := before + 100*dot.BonusCoefficient
			if math.Abs(dot.SnapshotBaseDamage-want) > 1e-9 {
				t.Errorf("tick 2 base %v after +100 spell power, want %v", dot.SnapshotBaseDamage, want)
			}
			if dot.SnapshotCritChance != crit {
				t.Errorf("tick 2 crit chance %v, want tick 1's %v", dot.SnapshotCritChance, crit)
			}
		})
	}
}
