package shaman

import "github.com/wowsims/classic/sim/core"

// StormstrikeWeaponDamageForTest exposes Stormstrike's weapon damage roll.
func (shaman *Shaman) StormstrikeWeaponDamageForTest(sim *core.Simulation, attackPower float64) float64 {
	return shaman.stormstrikeWeaponDamage(sim, attackPower)
}

// MaelstromWeaponChanceForTest exposes the chance a hit by spell grants a
// Maelstrom Weapon stack.
func (shaman *Shaman) MaelstromWeaponChanceForTest(spell *core.Spell) float64 {
	return shaman.maelstromWeaponChance(spell)
}
