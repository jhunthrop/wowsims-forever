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

// Balance (and Feral) register both the Balance and the Cat kits, and both
// kits carry Moonfire and Wrath. Registered twice, the first Moonfire's dot
// shared the second's aura and ticked beside it with no snapshot: 0 damage
// while dots snapshotted, its spell power share once they read each tick.
func TestBalanceRegistersMoonfireAndWrathOnce(t *testing.T) {
	built, _, _ := newBalanceDruidSimWithTalents(t, "5232220115501351-00000000000000000000-5050030000000000")
	count := map[int32]int{}
	for _, spell := range built.Spellbook {
		count[spell.ActionID.SpellID]++
	}
	for _, id := range []int32{9835, 9912} {
		if count[id] != 1 {
			t.Errorf("spell %d registered %d times, want 1", id, count[id])
		}
	}
}

// Insect Swarm is a non-channelled DoT, so its ticks crit in any form
// (owner ruling 2026-10-31, core.periodicDamageCanCrit). The engine zeroed
// the crit chance outside Moonkin Form, a Season of Discovery leftover:
// Forever's Moonkin Form (24858, 24907) grants armor, Omen of Clarity and
// a 3% crit aura, and no periodic crit. A Balance druid below 40 has no
// Moonkin Form yet.
func TestInsectSwarmTicksCritOutsideMoonkinForm(t *testing.T) {
	built, sim, target := newBalanceDruidSimWithTalents(t, druidTalentsString(t, map[int]int{insectSwarmField: 1}))
	if built.InForm(druid.Moonkin) {
		t.Fatal("the druid is in Moonkin Form without the talent")
	}
	spell := built.InsectSwarm[len(built.InsectSwarm)-1]
	var dot *core.Dot
	for attempt := 0; attempt < 20 && dot == nil; attempt++ {
		spell.ApplyEffects(sim, target, spell.Spell)
		if d := spell.Dot(target); d.IsActive() {
			dot = d
		}
	}
	if dot == nil {
		t.Fatal("Insect Swarm never landed")
	}
	dot.TickCount = 1
	dot.TickOnce(sim)
	if dot.SnapshotCritChance <= 0 {
		t.Errorf("Insect Swarm ticked with crit chance %v outside Moonkin Form, want the caster's spell crit", dot.SnapshotCritChance)
	}
}
