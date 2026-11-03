package core

import (
	"slices"

	"github.com/wowsims/classic/sim/core/proto"
)

// Orc's Command, re-added in Blizzard's 26 October notes ("increasing
// Warlock and Hunter pet damage by 5%") in place of Savage's 1% crit, as
// the 1.60.1.70753 client states it:
//   - 20576 (hunter, ClassMask 4) and 20575 (warlock, ClassMask 256) add 5
//     to every effect (aura 107, misc 8) of the pet's "Tamed Pet Passive
//     (DND)": 8875 on every hunter pet family, 18727-18730 on the
//     Voidwalker, Imp, Succubus and Felhunter. That passive is aura 79,
//     damage done, all schools, at 0, so the pet deals 5% more damage. An
//     item guardian (Emerald Dragon Whelp, the Core Hound) has no such
//     passive and gains nothing.
//   - 20575's second effect, new in 70753, is aura 108 (percent modifier,
//     damage) at 5 on warlock class mask 0x3000: the Imp's Firebolt
//     (0x1000) and the Succubus' Lash of Pain (0x2000). The owner's spell
//     modifiers reach its pet's spells, so those two take 5% more again.
//   - Every other class's Command is the dummy 21563 (ClassMask 1243).
const (
	commandPetDamageMultiplier   = 1.05
	commandDemonSpellsMultiplier = 1.05
)

// commandDemonSpellIDs are the warlock spells under class mask 0x3000
// (SpellClassOptions set 5): every Firebolt rank (0x1000) and every Lash of
// Pain rank (0x2000).
var commandDemonSpellIDs = []int32{
	3110, 7799, 7800, 7801, 7802, 11762, 11763, 20801, // Firebolt
	7814, 7815, 7816, 11778, 11779, 11780, // Lash of Pain
}

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
		pet.PseudoStats.DamageDealtMultiplier *= commandPetDamageMultiplier
		if character.Class == proto.Class_ClassWarlock {
			pet.OnSpellRegistered(func(spell *Spell) {
				if slices.Contains(commandDemonSpellIDs, spell.ActionID.SpellID) {
					spell.DamageMultiplier *= commandDemonSpellsMultiplier
				}
			})
		}
	}
}
