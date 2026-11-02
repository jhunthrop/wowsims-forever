package tankwarrior

import "testing"

// Battle Shout generates no threat in Forever. forever-bugs #64: a warrior's
// threat parser read 0 for Battle Shout, Demoralizing Shout and the paladin
// blessings, and the Blizzard reply (2026-09-22) says "Many Buffs and Debuffs
// now cause 0 threat. These ones are all intended to cause 0 threat." The
// engine still gave the shout Era's flat threat on the warrior and on five
// party hits (round 11, auras lane).
func TestBattleShoutGeneratesNoThreat(t *testing.T) {
	war, sim := buildTank(t, emptyTalents, true)
	war.AddRage(sim, 100, war.RageRefundMetrics)
	spell := war.BattleShout.Spell
	spell.Cast(sim, sim.Encounter.TargetUnits[0])
	for i, metrics := range spell.SpellMetrics {
		if metrics.TotalThreat != 0 {
			t.Errorf("Battle Shout put %v threat on target %d, want 0", metrics.TotalThreat, i)
		}
	}
}
