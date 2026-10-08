package warlock

// WarlockSpellMask* tag the spells a spell mod targets (the PvP sets'
// Immolate bonus). A new modifier adds its spell here.
const (
	WarlockSpellMaskImmolate uint64 = 1 << iota
)
