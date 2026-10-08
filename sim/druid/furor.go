package druid

import "time"

// Furor's Cat Form half, from the live text of node 104958 in client build
// 1.60.1.70009: "When you shift into Cat Form, you will regain 20% of the
// Energy you had when you were last in Cat Form, plus 2 Energy for each
// second you spent not in Bear Form, Cat Form, or Dire Bear Form, up to a
// maximum of 20 Energy." Every figure scales with the rank. The cap is read
// as applying to the whole refill, so rank 5 refills at most to the 100 bar.
const (
	furorPercentOfEnergyPerRank = 20.0
	furorEnergyPerSecondPerRank = 2.0
	furorMaxEnergyPerRank       = 20.0
)

// furorCatFormEnergy is the energy Cat Form starts with at the given Furor
// rank, given the energy the druid had when it last left Cat Form and how
// long it has been out of every shapeshift form since.
func furorCatFormEnergy(rank int32, lastCatEnergy float64, outOfForm time.Duration) float64 {
	if rank <= 0 {
		return 0
	}
	r := float64(rank)
	regained := furorPercentOfEnergyPerRank*r*lastCatEnergy/100 + furorEnergyPerSecondPerRank*r*outOfForm.Seconds()
	return min(regained, furorMaxEnergyPerRank*r)
}

// Furor's Bear Form half, same node: "Gives you a 100% chance to gain 10
// Rage when you shapeshift into Bear Form or Dire Bear Form" at rank 5, 20%
// a rank.
const (
	furorBearFormRageChancePerRank = 0.2
	furorBearFormRage              = 10.0
)

// furorBearFormRageChance is the chance a shift into Bear Form pays
// furorBearFormRage at the given Furor rank.
func furorBearFormRageChance(rank int32) float64 {
	return min(max(furorBearFormRageChancePerRank*float64(rank), 0), 1)
}
