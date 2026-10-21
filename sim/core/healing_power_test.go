package core

import (
	"testing"

	"github.com/wowsims/classic/sim/core/stats"
)

// Forever's items state a spell's damage and healing bonuses as separate
// stats, so a heal reads the healing stat alone: spell power (the damage
// bonus) must not leak into it.
func TestHealingPowerIsTheHealingStatAlone(t *testing.T) {
	caster := &Unit{}
	caster.stats[stats.SpellPower] = 30
	caster.stats[stats.SpellDamage] = 20
	caster.stats[stats.HealingPower] = 90
	target := &Unit{PseudoStats: stats.NewPseudoStats()}
	target.PseudoStats.BonusHealingTaken = 5

	spell := &Spell{Unit: caster}
	if got, want := spell.HealingPower(target), 95.0; got != want {
		t.Errorf("HealingPower = %v, want %v (healing 90 + 5 taken, no spell power)", got, want)
	}
}
