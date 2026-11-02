package dpswarrior

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/warrior"
)

// outcomes counts every outcome a spell's metrics recorded against target 0.
func outcomes(spell *core.Spell) int32 {
	m := spell.SpellMetrics[0]
	return m.Misses + m.Hits + m.Crits + m.Dodges + m.Parries + m.Glances + m.Blocks + m.BlockedCrits + m.Crushes
}

// Whirlwind's off-hand strike is the client's own off-hand attack, spell
// 462891: SpellMisc Attributes_3 0x01000000 (the off-hand attack flag),
// effect 121 normalized weapon damage on one target, no cost. It therefore
// strikes as the off hand: an off-hand special that rolls the off hand's
// procs and none of the main hand's (Windfury Totem, a main-hand enchant).
// It used to be dealt by Whirlwind itself, a main-hand special.
func TestWhirlwindOffHandStrikeIsAnOffHandSpecial(t *testing.T) {
	talents := talentStringWithRank(t, warrior.ForeverFuryTalents, "dual_wield_specialization", 5)
	built, sim := buildWarriorWithWeaponsAndSim(t, talents, 12784, 12784)
	if !built.AutoAttacks.IsDualWielding {
		t.Fatal("the test warrior is not dual wielding")
	}
	offHand := built.GetSpell(core.ActionID{SpellID: warrior.WhirlwindOffHandSpellID})
	if offHand == nil {
		t.Fatalf("no spell %d registered for Whirlwind's off-hand strike", warrior.WhirlwindOffHandSpellID)
	}
	if offHand.ProcMask != core.ProcMaskMeleeOHSpecial {
		t.Errorf("Whirlwind's off-hand strike proc mask = %v, want the off-hand special %v", offHand.ProcMask, core.ProcMaskMeleeOHSpecial)
	}
	if built.Talents.DualWieldSpecialization != 5 || offHand.DamageMultiplier != 1.25 {
		t.Errorf("Whirlwind's off-hand strike damage multiplier = %v at Dual Wield Specialization %d/5, want 1.25 at 5/5", offHand.DamageMultiplier, built.Talents.DualWieldSpecialization)
	}

	built.AddRage(sim, 100, built.NewRageMetrics(core.ActionID{OtherID: proto.OtherAction_OtherActionAttack}))
	built.Stance = warrior.BerserkerStance
	built.Whirlwind.Cast(sim, sim.Encounter.TargetUnits[0])

	if got := outcomes(built.Whirlwind.Spell); got != 1 {
		t.Errorf("Whirlwind (1680) rolled %d outcomes against one target, want 1 (the main hand only)", got)
	}
	if got := outcomes(offHand); got != 1 {
		t.Errorf("Whirlwind's off-hand strike rolled %d outcomes against one target, want 1", got)
	}
}
