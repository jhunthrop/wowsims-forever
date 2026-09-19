package dpswarrior

import (
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/warrior"
)

// Every ability a talent modifies must carry its ClassSpellMask, or the
// declarative mod in talents.go silently applies to nothing.
//
// This is asserted against the spells a built warrior actually
// registers. The version of this test that lived in sim/warrior only
// read the `1 << iota` constants back and checked they were non-zero and
// distinct — true by construction of the iota block, and green with
// every `ClassSpellMask:` line deleted from every ability file. Tasks 11
// and 12's whole declarative-talent design rests on these masks, so the
// invariant is worth a real character.
func TestFurySpellsCarryTheirMasksWhenRegistered(t *testing.T) {
	// Three builds, because two of the masked abilities are 31-point
	// talents that no single build reaches: Mortal Strike is the Arms
	// capstone and Shield Slam the Protection one. The weapons are not
	// the subject here — every ability registers regardless.
	oneHander := weaponWithHandType(t, proto.HandType_HandTypeOneHand)
	builds := []string{
		warrior.ForeverFuryTalents,
		warrior.ForeverProtectionTalents,
		talentStringWithRank(t, emptyWarriorTalents, "mortal_strike", 1),
	}

	// mask bit -> the registered spells carrying it, across all builds.
	carriers := map[uint64][]string{}
	var war *warrior.Warrior
	for _, talents := range builds {
		war = buildWarriorWithWeapons(t, talents, oneHander, 0)
		for _, spell := range war.GetCharacter().Spellbook {
			if spell.ClassSpellMask == 0 {
				continue
			}
			for bit := uint64(1); bit != 0; bit <<= 1 {
				if spell.ClassSpellMask&bit != 0 {
					carriers[bit] = append(carriers[bit], spell.ActionID.String())
				}
			}
		}
	}

	named := map[string]uint64{
		"Bloodthirst":   warrior.WarriorSpellMaskBloodthirst,
		"Whirlwind":     warrior.WarriorSpellMaskWhirlwind,
		"Execute":       warrior.WarriorSpellMaskExecute,
		"Heroic Strike": warrior.WarriorSpellMaskHeroicStrike,
		"Cleave":        warrior.WarriorSpellMaskCleave,
		"Mortal Strike": warrior.WarriorSpellMaskMortalStrike,
		"Overpower":     warrior.WarriorSpellMaskOverpower,
		"Rend":          warrior.WarriorSpellMaskRend,
		"Revenge":       warrior.WarriorSpellMaskRevenge,
		"Shield Slam":   warrior.WarriorSpellMaskShieldSlam,
		"Slam":          warrior.WarriorSpellMaskSlam,
		"Sunder Armor":  warrior.WarriorSpellMaskSunderArmor,
		"Thunder Clap":  warrior.WarriorSpellMaskThunderClap,
		"Hamstring":     warrior.WarriorSpellMaskHamstring,
		"Pummel":        warrior.WarriorSpellMaskPummel,
		"Piercing Howl": warrior.WarriorSpellMaskPiercingHowl,
	}
	for name, mask := range named {
		if len(carriers[mask]) == 0 {
			t.Errorf("no registered warrior spell carries %s's mask; every talent mod that names it applies to nothing", name)
		}
	}

	// The two groups the talents target must be covered too: a group bit
	// no registered spell carries is a mod that binds to nothing.
	for name, group := range map[string]uint64{
		"WarriorSpellMaskSpecials":    warrior.WarriorSpellMaskSpecials,
		"WarriorSpellMaskOnNextSwing": warrior.WarriorSpellMaskOnNextSwing,
	} {
		for bit := uint64(1); bit != 0; bit <<= 1 {
			if group&bit != 0 && len(carriers[bit]) == 0 {
				t.Errorf("%s names bit %#x, which no registered spell carries", name, bit)
			}
		}
	}

	// The masks must stay one-bit-per-ability: two abilities sharing a
	// bit means a talent that names one silently modifies both.
	for name, mask := range named {
		for other, otherMask := range named {
			if name < other && mask == otherMask {
				t.Errorf("%s and %s share mask %#x", name, other, mask)
			}
		}
	}

	// Auto-attacks deliberately carry no mask — see the Two-Handed
	// Weapon Specialization note in sim/warrior/talents.go.
	if mh := war.AutoAttacks.MHAuto(); mh != nil && mh.ClassSpellMask != 0 {
		t.Errorf("the main-hand auto-attack carries ClassSpellMask %#x; the talents assume autos are unmasked", mh.ClassSpellMask)
	}
}
