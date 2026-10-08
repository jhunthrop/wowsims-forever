package rogue

// DeadlyPoisonTickMultiplierForTest exposes the Deadly Poison tick spell's
// damage multiplier to the external test package.
func (rogue *Rogue) DeadlyPoisonTickMultiplierForTest() float64 {
	return rogue.deadlyPoisonTick.DamageMultiplier
}
