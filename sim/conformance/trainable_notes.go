package conformance

import "github.com/wowsims/classic/sim/core"

// modeledBuffRanks names, per class, the core.BuffRanks tables that model a
// trainable buff the presets assume as a raid buff instead of registering a
// castable spell: the buff is applied from the character's level through
// the table, with the client's amount for each rank. A trainable whose
// rank ids all sit in these tables is reported as modelled, not as missing.
var modeledBuffRanks = map[string][]core.BuffRanks{
	"druid":  {core.MarkOfTheWildArmorRanks},
	"hunter": {core.TrueshotAuraRanks},
}

// modeledBuffSpellIDs lists trainable ids that share a modeled buff's
// effect without being in its rank table: Gift of the Wild (21849, 21850)
// is the group cast of Mark of the Wild and the raid buff's GiftOfTheWild
// field applies the same core.MarkOfTheWildStats.
var modeledBuffSpellIDs = map[string][]int32{
	"druid": {21849, 21850},
}

// modeledBuffIDs is every spell id the class's buff tables and extra ids
// model.
func modeledBuffIDs(classSlug string) map[int32]bool {
	ids := map[int32]bool{}
	for _, table := range modeledBuffRanks[classSlug] {
		for _, rank := range table {
			ids[rank.SpellID] = true
		}
	}
	for _, id := range modeledBuffSpellIDs[classSlug] {
		ids[id] = true
	}
	return ids
}

// racialNotes marks the trainables the client's SkillLineAbility rows
// restrict to one race (RaceMasks: 1 Human, 4 Dwarf, 8 Night Elf, 16
// Undead, 64 Gnome, 128 Troll). The pipeline's trainables file carries no
// race, and the presets each use one race, so a racial the preset's race
// cannot learn reads as "missing" in the table unless it says so here.
var racialNotes = map[string]map[string]string{
	"priest": {
		"Chastise":          "Dwarf racial",
		"Confounding Flash": "Gnome racial",
		"Contingency Plan":  "Gnome racial",
		"Divine Grace":      "Human racial",
		"Elune's Grace":     "Night Elf racial",
		"Feedback":          "Human racial",
		"Hex of Weakness":   "Troll racial",
		"Shadowguard":       "Troll racial",
		"Starshards":        "Night Elf racial; registered by a Night Elf priest (starshards.go), absent from the Undead and Human presets",
		"Touch of Weakness": "Undead racial",
	},
}

func racialNote(classSlug, name string) string {
	return racialNotes[classSlug][name]
}
