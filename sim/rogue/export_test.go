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
