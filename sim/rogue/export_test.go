package rogue

// DeadlyPoisonTickMultiplierForTest exposes the Deadly Poison tick spell's
// damage multiplier to the external test package.
func (rogue *Rogue) DeadlyPoisonTickMultiplierForTest() float64 {
	return rogue.deadlyPoisonTick.DamageMultiplier
}

// ImprovedExposeArmorRefundForTest exposes the Improved Expose Armor refund.
func ImprovedExposeArmorRefundForTest(rank, comboPoints int32) int32 {
	return improvedExposeArmorRefund(rank, comboPoints)
}
