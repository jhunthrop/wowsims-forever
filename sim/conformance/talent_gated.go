package conformance

import (
	"fmt"
	"strings"

	"github.com/wowsims/classic/sim/core/spellconst"
	"github.com/wowsims/classic/sim/druid"
	"github.com/wowsims/classic/sim/hunter"
	"github.com/wowsims/classic/sim/mage"
	"github.com/wowsims/classic/sim/paladin"
	"github.com/wowsims/classic/sim/priest"
	"github.com/wowsims/classic/sim/rogue"
	"github.com/wowsims/classic/sim/shaman"
	"github.com/wowsims/classic/sim/warlock"
	"github.com/wowsims/classic/sim/warrior"
)

// TalentGatedSpell is one ability that does not exist in its class's
// spellbook AT ALL without its one granting talent (Seal of Command,
// Ice Lance, Lava Burst, Stormstrike, Mutilate, and the rest below) -
// as opposed to a spell that always exists and a talent merely modifies
// (Improved Frostbolt on Frostbolt, Reverberation on Earth Shock), which
// buildForComparison's empty-talent build already reports correctly.
//
// This list was built by comparing, for every class, the Spellbook of a
// representative preset built with every talent at rank 1 against the
// same preset built with none: every non-passive, real-SpellID spell
// that appeared only in the former is one entry here (minus a handful
// of false positives that are the SAME ability under a talent-swapped
// client id rather than a new one - Mage's Improved Blizzard relabels
// Blizzard's own ActionID per rank rather than gating a second spell,
// and Priest's Shadow Weaving registers an internal hit-roll helper,
// not a player-pressed button - see this package's SUMMARY.md).
//
// Tree and Pos are 1-based: Pos is this talent's ordinal position
// within Tree (0, 1 or 2), in the client's own tier-then-column reading
// order, exactly as each class's own talents_auto_gen.go declares its
// proto fields - the same order core.FillTalentsProto reads a talent
// string's digits in. talentsString therefore never needs to know a
// tree's total size: a talent string's trailing zeros are implicit
// (FillTalentsProto's own doc comment), so the string for "one point in
// position Pos of Tree" is just Pos-1 zeros and a 1, with the other two
// trees empty.
type TalentGatedSpell struct {
	ClassSlug string
	Label     string
	Tree      int
	Pos       int
	// Preset is the label of the Preset to build with, for a spell only one
	// spec of the class registers (Swiftmend belongs to the restoration
	// druid's kit, not the balance druid's). Empty means repPresetForClass.
	Preset string
}

func (g TalentGatedSpell) talentsString() string {
	parts := make([]string, 3)
	parts[g.Tree] = strings.Repeat("0", g.Pos-1) + "1"
	return strings.Join(parts, "-")
}

