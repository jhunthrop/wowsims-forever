package rogue

import "github.com/wowsims/classic/sim/core"

// DeadlyPoisonTickMultiplierForTest exposes the Deadly Poison tick spell's
// damage multiplier to the external test package.
func (rogue *Rogue) DeadlyPoisonTickMultiplierForTest() float64 {
	return rogue.deadlyPoisonTick.DamageMultiplier
}

// ImprovedExposeArmorRefundForTest exposes the Improved Expose Armor refund.
func ImprovedExposeArmorRefundForTest(rank, comboPoints int32) int32 {
	return improvedExposeArmorRefund(rank, comboPoints)
}

// HemorrhageWeaponDamageForTest exposes Hemorrhage's weapon damage roll.
func (rogue *Rogue) HemorrhageWeaponDamageForTest(sim *core.Simulation, attackPower float64) float64 {
	return rogue.hemorrhageWeaponDamage(sim, attackPower)
}

// MutilateOHWeaponDamageForTest exposes Mutilate's off-hand weapon damage roll.
func (rogue *Rogue) MutilateOHWeaponDamageForTest(sim *core.Simulation, attackPower float64) float64 {
	return rogue.mutilateOHWeaponDamage(sim, attackPower)
}

// PoisedEngineMaskForTest exposes the engine spells Poised's client family
// reaches.
func PoisedEngineMaskForTest() uint64 {
	return rogueClassMasks.Resolve(poisedFamilies)
}

// DeadlyPoisonSnapshotMultiplierForTest exposes the caster multiplier the
// Deadly Poison on target last took.
func (rogue *Rogue) DeadlyPoisonSnapshotMultiplierForTest(target *core.Unit) float64 {
	return rogue.deadlyPoisonTick.Dot(target).SnapshotAttackerMultiplier
}

// DeadlyPoisonStacksForTest exposes the Deadly Poison stacks on target.
func (rogue *Rogue) DeadlyPoisonStacksForTest(target *core.Unit) int32 {
	return rogue.deadlyPoisonTick.Dot(target).GetStacks()
}

// BleedTickDamageForTest deals one tick of a running Rupture or Garrote
// through the rogue's bleed path with a plain hit, and returns its damage.
func (rogue *Rogue) BleedTickDamageForTest(sim *core.Simulation, target *core.Unit, bleed *core.Spell) float64 {
	dot := bleed.Dot(target)
	return rogue.dealBleedTickWith(sim, target, dot, dot.Spell.OutcomeAlwaysHit).Damage
}

// HemorrhageDebuffForTest is the Hemorrhage debuff on target.
func (rogue *Rogue) HemorrhageDebuffForTest(target *core.Unit) *core.Aura {
	return rogue.hemorrhageAuras.Get(target)
}
