package core

import (
	"math"
	"strings"
	"testing"
	"time"

	googleProto "google.golang.org/protobuf/proto"

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
// confirmed by the Deep Dive panel; the numbers on three of the forty
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
		proto.Race_RaceHuman:              {"Will to Survive", "Perception", "Ruthless", "The Human Spirit"},
		proto.Race_RaceOrc:                {"Blood Fury", "Shatter Curse", "Command", "Hardiness"},
		proto.Race_RaceDwarf:              {"Stoneform", "Find Treasure", "Artisan's Precision", "Big Game Hunter"},
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

// The three entries the client tables do not settle - Undead's fourth
// racial, which no source names, and Tauren's two candidate second
// actives, of which neither is known to be the one - are named out loud,
// so the spec
// support page can say what the sim is guessing at. An unnamed or
// unpublished racial is Confirmed: false exactly like an unpublished
// percentage; a name is exactly as unconfirmed as a number.
func TestUnconfirmedRacialsNamesTheThree(t *testing.T) {
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
	// All three by name, and exactly three (Mace Specialization left the
	// list when the client stated its 1%; Big Game Hunter, Quickness,
	// Berserking and Touch of the Grave left it on 2026-10-07 when the
	// client rows for build 1.60.1.70009 settled them). The count is
	// asserted because a fourth means a number was marked unconfirmed
	// without anyone deciding it was, and a second means one was quietly
	// promoted to confirmed.
	want := []string{
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

// Tauren's Endurance (20550) is 5% Health. Neither the client table nor
// Wowhead Forever's live tooltip (read 2026-11-01) gives it Hit: the earlier
// 1% Hit came from a demo transcription. The shipped table's 2% critical
// strike bonus (aura 163) is gone from the hotfix push's row, which states
// aura 133 alone.
func TestEnduranceIsHealthWithoutHit(t *testing.T) {
	endurance := racialNamed(t, proto.Race_RaceTauren, "Endurance")
	if endurance.Kind != RacialPassive {
		t.Errorf("Endurance is %v, want a passive", endurance.Kind)
	}

	character := &Character{}
	const baseHealth = 4000.0
	character.AddStat(stats.Health, baseHealth)
	endurance.Apply(character)

	if got := character.GetStat(stats.Hit); got != 0 {
		t.Errorf("Endurance granted %v Hit, want none", got)
	}
	finalStats := character.StatDependencyManager.SortAndApplyStatDependencies(character.GetStats())
	if got, want := finalStats[stats.Health], baseHealth*1.05; got != want {
		t.Errorf("Endurance's finalized Health is %v, want %v (5%% of %v base Health)", got, want, baseHealth)
	}
}

// Orc after Blizzard's 26 October 2026 racial notes ("Savage removed (1%
// increased critical strike chance)"; "Command re-added, increasing Warlock
// and Hunter pet damage by 5%") and the 30 October hotfix cache over build
// 1.60.1.70685: the orc has no crit of its own, and every pet deals 5% more.
func TestOrcHasCommandAndNoSavage(t *testing.T) {
	for _, r := range RacialsFor(proto.Race_RaceOrc) {
		if r.Name == "Savage" {
			t.Fatal("Orc still has Savage; Blizzard removed it on 26 October")
		}
	}
	command := racialNamed(t, proto.Race_RaceOrc, "Command")
	if command.Kind != RacialPassive {
		t.Errorf("Command is %v, want a passive", command.Kind)
	}
	// The client gives the damage to hunter and warlock pets only (20576
	// ClassMask 4, 20575 ClassMask 256); every other class's Command is the
	// dummy 21563 (ClassMask 1243), so an orc warrior's item guardian (the
	// Emerald Dragon Whelp) gains nothing.
	for _, c := range []struct {
		class proto.Class
		want  float64
	}{{proto.Class_ClassHunter, 1.05}, {proto.Class_ClassWarlock, 1.05}, {proto.Class_ClassWarrior, 1}, {proto.Class_ClassShaman, 1}} {
		pet := &Pet{Character: Character{Unit: Unit{PseudoStats: stats.NewPseudoStats()}}}
		character := &Character{Class: c.class, Pets: []*Pet{pet}}
		command.Apply(character)
		if got := character.GetStat(stats.Crit); got != 0 {
			t.Errorf("Command granted the orc %v Crit, want none", got)
		}
		if got := pet.PseudoStats.DamageDealtMultiplier; math.Abs(got-c.want) > 1e-12 {
			t.Errorf("Command left a %v's pet damage multiplier at %v, want %v", c.class, got, c.want)
		}
	}
}

// The Human Spirit (20598): Blizzard's 26 October notes, "Human Spirit now
// increases Spirit by 10% (was 5%)", and the 30 October hotfix cache. The
// shipped 1.60.1.70707 row still reads 5.
func TestTheHumanSpiritIsTenPercentSpirit(t *testing.T) {
	spirit := racialNamed(t, proto.Race_RaceHuman, "The Human Spirit")
	character := &Character{}
	const baseSpirit = 100.0
	character.AddStat(stats.Spirit, baseSpirit)
	spirit.Apply(character)
	finalStats := character.StatDependencyManager.SortAndApplyStatDependencies(character.GetStats())
	if got, want := finalStats[stats.Spirit], baseSpirit*1.10; math.Abs(got-want) > 1e-9 {
		t.Errorf("The Human Spirit's finalized Spirit is %v, want %v", got, want)
	}
}

// racialNamed is the race's racial of that name.
func racialNamed(t *testing.T, race proto.Race, name string) Racial {
	t.Helper()
	for _, r := range RacialsFor(race) {
		if r.Name == name {
			return r
		}
	}
	t.Fatalf("%v has no %s", race, name)
	return Racial{}
}

// newGnomeEurekaTestCaster builds a bare Gnome spellcaster with just enough
// wired up to exercise RegisterSpell/AddMajorCooldown/OnCastComplete - the
// same "bare Unit plus Env{MeasuringStats:true}" pattern buffs_test.go's
// newBareTestUnit and spell_school_test.go's casters use, rather than
// standing up a full running Simulation.
func newGnomeEurekaTestCaster(class proto.Class) *Character {
	character := &Character{
		Unit: Unit{
			Type:        PlayerUnit,
			Level:       60,
			auraTracker: newAuraTracker(),
			PseudoStats: stats.NewPseudoStats(),
			Env:         &Environment{MeasuringStats: true},
		},
		Race:  proto.Race_RaceGnome,
		Class: class,
	}
	character.BaseMana = 1000
	// AddMajorCooldown reads character.Env directly off this manager
	// without a nil check; NewCharacter wires it via initialize(), which
	// this bare test caster stands in for.
	character.majorCooldownManager = majorCooldownManager{character: character}
	return character
}

// racialByName returns one racial of a race, failing the test if absent.
func racialByName(t *testing.T, race proto.Race, name string) Racial {
	t.Helper()
	for _, r := range RacialsFor(race) {
		if r.Name == name {
			return r
		}
	}
	t.Fatalf("%v has no racial %q", race, name)
	return Racial{}
}

// racialPlayer builds a real one-player environment for a race and returns
// the character, with a single level 60 target of the given mob type.
func racialPlayer(t *testing.T, race proto.Race, mob proto.MobType) (*Character, *Environment) {
	t.Helper()
	target := googleProto.Clone(DefaultTargetProtoLvl60).(*proto.Target)
	target.MobType = mob
	env, _, _ := NewEnvironment(
		SinglePlayerRaidProto(&proto.Player{
			Name:      "Racial Test",
			Race:      race,
			Class:     proto.Class_ClassShaman,
			Spec:      &proto.Player_ElementalShaman{ElementalShaman: &proto.ElementalShaman{}},
			Equipment: &proto.EquipmentSpec{},
		}, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		&proto.Encounter{Duration: 180, Targets: []*proto.Target{target}},
		false,
	)
	return env.Raid.Parties[0].Players[0].GetCharacter(), env
}

func damageMultiplierAgainstTarget(character *Character, env *Environment) float64 {
	return character.AttackTables[env.Encounter.TargetUnits[0].UnitIndex][proto.CastType_CastTypeMainHand].DamageDealtMultiplier
}

// Beast Slaying 20557, Big Game Hunter 1259721 and Elemental Insight
// 1259707 are aura 168 with 5 base points: 5% damage dealt, gated on the
// target's creature type, and nothing else (no separate crit multiplier).
func TestMobTypeRacialsAreFivePercentAgainstTheirTypeOnly(t *testing.T) {
	for _, tc := range []struct {
		race    proto.Race
		hit     proto.MobType
		missing proto.MobType
	}{
		{proto.Race_RaceTroll, proto.MobType_MobTypeBeast, proto.MobType_MobTypeUndead},
		{proto.Race_RaceDwarf, proto.MobType_MobTypeBeast, proto.MobType_MobTypeUndead},
		{proto.Race_RaceHighOrderSkyborne, proto.MobType_MobTypeElemental, proto.MobType_MobTypeBeast},
		{proto.Race_RaceWindshaperSkyborne, proto.MobType_MobTypeElemental, proto.MobType_MobTypeBeast},
	} {
		t.Run(tc.race.String(), func(t *testing.T) {
			character, env := racialPlayer(t, tc.race, tc.hit)
			if got := damageMultiplierAgainstTarget(character, env); math.Abs(got-1.05) > 1e-9 {
				t.Errorf("against %v the damage multiplier is %v, want 1.05", tc.hit, got)
			}
			character, env = racialPlayer(t, tc.race, tc.missing)
			if got := damageMultiplierAgainstTarget(character, env); got != 1 {
				t.Errorf("against %v the damage multiplier is %v, want 1", tc.missing, got)
			}
		})
	}
}

// Quickness 20582: dodge +1% (and 2% movement speed, not modeled).
func TestQuicknessIsOnePercentDodge(t *testing.T) {
	night, _ := racialPlayer(t, proto.Race_RaceNightElf, proto.MobType_MobTypeUnknown)
	human, _ := racialPlayer(t, proto.Race_RaceHuman, proto.MobType_MobTypeUnknown)
	got := (night.GetStat(stats.Dodge) - human.GetStat(stats.Dodge)) / DodgeRatingPerDodgeChance
	if math.Abs(got-1) > 1e-9 {
		t.Errorf("Night Elf dodge exceeds a Human's by %v%%, want 1%% (client 20582)", got)
	}
}

// Wind Blessed 1259710: 2% spellcasting, melee and ranged haste. Blizzard's
// 26 October notes: "Wind-Blessed Haste bonus reduced to 2% (was 3%)"; the
// shipped 1.60.1.70707 rows still read 3.
func TestWindBlessedHastesCastingMeleeAndRanged(t *testing.T) {
	skyborne, _ := racialPlayer(t, proto.Race_RaceHighOrderSkyborne, proto.MobType_MobTypeUnknown)
	human, _ := racialPlayer(t, proto.Race_RaceHuman, proto.MobType_MobTypeUnknown)
	for name, ratio := range map[string]float64{
		"melee":  skyborne.PseudoStats.MeleeSpeedMultiplier / human.PseudoStats.MeleeSpeedMultiplier,
		"ranged": skyborne.PseudoStats.RangedSpeedMultiplier / human.PseudoStats.RangedSpeedMultiplier,
		"cast":   skyborne.PseudoStats.CastSpeedMultiplier / human.PseudoStats.CastSpeedMultiplier,
	} {
		if math.Abs(ratio-1.02) > 1e-9 {
			t.Errorf("%s speed ratio is %v, want 1.02", name, ratio)
		}
	}
}

// Elune's Light 1259799: crit +10% for 15 s, 3 min cooldown. Berserking
// 20554: 15 s, 3 min, no cost (Blizzard's 27 October notes, "Berserking
// duration increased to 15 seconds (was 10 seconds)", and the 30 October
// hotfix cache; the shipped row's SpellDuration still gives 10 s). Both read back from the registered spell and
// aura, which is where the engine holds the numbers.
func TestElunesLightAndBerserkingMatchTheClientRows(t *testing.T) {
	night, _ := racialPlayer(t, proto.Race_RaceNightElf, proto.MobType_MobTypeUnknown)
	elune := night.GetSpell(ActionID{SpellID: 1259799})
	if elune == nil {
		t.Fatal("Elune's Light is not registered under the client id 1259799")
	}
	if elune.CD.Duration != 3*time.Minute {
		t.Errorf("Elune's Light cooldown is %v, want 3m", elune.CD.Duration)
	}
	if aura := night.GetAura("Elune's Light"); aura == nil || aura.Duration != 15*time.Second {
		t.Errorf("Elune's Light aura = %v, want a 15 s aura", aura)
	}

	troll, _ := racialPlayer(t, proto.Race_RaceTroll, proto.MobType_MobTypeUnknown)
	berserking := troll.GetSpell(ActionID{SpellID: 26297})
	if berserking == nil {
		t.Fatal("Berserking is not registered")
	}
	if berserking.CD.Duration != 3*time.Minute {
		t.Errorf("Berserking cooldown is %v, want 3m", berserking.CD.Duration)
	}
	if berserking.Cost != nil {
		t.Errorf("Berserking has a resource cost; the client row has none")
	}
	if aura := troll.GetAura("Berserking"); aura == nil || aura.Duration != 15*time.Second {
		t.Errorf("Berserking aura = %v, want a 15 s aura", aura)
	}
}

// Berserking is a flat 10% to both casting and attack speed, for every
// class, never a health-scaled value and never the old 1/(1-x) cast-time
// reading that gave mana users 11.1%.
func TestBerserkingIsAFlatTenPercentHaste(t *testing.T) {
	troll, env := racialPlayer(t, proto.Race_RaceTroll, proto.MobType_MobTypeUnknown)
	sim := &Simulation{Environment: env}
	meleeBefore := troll.PseudoStats.MeleeSpeedMultiplier
	castBefore := troll.PseudoStats.CastSpeedMultiplier
	troll.GetAura("Berserking").Activate(sim)
	if got := troll.PseudoStats.MeleeSpeedMultiplier / meleeBefore; math.Abs(got-1.1) > 1e-9 {
		t.Errorf("Berserking attack speed ratio is %v, want 1.1", got)
	}
	if got := troll.PseudoStats.CastSpeedMultiplier / castBefore; math.Abs(got-1.1) > 1e-9 {
		t.Errorf("Berserking casting speed ratio is %v, want 1.1", got)
	}
}

// Blood Fury 20572: attack power, ranged attack power and spell power each
// +10% of the unit's current total, for 15 s, 2 min cooldown, off the global
// cooldown (SpellCooldowns StartRecoveryTime 0, as Berserking's). The spell
// power effect is aura 317 with misc value 0: no school mask and no
// damage-only selector, so it raises healing as well as spell damage.
func TestBloodFuryIsTenPercentOfTotalPowers(t *testing.T) {
	character := &Character{
		Unit: Unit{
			Type:        PlayerUnit,
			Level:       60,
			auraTracker: newAuraTracker(),
			PseudoStats: stats.NewPseudoStats(),
			Env:         &Environment{MeasuringStats: true},
		},
		Race: proto.Race_RaceOrc,
	}
	character.majorCooldownManager = majorCooldownManager{character: character}
	racialByName(t, proto.Race_RaceOrc, "Blood Fury").Apply(character)

	spell := character.GetSpell(ActionID{SpellID: 20572})
	if spell == nil {
		t.Fatal("Blood Fury is not registered")
	}
	if spell.CD.Duration != 2*time.Minute {
		t.Errorf("Blood Fury cooldown is %v, want 2m", spell.CD.Duration)
	}
	if spell.DefaultCast.GCD != 0 {
		t.Errorf("Blood Fury triggers a %v global cooldown, want none", spell.DefaultCast.GCD)
	}
	aura := character.GetAura("Blood Fury")
	if aura.Duration != 15*time.Second {
		t.Errorf("Blood Fury lasts %v, want 15s", aura.Duration)
	}

	base := stats.Stats{}
	base[stats.AttackPower] = 1000
	base[stats.RangedAttackPower] = 800
	base[stats.SpellPower] = 500
	base[stats.HealingPower] = 600
	base[stats.Strength] = 200 // not a percentage target: must stay put

	final := func() stats.Stats { return character.StatDependencyManager.SortAndApplyStatDependencies(base) }
	if got := final(); got[stats.AttackPower] != 1000 {
		t.Fatalf("before the aura attack power is %v, want 1000", got[stats.AttackPower])
	}
	aura.Activate(&Simulation{})
	got := final()
	for _, tc := range []struct {
		stat stats.Stat
		want float64
	}{
		{stats.AttackPower, 1100},
		{stats.RangedAttackPower, 880},
		{stats.SpellPower, 550},
		{stats.HealingPower, 660},
		{stats.Strength, 200},
	} {
		if math.Abs(got[tc.stat]-tc.want) > 1e-9 {
			t.Errorf("with Blood Fury %v is %v, want %v", tc.stat.StatName(), got[tc.stat], tc.want)
		}
	}
	aura.Deactivate(&Simulation{})
	if got := final(); got[stats.AttackPower] != 1000 || got[stats.SpellPower] != 500 {
		t.Errorf("after the aura attack power is %v and spell power %v, want 1000 and 500", got[stats.AttackPower], got[stats.SpellPower])
	}
}

// Eureka! is one client row per class (SpellClassSet): Warrior 1259813,
// Rogue 1259812, Mage 1259817, Priest 1259823, Warlock 1259821, the five
// classes combos.json lets be Gnome. Each lasts 8 s (DurationIndex 31) with
// no SpellAuraOptions row, so no charges, on a 2 min cooldown. Wowhead
// Forever: "deal 10% more damage for 8 sec".
func TestEurekaMatchesTheClientRow(t *testing.T) {
	for class, want := range map[proto.Class]int32{
		proto.Class_ClassWarrior: 1259813,
		proto.Class_ClassRogue:   1259812,
		proto.Class_ClassMage:    1259817,
		proto.Class_ClassPriest:  1259823,
		proto.Class_ClassWarlock: 1259821,
	} {
		character := newGnomeEurekaTestCaster(class)
		racialByName(t, proto.Race_RaceGnome, "Eureka!").Apply(character)
		spell := character.GetSpell(ActionID{SpellID: want})
		if spell == nil {
			t.Errorf("%v: Eureka! is not registered under the client id %d", class, want)
			continue
		}
		if spell.CD.Duration != 2*time.Minute {
			t.Errorf("%v: Eureka! cooldown is %v, want 2m", class, spell.CD.Duration)
		}
		aura := character.GetAura("Eureka!")
		if aura.Duration != 8*time.Second {
			t.Errorf("%v: Eureka! lasts %v, want 8s", class, aura.Duration)
		}
		if aura.MaxStacks != 0 {
			t.Errorf("%v: Eureka! has %d charges, want none", class, aura.MaxStacks)
		}
	}
}

func newEurekaTestSpell(character *Character, spellID int32, school SpellSchool, flags SpellFlag) *Spell {
	return character.RegisterSpell(SpellConfig{
		ActionID:         ActionID{SpellID: spellID},
		SpellSchool:      school,
		ProcMask:         ProcMaskSpellDamage,
		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		Flags:            SpellFlagAPL | flags,
		ManaCost:         ManaCostOptions{BaseCost: 0.1},
		Cast:             CastConfig{DefaultCast: Cast{GCD: GCDDefault}},
		ApplyEffects:     func(*Simulation, *Unit, *Spell) {},
	})
}

// The warlock row's modifiers reach only their class masks: damage (misc 0)
// Shadow Bolt but not Corruption or Incinerate, periodic (misc 22) Drain
// Life, cost (misc 14) both Shadow Bolt and Drain Life. Casts spend nothing.
func TestEurekaReachesOnlyItsClassMasks(t *testing.T) {
	character := newGnomeEurekaTestCaster(proto.Class_ClassWarlock)
	shadowBolt := newEurekaTestSpell(character, 11661, SpellSchoolShadow, 0)
	corruption := newEurekaTestSpell(character, 11672, SpellSchoolShadow, 0)
	incinerate := newEurekaTestSpell(character, 1293813, SpellSchoolFire, 0)
	drainLife := newEurekaTestSpell(character, 11700, SpellSchoolShadow, SpellFlagChanneled)
	racialByName(t, proto.Race_RaceGnome, "Eureka!").Apply(character)

	type snapshot struct{ impact, periodic, cost float64 }
	read := func(spell *Spell) snapshot {
		return snapshot{spell.ImpactDamageMultiplierAdditive, spell.PeriodicDamageMultiplierAdditive, spell.Cost.GetCurrentCost()}
	}
	before := map[*Spell]snapshot{shadowBolt: read(shadowBolt), corruption: read(corruption), incinerate: read(incinerate), drainLife: read(drainLife)}

	sim := &Simulation{}
	eureka := character.GetSpell(ActionID{SpellID: 1259821})
	eureka.ApplyEffects(sim, nil, eureka)
	aura := character.GetAura("Eureka!")

	check := func(name string, spell *Spell, impact, periodic, costFactor float64) {
		t.Helper()
		got, was := read(spell), before[spell]
		if math.Abs(got.impact-was.impact-impact) > 1e-9 || math.Abs(got.periodic-was.periodic-periodic) > 1e-9 {
			t.Errorf("%s: impact +%.2f periodic +%.2f, want +%.2f / +%.2f", name, got.impact-was.impact, got.periodic-was.periodic, impact, periodic)
		}
		if math.Abs(got.cost-was.cost*costFactor) > 1e-9 {
			t.Errorf("%s: costs %v, want %v", name, got.cost, was.cost*costFactor)
		}
	}
	check("Shadow Bolt", shadowBolt, 0.10, 0, 0.9)
	check("Corruption", corruption, 0, 0, 1)
	check("Incinerate", incinerate, 0, 0, 1)
	check("Drain Life", drainLife, 0, 0.10, 0.9)

	for i := 0; i < 4; i++ {
		character.OnCastComplete(sim, shadowBolt)
	}
	if !aura.IsActive() {
		t.Error("Eureka! ended after four casts; it has no charges and lasts 8 s")
	}
	aura.Deactivate(sim)
	for spell, was := range before {
		if read(spell) != was {
			t.Errorf("%v not restored after Eureka! ends", spell.ActionID)
		}
	}
}

// The Rage row (1259813) reaches warrior abilities: Bloodthirst's damage.
func TestEurekaReachesWarriorAbilities(t *testing.T) {
	character := newGnomeEurekaTestCaster(proto.Class_ClassWarrior)
	bloodthirst := newEurekaTestSpell(character, 23894, SpellSchoolPhysical, 0)
	racialByName(t, proto.Race_RaceGnome, "Eureka!").Apply(character)
	before := bloodthirst.ImpactDamageMultiplierAdditive
	eureka := character.GetSpell(ActionID{SpellID: 1259813})
	eureka.ApplyEffects(&Simulation{}, nil, eureka)
	if got := bloodthirst.ImpactDamageMultiplierAdditive - before; math.Abs(got-0.10) > 1e-9 {
		t.Errorf("Bloodthirst impact multiplier rose by %.2f under Eureka!, want 0.10", got)
	}
}

// Touch of the Grave, client 1.60.1.70707: two passives by class
// (SkillLineAbility 55923 and 55924). 1260189 (ClassMask 11: warrior,
// paladin, rogue) has ProcChance 20, 1260201 (ClassMask 400: priest, mage,
// warlock) ProcChance 30, and ProcTypeMask 69972 (no periodic bit). The
// internal cooldown is 15 s: Blizzard's 27 October notes ("Touch of the
// Grave internal cooldown reduced to 15 seconds (was 25 seconds)") and the
// 30 October hotfix cache, over the shipped 25000 ms ProcCategoryRecovery. The drain 1260198 takes 5% of the
// caster's maximum Health.
func TestTouchOfTheGraveMatchesTheClientRows(t *testing.T) {
	for _, c := range []struct {
		class  proto.Class
		id     int32
		chance float64
	}{
		{proto.Class_ClassPaladin, 1260189, 0.20},
		{proto.Class_ClassWarrior, 1260189, 0.20},
		{proto.Class_ClassRogue, 1260189, 0.20},
		{proto.Class_ClassPriest, 1260201, 0.30},
		{proto.Class_ClassMage, 1260201, 0.30},
		{proto.Class_ClassWarlock, 1260201, 0.30},
	} {
		trigger := touchOfTheGraveTrigger(nil, c.class)
		if trigger.ActionID.SpellID != c.id || trigger.ProcChance != c.chance {
			t.Errorf("%v: passive %d at %v, want %d at %v", c.class, trigger.ActionID.SpellID, trigger.ProcChance, c.id, c.chance)
		}
		if trigger.ICD != 15*time.Second {
			t.Errorf("%v: internal cooldown is %v, want 15s", c.class, trigger.ICD)
		}
	}
	trigger := touchOfTheGraveTrigger(nil, proto.Class_ClassPaladin)
	if trigger.Callback != CallbackOnSpellHitDealt {
		t.Errorf("callback is %v, want direct hits only (the mask has no periodic bit)", trigger.Callback)
	}
	if want := ProcMaskMeleeOrRanged | ProcMaskSpellDamage; trigger.ProcMask != want {
		t.Errorf("proc mask is %v, want melee, ranged and spell damage %v", trigger.ProcMask, want)
	}
	if trigger.Outcome != OutcomeLanded {
		t.Errorf("outcome is %v, want landed hits only", trigger.Outcome)
	}
	if got := touchOfTheGraveDamage(4000); got != 200 {
		t.Errorf("drain for 4000 maximum Health is %v, want 200 (5%%)", got)
	}

	undead, env := racialPlayer(t, proto.Race_RaceUndead, proto.MobType_MobTypeUnknown)
	drain := undead.GetSpell(ActionID{SpellID: 1260198})
	if drain == nil {
		t.Fatal("the drain spell 1260198 is not registered for Undead")
	}
	if drain.SpellSchool != SpellSchoolShadow {
		t.Errorf("drain school is %v, want Shadow", drain.SpellSchool)
	}
	// SpellCategories gives 1260198 DefenseType 1 (magic) and SpellMisc
	// Attributes_2 0x20000000 (cannot crit): the drain rolls the spell hit
	// table and never crits.
	if drain.DefenseType != DefenseTypeMagic {
		t.Errorf("drain defense type is %v, want magic", drain.DefenseType)
	}
	at := undead.AttackTables[env.Encounter.TargetUnits[0].UnitIndex][proto.CastType_CastTypeMainHand]
	if got := drain.SpellChanceToMiss(at); math.Abs(got-0.17) > 1e-9 {
		t.Errorf("drain misses a level-63 target %.3f of the time with no hit, want the spell table's 0.17", got)
	}
	if aura := undead.GetAura("Touch of the Grave"); aura == nil {
		t.Error("Undead has no Touch of the Grave aura")
	}
	human, _ := racialPlayer(t, proto.Race_RaceHuman, proto.MobType_MobTypeUnknown)
	if human.GetAura("Touch of the Grave") != nil {
		t.Error("a Human must not have Touch of the Grave")
	}
}