// TalentGatedSpells is every talent-only ability this package found,
// addressed by 1-based position in the trees generated from build
// 1.60.1.70009 with the live Wowhead overlay (2026-10-07: the Fury,
// Protection, Feral, Marksmanship and Holy layouts moved, so Sniper Shot,
// Divine Favor, Holy Shock, Death Wish, Bloodthirst, Last Stand and
// Berserk changed position; an out-of-range position is skipped silently,
// so a layout change shows up here as a vanished golden row).
// one representative preset's class worth at a time (see
// repPresetForClass). Grouped and ordered by class slug, then by tree,
// then by position, so a diff against a future regeneration reads in
// the same tree order as the client's own talent calculator.
var TalentGatedSpells = []TalentGatedSpell{
	// Hunter (Beast Mastery 16, Marksmanship 17, Survival 18).
	{ClassSlug: "hunter", Label: "Summon Hawk", Tree: 0, Pos: 11},
	{ClassSlug: "hunter", Label: "Bestial Wrath", Tree: 0, Pos: 16},
	{ClassSlug: "hunter", Label: "Sniper Shot", Tree: 1, Pos: 16},
	{ClassSlug: "hunter", Label: "Counterattack", Tree: 2, Pos: 12},
	{ClassSlug: "hunter", Label: "Strider Kick", Tree: 2, Pos: 16},

	// Mage (Arcane 18, Fire 17, Frost 19).
	{ClassSlug: "mage", Label: "Arcane Blast", Tree: 0, Pos: 10},
	{ClassSlug: "mage", Label: "Presence of Mind", Tree: 0, Pos: 15},
	{ClassSlug: "mage", Label: "Arcane Power", Tree: 0, Pos: 18},
	{ClassSlug: "mage", Label: "Pyroblast", Tree: 1, Pos: 9},
	{ClassSlug: "mage", Label: "Blast Wave", Tree: 1, Pos: 15},
	{ClassSlug: "mage", Label: "Combustion", Tree: 1, Pos: 17},
	{ClassSlug: "mage", Label: "Ice Lance", Tree: 2, Pos: 10},
	{ClassSlug: "mage", Label: "Cold Snap", Tree: 2, Pos: 16},
	{ClassSlug: "mage", Label: "Ice Barrier", Tree: 2, Pos: 19},

	// Warlock (Affliction 17, Demonology 19, Destruction 16).
	{ClassSlug: "warlock", Label: "Amplify Curse", Tree: 0, Pos: 9},
	{ClassSlug: "warlock", Label: "Siphon Life", Tree: 0, Pos: 14},
	{ClassSlug: "warlock", Label: "Wrack", Tree: 0, Pos: 17},
	{ClassSlug: "warlock", Label: "Demonic Sacrifice", Tree: 1, Pos: 10},
	{ClassSlug: "warlock", Label: "Fel Domination", Tree: 1, Pos: 13},
	{ClassSlug: "warlock", Label: "Soul Link", Tree: 1, Pos: 16},
	{ClassSlug: "warlock", Label: "Shadowburn", Tree: 2, Pos: 8},
	{ClassSlug: "warlock", Label: "Conflagrate", Tree: 2, Pos: 11},
	{ClassSlug: "warlock", Label: "Bane of Havoc", Tree: 2, Pos: 13},
	{ClassSlug: "warlock", Label: "Incinerate", Tree: 2, Pos: 16},

	// Paladin (Holy 18, Protection 16, Retribution 18).
	{ClassSlug: "paladin", Label: "Divine Favor", Tree: 0, Pos: 12},
	{ClassSlug: "paladin", Label: "Holy Shock", Tree: 0, Pos: 14},
	{ClassSlug: "paladin", Label: "Holy Shield", Tree: 1, Pos: 16},
	{ClassSlug: "paladin", Label: "Seal of Command", Tree: 2, Pos: 8},

	// Warrior (Arms 17, Fury 18, Protection 18).
	{ClassSlug: "warrior", Label: "Spearing Strike", Tree: 0, Pos: 9},
	{ClassSlug: "warrior", Label: "Sweeping Strikes", Tree: 0, Pos: 13},
	{ClassSlug: "warrior", Label: "Mortal Strike", Tree: 0, Pos: 17},
	{ClassSlug: "warrior", Label: "Piercing Howl", Tree: 1, Pos: 6},
	{ClassSlug: "warrior", Label: "Death Wish", Tree: 1, Pos: 13},
	{ClassSlug: "warrior", Label: "Bloodthirst", Tree: 1, Pos: 17},
	{ClassSlug: "warrior", Label: "Last Stand", Tree: 2, Pos: 7},
	{ClassSlug: "warrior", Label: "Shield Slam", Tree: 2, Pos: 18},

	// Druid (Balance 16, Feral Combat 19, Restoration 16).
	{ClassSlug: "druid", Label: "Insect Swarm", Tree: 0, Pos: 9},
	{ClassSlug: "druid", Label: "Moonkin Form", Tree: 0, Pos: 16},
	{ClassSlug: "druid", Label: "Berserk", Tree: 1, Pos: 20},
	{ClassSlug: "druid", Label: "Swiftmend", Tree: 2, Pos: 11, Preset: "RestorationDruid"},
	{ClassSlug: "druid", Label: "Nature's Swiftness", Tree: 2, Pos: 12},
	{ClassSlug: "druid", Label: "Wild Growth", Tree: 2, Pos: 16, Preset: "RestorationDruid"},

	// Priest (Discipline 18, Holy 17, Shadow 18).
	{ClassSlug: "priest", Label: "Inner Focus", Tree: 0, Pos: 9},
	{ClassSlug: "priest", Label: "Penance", Tree: 0, Pos: 15, Preset: "HealingPriest"},
	{ClassSlug: "priest", Label: "Power Infusion", Tree: 0, Pos: 18, Preset: "HealingPriest"},
	{ClassSlug: "priest", Label: "Holy Nova", Tree: 1, Pos: 6, Preset: "HealingPriest"},
	{ClassSlug: "priest", Label: "Binding Heal", Tree: 1, Pos: 12, Preset: "HealingPriest"},
	{ClassSlug: "priest", Label: "Prayer of Mending", Tree: 1, Pos: 17, Preset: "HealingPriest"},
	{ClassSlug: "priest", Label: "Mind Flay", Tree: 2, Pos: 9},
	{ClassSlug: "priest", Label: "Vampiric Embrace", Tree: 2, Pos: 12},
	{ClassSlug: "priest", Label: "Shadowform", Tree: 2, Pos: 18},

	// Shaman (Elemental 16, Enhancement 18, Restoration 16).
	{ClassSlug: "shaman", Label: "Lava Burst", Tree: 0, Pos: 16},
	{ClassSlug: "shaman", Label: "Stormstrike", Tree: 1, Pos: 13},
	{ClassSlug: "shaman", Label: "Rage of the Farseer", Tree: 1, Pos: 18},
	{ClassSlug: "shaman", Label: "Water Shield", Tree: 2, Pos: 9},
	{ClassSlug: "shaman", Label: "Mana Tide Totem", Tree: 2, Pos: 12, Preset: "RestorationShaman"},
	{ClassSlug: "shaman", Label: "Nature's Swiftness", Tree: 2, Pos: 14},
	{ClassSlug: "shaman", Label: "Riptide", Tree: 2, Pos: 16, Preset: "RestorationShaman"},

	// Rogue (Assassination 17, Combat 17, Subtlety 19).
	{ClassSlug: "rogue", Label: "Cold Blood", Tree: 0, Pos: 11},
	{ClassSlug: "rogue", Label: "Mutilate", Tree: 0, Pos: 14},
	{ClassSlug: "rogue", Label: "Riposte", Tree: 1, Pos: 8},
	{ClassSlug: "rogue", Label: "Blade Flurry", Tree: 1, Pos: 13},
	{ClassSlug: "rogue", Label: "Adrenaline Rush", Tree: 1, Pos: 17},
	{ClassSlug: "rogue", Label: "Ghostly Strike", Tree: 2, Pos: 9},
	{ClassSlug: "rogue", Label: "Premeditation", Tree: 2, Pos: 12},
	{ClassSlug: "rogue", Label: "Preparation", Tree: 2, Pos: 15},
	{ClassSlug: "rogue", Label: "Hemorrhage", Tree: 2, Pos: 16},
}

