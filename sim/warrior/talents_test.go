package warrior

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/talents"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// The generated tree must be the client's: Arms 17, Fury 18,
// Protection 18, from build 1.60.1.69893. If Task 17 has not run, or
// ran against Era, this fails first and everything below it is noise.
func TestTheGeneratedTreeIsTheClients(t *testing.T) {
	if got, want := TalentTreeSizes, [3]int{17, 18, 18}; got != want {
		t.Fatalf("TalentTreeSizes = %v, want %v (Arms 17, Fury 18, Protection 18)", got, want)
	}
	if TalentsBuild == "" {
		t.Error("TalentsBuild is empty; the generated file must record the client build")
	}
}

// The talent string is parsed positionally against TalentTreeSizes, so a
// tree size that does not match the proto's field count silently reads
// the wrong talent. Task 17 generates both from one file precisely so
// they cannot disagree; this is the assertion that says so.
func TestTalentTreeSizesMatchTheProto(t *testing.T) {
	var total int
	for _, n := range TalentTreeSizes {
		total += n
	}
	fields := (&proto.WarriorTalents{}).ProtoReflect().Descriptor().Fields()
	if fields.Len() != total {
		t.Errorf("WarriorTalents has %d fields, TalentTreeSizes sums to %d", fields.Len(), total)
	}
	if total != 53 {
		t.Errorf("the warrior has %d talents, want 53 from build %s", total, TalentsBuild)
	}
}

// foreverFuryTalentsApplied is every talent ApplyTalents reads, by its
// generated proto field name. Keep it in step with talents.go by hand:
// it is the list the next test checks against the client's own tree, and
// a talent that is applied but absent from this list is one this test
// cannot protect.
var foreverFuryTalentsApplied = []string{
	// Flat stats, in ApplyTalents.
	"cruelty",
	"precision",
	"toughness",
	"anticipation",
	"deflection",
	// Declarative mods, in applyDeclarativeTalents.
	"improved_heroic_strike",
	"improved_cleave",
	"improved_execute",
	"improved_thunder_clap",
	"improved_sunder_armor",
	"two_handed_weapon_specialization",
	// Read by stances.go's rage retention.
	"improved_tactical_mastery",
	// Read by an ability file rather than by a mod.
	"impale",
	"improved_overpower",
	"improved_rend",
	"improved_slam",
	"improved_bloodrage",
	"improved_berserker_rage",
	"improved_shield_wall",
	"defiance",
	"booming_voice",
	"mortal_strike",
	"shield_slam",
	"bloodthirst",
	"piercing_howl",
	// Talents with their own function.
	"anger_management",
	"deep_wounds",
	"weaponmaster",
	"unbridled_wrath",
	"dual_wield_specialization",
	"enrage",
	"flurry",
	"shield_specialization",
	"death_wish",
	"sweeping_strikes",
	"last_stand",
}

// Every talent this spec's behaviour reads must exist in the generated
// tree. A typo'd field name compiles if another talent happens to share
// it and silently applies the wrong effect otherwise.
func TestEveryTalentThisSpecAppliesExists(t *testing.T) {
	for _, name := range foreverFuryTalentsApplied {
		if _, ok := TalentNodeIDs[name]; !ok {
			t.Errorf("talents.go applies %q, which is not in the client's tree", name)
		}
		if len(TalentSpellIDs[name]) == 0 {
			t.Errorf("%q has no rank spell ids", name)
		}
	}
}

