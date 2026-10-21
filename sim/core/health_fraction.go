package core

// SetHealthFraction puts current health at the given share of maximum
// health without recording a gain or a loss. A shapeshift that raises or
// lowers maximum health keeps the share the character had, and that is
// neither healing nor damage: it must not reach the healing, damage taken
// or TMI metrics.
func (hb *healthBar) SetHealthFraction(fraction float64) {
	hb.currentHealth = min(max(fraction, 0), 1) * hb.MaxHealth()
}