// talentTreeSizesBySlug is used only to bounds-check each
// TalentGatedSpell's Pos against its class's real tree size (fail fast
// on a stale entry rather than silently building a one-point string
// past a tree that shrank), never to pad talentsString itself.
var talentTreeSizesBySlug = map[string][3]int{
	"hunter":  hunter.TalentTreeSizes,
	"mage":    mage.TalentTreeSizes,
	"warlock": warlock.TalentTreeSizes,
	"paladin": paladin.TalentTreeSizes,
	"warrior": warrior.TalentTreeSizes,
	"druid":   druid.TalentTreeSizes,
	"priest":  priest.TalentTreeSizes,
	"shaman":  shaman.TalentTreeSizes,
	"rogue":   rogue.TalentTreeSizes,
}

// repPresetForClass returns Presets' first entry for classSlug, purely
// as a source of a working Class/Race/SpecOptions/DistanceFromTarget to
// build from - every spec of a class shares the same class-level
// Initialize(), which is what registers every TalentGatedSpell, so
// which spec stands in does not affect which talent-gated spells are
// found.
func repPresetForClass(classSlug string) (Preset, bool) {
	for _, p := range Presets {
		if p.ClientClassSlug == classSlug {
			return p, true
		}
	}
	return Preset{}, false
}

func presetByLabel(label string) (Preset, bool) {
	for _, p := range Presets {
		if p.Label == label {
			return p, true
		}
	}
	return Preset{}, false
}

