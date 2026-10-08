package druid

// Ferocity (node 104938, five ranks): "Reduces the cost of your Maul,
// Primal Bite, Swipe, Claw, and Rake abilities by 1 Rage or Energy" a rank.
// Savage Fury (node 104948, two ranks): "Increases the damage caused by your
// Claw, Rake, Shred, Maul, and Swipe abilities by 5%" a rank.
const (
	ferocityMaxRank          = 5
	ferocityCostDiscountRank = 1.0
	savageFuryMaxRank        = 2
	savageFuryDamagePerRank  = 0.05
)

// ferocityRageDiscount is the rage Ferocity takes off a bear ability.
func ferocityRageDiscount(rank int32) float64 {
	return ferocityCostDiscountRank * float64(clampRank(rank, ferocityMaxRank))
}

// savageFuryDamageMultiplier is the additive damage multiplier Savage Fury
// puts on its abilities.
func (druid *Druid) savageFuryDamageMultiplier() float64 {
	return 1 + savageFuryDamagePerRank*float64(clampRank(druid.Talents.SavageFury, savageFuryMaxRank))
}