// Every ability a talent modifies must carry a ClassSpellMask, or the
// declarative mod silently applies to nothing.
func TestFurySpellsCarryTheirMasks(t *testing.T) {
	cases := []struct {
		name string
		mask uint64
	}{
		{"Bloodthirst", WarriorSpellMaskBloodthirst},
		{"Whirlwind", WarriorSpellMaskWhirlwind},
		{"Execute", WarriorSpellMaskExecute},
		{"Heroic Strike", WarriorSpellMaskHeroicStrike},
		{"Cleave", WarriorSpellMaskCleave},
		{"Mortal Strike", WarriorSpellMaskMortalStrike},
		{"Overpower", WarriorSpellMaskOverpower},
		{"Rend", WarriorSpellMaskRend},
		{"Revenge", WarriorSpellMaskRevenge},
		{"Shield Slam", WarriorSpellMaskShieldSlam},
		{"Slam", WarriorSpellMaskSlam},
		{"Sunder Armor", WarriorSpellMaskSunderArmor},
		{"Thunder Clap", WarriorSpellMaskThunderClap},
		{"Hamstring", WarriorSpellMaskHamstring},
		{"Pummel", WarriorSpellMaskPummel},
		{"Piercing Howl", WarriorSpellMaskPiercingHowl},
	}
	seen := uint64(0)
	for _, c := range cases {
		if c.mask == 0 {
			t.Errorf("%s has a zero mask; an empty mask matches nothing", c.name)
		}
		if seen&c.mask != 0 {
			t.Errorf("%s reuses a bit already taken", c.name)
		}
		seen |= c.mask
	}

	// The groups must be subsets of the single-spell bits, or a talent
	// config targets a bit no spell carries.
	for name, group := range map[string]uint64{
		"WarriorSpellMaskSpecials":    WarriorSpellMaskSpecials,
		"WarriorSpellMaskOnNextSwing": WarriorSpellMaskOnNextSwing,
	} {
		if group&^seen != 0 {
			t.Errorf("%s names a bit no ability carries", name)
		}
	}
}

// The reference Fury build must spend exactly 51 points and must reach
// the 31-point talent in Fury, or the suite is validating a build nobody
// would play.
func TestForeverFuryTalentsAreAValidBuild(t *testing.T) {
	// Each segment must be exactly its tree's width, or every talent
	// after the short one is read from the wrong position and nothing
	// complains.
	parts := strings.Split(ForeverFuryTalents, "-")
	if len(parts) != 3 {
		t.Fatalf("ForeverFuryTalents has %d segments, want 3", len(parts))
	}
	var spent int
	for i, part := range parts {
		if len(part) != TalentTreeSizes[i] {
			t.Errorf("segment %d is %d characters, want %d", i, len(part), TalentTreeSizes[i])
		}
		for _, c := range part {
			if c < '0' || c > '9' {
				t.Fatalf("segment %d contains %q", i, c)
			}
			spent += int(c - '0')
		}
	}
	if spent != 51 {
		t.Errorf("the reference build spends %d points, want 51", spent)
	}
	if sum(parts[1]) != 31 {
		t.Errorf("the reference build spends %d points in Fury, want 31 to reach the capstone", sum(parts[1]))
	}

	talents := &proto.WarriorTalents{}
	fillWarriorTalents(talents, ForeverFuryTalents)
	if !talents.Bloodthirst {
		t.Error("the reference Fury build does not take Bloodthirst, the 31-point Fury talent")
	}
}

// A talent string that spends points the tier gates do not open, or
// skips a prerequisite, is a build the client would refuse. The suite
// would still run it, and its DPS would be a number nobody can reach.
func TestTheReferenceBuildsRespectTheTiersAndPrerequisites(t *testing.T) {
	for name, build := range map[string]string{
		"ForeverFuryTalents":       ForeverFuryTalents,
		"ForeverProtectionTalents": ForeverProtectionTalents,
	} {
		t.Run(name, func(t *testing.T) {
			assertWarriorBuildIsLegal(t, build)
		})
	}
}

