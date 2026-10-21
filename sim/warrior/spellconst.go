package warrior

// The ability files in this package take their published numbers from
// constants_auto_gen.go, which sim/core/spellconst/gen emits from the
// client's own spell table. The arrays there are indexed by the rank
// LABEL, so a rank is a number this package chooses and then uses to
// read every column of that rank at once, the way sim/mage/frostbolt.go
// does.
//
// Two helpers do the choosing, so that no ability file repeats the loop.

// rankAtLevel is the highest rank in a generated <Spell>Level array the
// character has learned. Rank 0 is included because the client gives
// plenty of spells a single, unnumbered rank that the generator emits at
// label 0; a rank the character is too low for is simply never the
// highest one that qualifies.
func rankAtLevel(levels []int, level int32) int {
	best := 0
	for rank, required := range levels {
		if required <= int(level) {
			best = rank
		}
	}
	return best
}

// rageCost converts a generated <Spell>ManaCost entry into rage. The
// client stores a rage cost in tenths of a point in the same column it
// stores mana, so 300 there is 30 rage here.
func rageCost(cost float64) float64 {
	return cost / 10
}