// collectTalentGatedRows builds one single-talent character per
// TalentGatedSpell belonging to classSlug and reports every row that
// talent's one point registers which an otherwise-identical EMPTY-talent
// build of the same preset does not - using the same rowFor/verdictFor
// path as the main table, restricted to that difference so an ordinary
// always-available spell this preset also happens to pick up (Frostbolt,
// Fireball, every spell already in the main table) does not leak into
// this section a second time under the gated talent's label. A spell
// gated by a talent this build's class package does not (yet) grant at
// the tested levels - or that isn't in the client's own class table -
// simply contributes no row, the same way rowFor already handles any
// other unregistered spell.
func collectTalentGatedRows(clientClass spellconst.Class, classSlug string) (rows []Row, buildErrors []string) {
	preset, ok := repPresetForClass(classSlug)
	if !ok {
		return nil, nil
	}

	treeSizes, ok := talentTreeSizesBySlug[classSlug]
	if !ok {
		return nil, []string{fmt.Sprintf("talent-gated spells for %s: no TalentTreeSizes registered", classSlug)}
	}

	baselines := map[string]map[int32]map[int32]bool{}
	baselineOf := func(p Preset) map[int32]map[int32]bool {
		if cached, ok := baselines[p.Label]; ok {
			return cached
		}
		byLevel, errs := baselineSpellIDsByLevel(p)
		buildErrors = append(buildErrors, errs...)
		baselines[p.Label] = byLevel
		return byLevel
	}

	seen := map[string]bool{}
	for _, gated := range TalentGatedSpells {
		if gated.ClassSlug != classSlug {
			continue
		}
		if gated.Pos < 1 || gated.Tree < 0 || gated.Tree > 2 || gated.Pos > treeSizes[gated.Tree] {
			buildErrors = append(buildErrors, fmt.Sprintf("%s: %s's talent position (tree %d, pos %d) is out of range for tree size %d", classSlug, gated.Label, gated.Tree, gated.Pos, treeSizes[gated.Tree]))
			continue
		}

		gatedPreset := preset
		if gated.Preset != "" {
			named, found := presetByLabel(gated.Preset)
			if !found {
				buildErrors = append(buildErrors, fmt.Sprintf("%s: %s names preset %q, which does not exist", classSlug, gated.Label, gated.Preset))
				continue
			}
			gatedPreset = named
		}
		baseline := baselineOf(gatedPreset)
		gatedPreset.Label = fmt.Sprintf("%s (%s talent)", gatedPreset.Label, gated.Label)
		gatedTalents := gated.talentsString()

		for _, level := range Levels {
			built, err := buildCharacter(gatedPreset, level, gatedTalents)
			if err != nil {
				buildErrors = append(buildErrors, fmt.Sprintf("%s/L%d: %v", gatedPreset.Label, level, err))
				continue
			}
			for _, spell := range built.Spellbook {
				if baseline[level][spell.ActionID.SpellID] {
					continue // registers with no talents too; not this talent's gate
				}
				row, ok := rowFor(clientClass, gatedPreset, level, built, spell)
				if !ok {
					continue
				}
				key := fmt.Sprintf("%s|%d|%d|%d", row.Spec, row.Level, row.SpellID, row.Rank)
				if seen[key] {
					continue
				}
				seen[key] = true
				rows = append(rows, row)
			}
		}
	}

	sortRows(rows)
	return rows, buildErrors
}

// baselineSpellIDsByLevel is preset's own empty-talent spellbook at
// every level in Levels, keyed by level then SpellID, so
// collectTalentGatedRows can tell "newly registered by this one talent"
// apart from "was already here with no talents at all".
func baselineSpellIDsByLevel(preset Preset) (byLevel map[int32]map[int32]bool, buildErrors []string) {
	byLevel = make(map[int32]map[int32]bool, len(Levels))
	for _, level := range Levels {
		ids := map[int32]bool{}
		built, err := buildCharacter(preset, level, "")
		if err != nil {
			buildErrors = append(buildErrors, fmt.Sprintf("%s/L%d: baseline for talent-gated spells: %v", preset.Label, level, err))
			byLevel[level] = ids
			continue
		}
		for _, spell := range built.Spellbook {
			ids[spell.ActionID.SpellID] = true
		}
		byLevel[level] = ids
	}
	return byLevel, buildErrors
}