// assertWarriorBuildIsLegal walks the client's own tier and prerequisite
// data and checks the build against it. The data is read from
// ui/core/talents/trees/warrior.json, which Task 17 generated from the
// same trait tables as the proto, rather than retyped here: a tier or a
// prerequisite typed into a test is a second source for something the
// client already states.
func assertWarriorBuildIsLegal(t *testing.T, build string) {
	t.Helper()

	trees := loadGeneratedWarriorTrees(t)
	spends := warriorTalentSpends(t, build)

	for _, tree := range trees {
		byLocation := map[[2]int]uiTalent{}
		for _, talent := range tree.Talents {
			byLocation[[2]int{talent.Location.RowIdx, talent.Location.ColIdx}] = talent
		}

		// The tier gate below accumulates points in list order, so it
		// is only correct while the generator emits a tree sorted by
		// row. It does today; this is the assertion that says so rather
		// than the check silently going soft if that ever changes.
		for i := 1; i < len(tree.Talents); i++ {
			if tree.Talents[i].Location.RowIdx < tree.Talents[i-1].Location.RowIdx {
				t.Fatalf("%s: the generated tree is not sorted by row (talent %d is tier %d after tier %d); "+
					"the cumulative tier-gate check below depends on that order",
					tree.Name, i, tree.Talents[i].Location.RowIdx, tree.Talents[i-1].Location.RowIdx)
			}
		}

		var cumulative int
		for _, talent := range tree.Talents {
			field := snakeCase(talent.FieldName)
			points := spends[field]
			if points == 0 {
				continue
			}
			if points > talent.MaxPoints {
				t.Errorf("%s: the build spends %d points, the client's max rank is %d", field, points, talent.MaxPoints)
			}

			gate := 5 * talent.Location.RowIdx
			if cumulative < gate {
				t.Errorf("%s is tier %d in %s and needs %d points spent there, but the build has spent %d",
					field, talent.Location.RowIdx, tree.Name, gate, cumulative)
			}
			// The PrereqRank > 0 guard is load-bearing, not defensive:
			// a talent with no prerequisite leaves PrereqLocation at
			// its zero value, which is a real grid position (tier 0,
			// column 0) and would otherwise be read as "requires the
			// first talent in the tree".
			if prereq, ok := byLocation[[2]int{talent.PrereqLocation.RowIdx, talent.PrereqLocation.ColIdx}]; ok && talent.PrereqRank > 0 {
				if got := spends[snakeCase(prereq.FieldName)]; got < talent.PrereqRank {
					t.Errorf("%s needs %s at rank %d, the build has rank %d",
						field, snakeCase(prereq.FieldName), talent.PrereqRank, got)
				}
			}

			cumulative += points
		}
	}
}

// The one-point talents are the tier gates stated as totals: a
// one-point talent at tier r is takeable the moment the tree holds
// 5r+1 points, so the set of those totals across every tree is exactly
// talents.ForeverMilestones. Task 17 declares that table once for all
// nine classes; this is the warrior's half of the check that it has not
// drifted from the trees it describes.
func TestTheOnePointTalentsSitOnTheForeverMilestones(t *testing.T) {
	found := map[int]bool{}
	for _, tree := range loadGeneratedWarriorTrees(t) {
		for _, talent := range tree.Talents {
			if talent.MaxPoints != 1 {
				continue
			}
			found[5*talent.Location.RowIdx+1] = true
		}
	}
	for _, milestone := range talents.ForeverMilestones {
		if !found[milestone] {
			t.Errorf("no warrior tree has a one-point talent takeable at %d points, but ForeverMilestones names it", milestone)
		}
		delete(found, milestone)
	}
	for total := range found {
		t.Errorf("a warrior one-point talent is takeable at %d points, which ForeverMilestones %v does not name",
			total, talents.ForeverMilestones)
	}
}

type uiLocation struct {
	RowIdx int `json:"rowIdx"`
	ColIdx int `json:"colIdx"`
}

type uiTalent struct {
	FieldName      string     `json:"fieldName"`
	Location       uiLocation `json:"location"`
	MaxPoints      int        `json:"maxPoints"`
	PrereqLocation uiLocation `json:"prereqLocation"`
	PrereqRank     int        `json:"prereqRank"`
}

type uiTree struct {
	Name    string     `json:"name"`
	Talents []uiTalent `json:"talents"`
}

func loadGeneratedWarriorTrees(t *testing.T) []uiTree {
	t.Helper()

	raw, err := os.ReadFile("../../ui/core/talents/trees/warrior.json")
	if err != nil {
		t.Fatalf("reading the generated UI tree: %v", err)
	}
	var trees []uiTree
	if err := json.Unmarshal(raw, &trees); err != nil {
		t.Fatalf("parsing the generated UI tree: %v", err)
	}
	if len(trees) != len(TalentTreeSizes) {
		t.Fatalf("the generated UI tree has %d trees, TalentTreeSizes has %d", len(trees), len(TalentTreeSizes))
	}
	for i, tree := range trees {
		if len(tree.Talents) != TalentTreeSizes[i] {
			t.Errorf("%s has %d talents in the UI tree, TalentTreeSizes says %d", tree.Name, len(tree.Talents), TalentTreeSizes[i])
		}
	}
	return trees
}

