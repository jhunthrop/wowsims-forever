package core

import (
	"github.com/wowsims/classic/sim/core/proto"
)

// Orc's Command, re-added in Blizzard's 26 October notes ("increasing
// Warlock and Hunter pet damage by 5%") in place of Savage's 1% crit, as
// the live client states it: 20576 (hunter, ClassMask 4) and 20575
// (warlock, ClassMask 256) add 5 to every effect (aura 107, misc 8) of the
// pet's "Tamed Pet Passive (DND)": 8875 on every hunter pet family,
// 18727-18730 on the Voidwalker, Imp, Succubus and Felhunter. That passive
// is aura 79, damage done, all schools, at 0, so the pet deals 5% more
// damage. An item guardian (Emerald Dragon Whelp, the Core Hound) has no
// such passive and gains nothing. The +5 adds to the other points on the
// same passive (Unholy Power, Unleashed Fury): 1 + 0.10 + 0.05 with Unholy
// Power 5, not 1.10 x 1.05.
//
// 20575 gained a second effect in 70753, a 5% damage modifier on the Imp's
// Firebolt and the Succubus' Lash of Pain (class mask 0x3000), so those two
// took Command twice. The 6 Nov hotfix ("Fixes an issue where some Warlock
// pet abilities were benefitting too much from the Orc Command racial
// ability") sets it to 0: client 1.60.1.70888 and push 113702 (SpellEffect
// 697943).
const commandPetPassivePercent = 5

func applyCommand(character *Character) {
	if character.Class != proto.Class_ClassHunter && character.Class != proto.Class_ClassWarlock {
		return
	}
	// Pets are added while the class agent is built, before
	// applyAllEffects runs the racials, so every pet is here.
	for _, pet := range character.Pets {
		if pet.IsGuardian() {
			continue
		}
		pet.AddPassiveDamagePercent(commandPetPassivePercent)
	}
}
