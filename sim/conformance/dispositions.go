package conformance

// Disposition kinds for a trainable ability the engine registers no spell
// for. The text after the kind says why, in one line.
const (
	dispUtility = "utility"
	dispAura    = "modeled as aura"
	dispBuff    = "modeled as core buff"
	dispDefer   = "not registered, no preset reads it"
)

// trainableDispositions classifies, per client class slug, every ability in
// the golden's unregistered tables. A class appears here once its lane has
// classified its list; its golden then gains a Disposition column and a test
// (TestClassifiedTrainablesAreAllDispositioned) fails on any ability without
// an entry. The rule for "affects a simmed result": it would change a
// preset's damage, mana, pet damage, or a buff or debuff the preset assumes.
var trainableDispositions = map[string]map[string]string{
	"mage": {
		"Arcane Intellect":        dispBuff + ": core.ArcaneIntellectRanks (client rank table); casting it changes no result",
		"Arcane Brilliance":       dispBuff + ": the same Intellect value as Arcane Intellect rank 5, applied as a raid buff",
		"Frost Armor":             dispAura + ": sim/mage/armors.go, ranks 168/7300/7301 by level",
		"Ice Armor":               dispAura + ": sim/mage/armors.go, ranks 7302/7320/10219/10220 by level",
		"Mage Armor":              dispAura + ": sim/mage/armors.go, 50% regeneration while casting from the client's aura 134",
		"Conjure Water":           dispUtility + ": creates a drink",
		"Conjure Food":            dispUtility + ": creates food",
		"Conjure Mana Agate":      dispUtility + ": creates a mana gem; the gem itself is simmed (mana_gem.go), conjuring it is not",
		"Conjure Mana Jade":       dispUtility + ": creates a mana gem; the gem itself is simmed (mana_gem.go), conjuring it is not",
		"Conjure Mana Citrine":    dispUtility + ": creates a mana gem; the gem itself is simmed (mana_gem.go), conjuring it is not",
		"Conjure Mana Ruby":       dispUtility + ": creates a mana gem; the gem itself is simmed (mana_gem.go), conjuring it is not",
		"Ice Block":               dispUtility + ": invulnerability, no incoming damage in a sim",
		"Polymorph":               dispUtility + ": crowd control",
		"Polymorph: Cow":          dispUtility + ": crowd control",
		"Dampen Magic":            dispUtility + ": reduces magic damage and healing taken by the target",
		"Amplify Magic":           dispUtility + ": increases magic damage and healing taken by the target",
		"Slow Fall":               dispUtility + ": movement",
		"Remove Lesser Curse":     dispUtility + ": dispel",
		"Blink":                   dispUtility + ": movement",
		"Fire Ward":               dispUtility + ": absorbs incoming fire damage",
		"Frost Ward":              dispUtility + ": absorbs incoming frost damage",
		"Mana Shield":             dispUtility + ": absorbs incoming damage, none in a sim",
		"Teleport: Ironforge":     dispUtility + ": travel",
		"Teleport: Orgrimmar":     dispUtility + ": travel",
		"Teleport: Stormwind":     dispUtility + ": travel",
		"Teleport: Undercity":     dispUtility + ": travel",
		"Teleport: Darnassus":     dispUtility + ": travel",
		"Teleport: Thunder Bluff": dispUtility + ": travel",
		"Teleport: Dalaran":       dispUtility + ": travel",
		"Portal: Ironforge":       dispUtility + ": travel",
		"Portal: Orgrimmar":       dispUtility + ": travel",
		"Portal: Stormwind":       dispUtility + ": travel",
		"Portal: Undercity":       dispUtility + ": travel",
		"Portal: Darnassus":       dispUtility + ": travel",
		"Portal: Thunder Bluff":   dispUtility + ": travel",
		"Felfire":                 dispDefer + ": 500 Fire damage at coefficient 1.0 for 100 mana on a 1.5s GCD with no cooldown, on a Fire skill-line row with class mask 0 (not a mage-only row); looks like a shared test or item spell, so it is held for the owner to confirm before it becomes a rotation spell",
		"Earth Volley":            dispUtility + ": NPC spell, not a player spellbook entry",
		"Fire Volley":             dispUtility + ": NPC spell, not a player spellbook entry",
		"Frost Volley":            dispUtility + ": NPC spell, not a player spellbook entry",
		"Lightning Volley":        dispUtility + ": NPC spell, not a player spellbook entry",
		"Sleep":                   dispUtility + ": crowd control, no learn row",
		"Khadgar's Unlocking":     dispUtility + ": opens locks, no learn row",
	},
	"warlock": {
		"Curse of Weakness":   dispUtility + ": lowers the target's attack power; no sim number reads it",
		"Curse of Tongues":    dispUtility + ": slows the target's casting; no sim number reads it",
		"Fear":                dispUtility + ": crowd control",
		"Banish":              dispUtility + ": crowd control",
		"Howl of Terror":      dispUtility + ": crowd control",
		"Subjugate Demon":     dispUtility + ": crowd control",
		"Create Healthstone":  dispUtility + ": creates a consumable",
		"Create Soulstone":    dispUtility + ": creates a consumable",
		"Create Firestone":    dispUtility + ": creates a weapon enchant item",
		"Create Spellstone":   dispUtility + ": creates a weapon enchant item",
		"Demon Skin":          dispUtility + ": leveling armor and health regeneration, replaced by Demon Armor at 20; no sim number reads either",
		"Demon Armor":         dispAura + ": sim/warlock/armors.go, ranks 706/1086/11733/11734/11735 by level",
		"Health Funnel":       dispUtility + ": heals the pet from the warlock's health; pet health is not simmed",
		"Unending Breath":     dispUtility + ": underwater breathing",
		"Ritual of Summoning": dispUtility + ": travel",
		"Eye of Kilrogg":      dispUtility + ": scouting",
		"Detect Invisibility": dispUtility + ": detection",
		"Drain Mana":          dispUtility + ": drains target mana, none matters for a boss",
		"Shadow Ward":         dispUtility + ": absorbs incoming shadow damage",
		"Portal of Summoning": dispUtility + ": travel",
		"Ritual of Doom":      dispUtility + ": summons a Doomguard for a ritual, not a fight cooldown",
		"Summon Incubus":      dispDefer + ": a Forever pet with no pet data in the client tables this engine reads; the presets use Imp, Succubus, Felhunter and Voidwalker",
		"Inferno":             dispDefer + ": summons an Infernal with a one hour cooldown; no pet stat or damage rows exist in the client tables, so it cannot be modeled from them",
		"Hellfire":            dispDefer + ": AoE channel that also damages the caster; the single-target presets never cast it, so no preset result moves",
		"Haunt":               dispDefer + ": the client lists it on no learn row (no Forever trainer teaches it); not a spell a player learns",
		"Unstable Affliction": dispDefer + ": the client lists it on no learn row (no Forever trainer teaches it); not a spell a player learns",
		"Dispel Magic":        dispUtility + ": dispel (the Felhunter pet's ability), no learn row",
	},
}
