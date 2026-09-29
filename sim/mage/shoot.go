package mage

// registerShootSpell wires up the wand "Shoot" action (core.RegisterShootSpell). Classic
// casters can wand-weave while leveling and this engine never modeled it - a mage with a
// wand equipped dealt zero wand damage until now. Returns without registering anything if
// the mage has no wand equipped, matching core.RegisterShootSpell's nil-safe contract.
func (mage *Mage) registerShootSpell() {
	mage.Shoot = mage.RegisterShootSpell(mage.wandSpecializationMultiplier())
}

// wandSpecializationMultiplier returns the wand damage bonus from the Wand Specialization
// talent: 13% at rank 1, 25% at rank 2, 1x (no bonus) if untalented.
func (mage *Mage) wandSpecializationMultiplier() float64 {
	return 1 + [3]float64{0, 0.13, 0.25}[mage.Talents.WandSpecialization]
}
