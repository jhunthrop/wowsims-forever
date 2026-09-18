package core

import (
	"strings"
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// Ten races: the eight vanilla ones plus Skyborne's two faction rows.
// Skyborne is one neutral race whose faction is chosen at creation and
// whose second active differs by faction, which is why the client's own
// race table carries it as two rows and the engine follows.
func TestPlayableRacesAreTheTen(t *testing.T) {
	want := []proto.Race{
		proto.Race_RaceDwarf,
		proto.Race_RaceGnome,
		proto.Race_RaceHuman,
		proto.Race_RaceNightElf,
		proto.Race_RaceOrc,
		proto.Race_RaceTauren,
		proto.Race_RaceTroll,
		proto.Race_RaceUndead,
		proto.Race_RaceHighOrderSkyborne,
		proto.Race_RaceWindshaperSkyborne,
	}
	got := PlayableRaces()
	if len(got) != len(want) {
		t.Fatalf("PlayableRaces() has %d entries, want %d", len(got), len(want))
	}
	seen := map[proto.Race]bool{}
	for _, r := range got {
		seen[r] = true
	}
	for _, r := range want {
		if !seen[r] {
			t.Errorf("PlayableRaces() is missing %v", r)
		}
	}
}

// Forever: two active and two passive racials per race. The shape is
// confirmed by the Deep Dive panel; the numbers on seven of the forty
// entries are not, and UnconfirmedRacials names exactly those.
func TestEveryRaceHasTwoActivesAndTwoPassives(t *testing.T) {
	for _, race := range PlayableRaces() {
		t.Run(race.String(), func(t *testing.T) {
			got := RacialsFor(race)
			if len(got) != 4 {
				t.Fatalf("%v has %d racials, want 4", race, len(got))
			}
			var actives, passives int
			names := map[string]bool{}
			for _, r := range got {
				// RacialPassive is iota 0, so a zero-valued Kind is a
				// passive and "has no kind" is unrepresentable. The
				// default arm therefore catches only an out-of-range
				// value, and the assertion that carries the weight is
				// that the two counts come out at two and two.
				switch r.Kind {
				case RacialActive:
					actives++
				case RacialPassive:
					passives++
				default:
					t.Errorf("%v: %q has kind %d, which is neither active nor passive", race, r.Name, int(r.Kind))
				}
				if r.Name == "" {
					t.Errorf("%v: a racial has no name", race)
				}
				if names[r.Name] {
					t.Errorf("%v: %q is listed twice", race, r.Name)
				}
				names[r.Name] = true
				if r.Apply == nil {
					t.Errorf("%v: %q has no Apply function; a racial with no combat effect gets an Apply that does nothing and says so", race, r.Name)
				}
				if !r.Confirmed && r.Note == "" {
					t.Errorf("%v: %q is unconfirmed but says nothing about what is unread", race, r.Name)
				}
			}
			if actives != 2 {
				t.Errorf("%v has %d actives, want 2", race, actives)
			}
			if passives != 2 {
				t.Errorf("%v has %d passives, want 2", race, passives)
			}
		})
	}
}

// The two Skyborne rows share both passives and their first active; only
// the second active differs. Duplicating the shared three would let one
// drift from the other silently.
func TestSkyborneRowsDifferOnlyInTheirSecondActive(t *testing.T) {
	al := RacialsFor(proto.Race_RaceHighOrderSkyborne)
	ho := RacialsFor(proto.Race_RaceWindshaperSkyborne)
	alNames := make([]string, len(al))
	hoNames := make([]string, len(ho))
	for i := range al {
		alNames[i], hoNames[i] = al[i].Name, ho[i].Name
	}
	var differ int
	for i := range alNames {
		if alNames[i] != hoNames[i] {
			differ++
		}
	}
	if differ != 1 {
		t.Errorf("the two Skyborne rows differ in %d racials, want exactly 1 (Read Ley Line vs Skysight): %v vs %v", differ, alNames, hoNames)
	}
}

// Every named racial the demo transcriptions gave a number for must be in
// the table. This is the regression that catches a rewrite dropping one.
func TestTheNamedRacialsAreAllPresent(t *testing.T) {
	want := map[proto.Race][]string{
		proto.Race_RaceHuman:              {"Will to Survive", "Perception", "Sword Specialization", "The Human Spirit"},
		proto.Race_RaceOrc:                {"Blood Fury", "Shatter Curse", "Axe Specialization", "Hardiness"},
		proto.Race_RaceDwarf:              {"Stoneform", "Find Treasure", "Mace Specialization", "Big Game Hunter"},
		proto.Race_RaceNightElf:           {"Elune's Light", "Shadowmeld", "Quickness", "Wisp Spirit"},
		proto.Race_RaceUndead:             {"Will of the Forsaken", "Cannibalize", "Touch of the Grave"},
		proto.Race_RaceTauren:             {"War Stomp", "Endurance"},
		proto.Race_RaceGnome:              {"Escape Artist", "Eureka!", "Expansive Mind", "Engineering Specialization"},
		proto.Race_RaceTroll:              {"Berserking", "Rapid Regeneration", "Beast Slaying", "Regeneration"},
		proto.Race_RaceHighOrderSkyborne:  {"Walk on Air", "Read Ley Line", "Wind Blessed", "Elemental Insight"},
		proto.Race_RaceWindshaperSkyborne: {"Walk on Air", "Skysight", "Wind Blessed", "Elemental Insight"},
	}
	for race, names := range want {
		have := map[string]bool{}
		for _, r := range RacialsFor(race) {
			have[r.Name] = true
		}
		for _, n := range names {
			if !have[n] {
				t.Errorf("%v is missing the racial %q", race, n)
			}
		}
	}
}

// The eight entries whose numbers - or, for Undead's fourth racial, whose
// very name - the demo did not settle are named out loud, so the spec
// support page can say what the sim is guessing at. An unnamed or
// unpublished racial is Confirmed: false exactly like an unpublished
// percentage; a name is exactly as unconfirmed as a number.
func TestUnconfirmedRacialsNamesTheEight(t *testing.T) {
	got := UnconfirmedRacials()
	if len(got) == 0 {
		t.Skip("nothing is unconfirmed: the beta settled the numbers and this test has done its job")
	}
	// Print them: this list is what the spec support page shows, and it
	// is worth reading on every run rather than only when it breaks.
	for _, line := range got {
		t.Log(line)
	}
	joined := strings.Join(got, "\n")
	// All eight by name, and exactly eight. The count is asserted because
	// a ninth means a number was marked unconfirmed without anyone
	// deciding it was, and a seventh means one was quietly promoted to
	// confirmed - and a test named for eight that checks six would
	// notice neither.
	want := []string{
		"Mace Specialization",       // Dwarf: the crit percentage is unread
		"Big Game Hunter",           // Dwarf: the damage percentage is unread
		"Quickness",                 // Night Elf: 1% or 2% dodge, the two readings disagree
		"Berserking",                // Troll: 10 s or 12 s, the two readings disagree
		"Touch of the Grave",        // Undead: proc chance and amount unread
		"Unannounced Fourth Racial", // Undead: no source names this racial at all
		"Cultivation",               // Tauren: which of these two is the second
		"Plainsrunning",             //   active is unread; both are listed
	}
	for _, w := range want {
		if !strings.Contains(joined, w) {
			t.Errorf("UnconfirmedRacials() does not mention %q:\n%s", w, joined)
		}
	}
	if len(got) != len(want) {
		t.Errorf("UnconfirmedRacials() has %d entries, want %d:\n%s", len(got), len(want), joined)
	}
	for _, line := range got {
		if !strings.Contains(line, ": ") || !strings.Contains(line, "(") {
			t.Errorf("%q is not in the form \"<race>: <name> (<note>)\"", line)
		}
	}
}

// Tauren's Endurance grants 1% Hit, which after the Task 4 merge is one
// stat covering melee, ranged and spell. It is the only racial that
// touches the attack table, so a regression here is a silent DPS change
// for every Tauren - and the test therefore applies it to a character
// and reads stats.Hit back, rather than only checking that an entry of
// that name exists, which was all an earlier draft did.
//
// Endurance's Health bonus is a genuine multiplicative stat dependency
// (MultiplyStat), not a flat AddStat - production code carries no extra
// behaviour whose only purpose is to make an unfinalized GetStat read
// succeed. So this test gives the character a known base Health, finalizes
// the character's stat dependencies the same way the real engine does
// before a sim runs, and reads the finalized Health back, instead of
// reading GetStat on a character that was never finalized.
func TestEnduranceGrantsTheOneHitStat(t *testing.T) {
	var endurance *Racial
	for _, r := range RacialsFor(proto.Race_RaceTauren) {
		if r.Name == "Endurance" {
			r := r
			endurance = &r
		}
	}
	if endurance == nil {
		t.Fatal("Tauren has no Endurance")
	}
	if endurance.Kind != RacialPassive {
		t.Errorf("Endurance is %v, want a passive", endurance.Kind)
	}

	character := &Character{}
	const baseHealth = 4000.0
	character.AddStat(stats.Health, baseHealth)
	beforeHit := character.GetStat(stats.Hit)
	endurance.Apply(character)

	if got, want := character.GetStat(stats.Hit)-beforeHit, 1.0*HitRatingPerHitChance; got != want {
		t.Errorf("Endurance granted %v Hit, want %v (1%% at %v rating per percent)", got, want, HitRatingPerHitChance)
	}

	finalStats := character.StatDependencyManager.SortAndApplyStatDependencies(character.GetStats())
	if got, want := finalStats[stats.Health], baseHealth*1.05; got != want {
		t.Errorf("Endurance's finalized Health is %v, want %v (5%% of %v base Health); the demo reads it as 5%% Health and 1%% Hit", got, want, baseHealth)
	}
}
