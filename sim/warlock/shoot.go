package warlock

// registerShootSpell wires up the wand "Shoot" action (core.RegisterShootSpell). Classic
// casters can wand-weave while leveling and this engine never modeled it - a warlock with a
// wand equipped dealt zero wand damage until now. Returns without registering anything if
// the warlock has no wand equipped, matching core.RegisterShootSpell's nil-safe contract.
// Warlocks have no Wand Specialization equivalent, so the damage multiplier is always 1.
func (warlock *Warlock) registerShootSpell() {
	warlock.Shoot = warlock.RegisterShootSpell(1)
}
