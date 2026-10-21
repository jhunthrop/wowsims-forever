package core

// HighestRankAtLevel is the rank a character of the given level has learned
// when rank r (1-based) is learned at learnLevels[r-1]: the last rank whose
// learn level is at or below level, or 0 when none is. It replaces the
// Season of Discovery bracket maps (25/40/50/60) the class code inherited,
// which answered nothing for the levels between them.
func HighestRankAtLevel(learnLevels []int, level int32) int {
	rank := 0
	for i, learnLevel := range learnLevels {
		if int32(learnLevel) <= level {
			rank = i + 1
		}
	}
	return rank
}
