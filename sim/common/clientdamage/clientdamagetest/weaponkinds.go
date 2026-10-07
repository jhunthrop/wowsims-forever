package clientdamagetest

var (
	// Dummy is a dummy effect (effect 3), where the client parks a flat
	// the spell's script reads: Execute's base, Bloodthirst's attack
	// power percentage.
	Dummy = Kind{effect: 3}
	// WeaponDamageNoSchool is the flat bonus of a weapon-damage effect
	// (effect 17), the part that is not the weapon: Heroic Strike, Cleave
	// and Slam.
	WeaponDamageNoSchool = Kind{effect: 17}
	// WeaponDamage is the flat bonus of a weapon-damage effect with a
	// school (effect 58): Raptor Strike.
	WeaponDamage = Kind{effect: 58}
)