// snakeCase converts the UI tree's lowerCamel field names into the
// proto's snake_case ones. Both are generated from the same table, so
// the conversion is mechanical rather than a mapping to maintain.
func snakeCase(camel string) string {
	var b strings.Builder
	for _, r := range camel {
		if r >= 'A' && r <= 'Z' {
			b.WriteByte('_')
			b.WriteRune(r + ('a' - 'A'))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// warriorTalentSpends reads a talent string back out of the proto, so
// the positional parse this package ships is the one under test rather
// than a second parser written for the test.
func warriorTalentSpends(t *testing.T, build string) map[string]int {
	t.Helper()

	parsed := &proto.WarriorTalents{}
	fillWarriorTalents(parsed, build)

	spends := map[string]int{}
	message := parsed.ProtoReflect()
	fields := message.Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		name := string(field.Name())
		value := message.Get(field)
		if field.Kind() == protoreflect.BoolKind {
			if value.Bool() {
				spends[name] = 1
			}
			continue
		}
		spends[name] = int(value.Int())
	}
	return spends
}

func sum(segment string) int {
	var n int
	for _, c := range segment {
		n += int(c - '0')
	}
	return n
}

// The on-next-swing abilities are registered twice: the ability itself,
// which deals the damage, and a queue spell at tag 1, which is what a
// player actually presses. The engine resolves an APL's spell id by
// ActionID, tag included, so a rotation that names Heroic Strike without
// tag 1 gets the direct spell - and the direct spell carries
// SpellFlagNoOnCastComplete, costs no global cooldown, and can therefore
// be cast on every APL iteration as a free extra weapon swing.
//
// On the phase-1 Fury profile that mistake is worth 5,996 DPS against
// 1,739 - measured one run each, everything but the tag held fixed - so
// it is not a rounding error a reviewer would spot in a golden.
//
// ui/warrior/apls/forever_fury.apl.json is pinned from
// data/curated/apl/warrior-fury.json. That file omitted the tag when
// this task landed and the pin added it by hand; the data lane has
// since corrected the canonical (site main, commit ed49176), so the
// pinned copy is now that file's `rotation` value verbatim, dedented by
// the two spaces the wrapper adds and nothing else. It carries no note
// of its own precisely so the next `make engine-pin` is a byte-for-byte
// copy - the reasoning lives here instead, and this test is what stops
// a future re-pin from quietly dropping the tag again.
func TestTheForeverFuryRotationQueuesHeroicStrike(t *testing.T) {
	raw, err := os.ReadFile("../../ui/warrior/apls/forever_fury.apl.json")
	if err != nil {
		t.Fatalf("reading the pinned Forever Fury rotation: %v", err)
	}

	var rotation struct {
		PriorityList []struct {
			Action struct {
				CastSpell *struct {
					SpellID struct {
						SpellID int32 `json:"spellId"`
						Tag     int32 `json:"tag"`
					} `json:"spellId"`
				} `json:"castSpell"`
			} `json:"action"`
		} `json:"priorityList"`
	}
	if err := json.Unmarshal(raw, &rotation); err != nil {
		t.Fatalf("parsing the pinned Forever Fury rotation: %v", err)
	}

	// The ids the engine registers for the two on-next-swing abilities,
	// read from the ability files rather than retyped.
	onNextSwing := map[int32]string{
		heroicStrikeSpellID(): "Heroic Strike",
		cleaveSpellID():       "Cleave",
	}

	var found int
	for _, entry := range rotation.PriorityList {
		cast := entry.Action.CastSpell
		if cast == nil {
			continue
		}
		name, ok := onNextSwing[cast.SpellID.SpellID]
		if !ok {
			continue
		}
		found++
		if cast.SpellID.Tag != 1 {
			t.Errorf("the rotation casts %s (spell %d) with tag %d; it must be tag 1, the queued "+
				"on-next-swing spell, or the engine casts the free instant one every APL iteration",
				name, cast.SpellID.SpellID, cast.SpellID.Tag)
		}
	}
	if found == 0 {
		t.Error("the rotation casts neither Heroic Strike nor Cleave; the rage dump has gone missing")
	}
}
