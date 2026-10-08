package holy

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
	"github.com/wowsims/classic/sim/healsim"
)

const (
	holyLightTopRank     int32 = 25292
	flashOfLightTopRank  int32 = 19943
	holyShockTopCast     int32 = 20930
	holyShockTopHeal     int32 = 25903
	lightsVigilTopCast   int32 = 1311595
	lightsVigilTopHeal   int32 = 1311596
	divineFavorSpell     int32 = 20216
	illuminationAction   int32 = 20210
	blessingOfLightTop   int32 = 19979
	maxIlluminationRanks       = 5
	percent                    = 0.01
)

// healFor is the client's mean (non-crit) heal of a spell at level 60 with
// the numeric tests' healing power.
func healFor(t *testing.T, healID int32) float64 {
	t.Helper()
	spell := clientSpell(t, healID)
	low, high, _ := spell.DamageRange(0, defaultLevel)
	return (low+high)/2 + spell.Effects[0].ResolvedSPCoefficient*healingPower
}

func TestHealingLightRaisesHolyLightFlashOfLightAndHolyShock(t *testing.T) {
	cases := []struct {
		name         string
		cast, healed int32
	}{
		{"Holy Light", holyLightTopRank, holyLightTopRank},
		{"Flash of Light", flashOfLightTopRank, flashOfLightTopRank},
		{"Holy Shock", holyShockTopCast, holyShockTopHeal},
	}
	for rank := 1; rank <= 3; rank++ {
		for _, c := range cases {
			f := numericFight(castLoop(c.cast))
			f.talents = map[string]int{"healing_light": rank, "holy_shock": 1}
			totals := totalsOf(f.run(t, numericRuns), c.healed)
			want := healFor(t, c.healed) * (1 + 4*percent*float64(rank))
			if got := totals.normalHeal(); !within(got, want, averageTolerance) {
				t.Errorf("%s with Healing Light %d: heals %.1f, want %.1f", c.name, rank, got, want)
			}
		}
	}
}

func TestDivineFavorMakesTheNextHealCritAndIsSpent(t *testing.T) {
	f := numericFight(rotationOf(castItem(divineFavorSpell, tankIndex), castItem(holyLightTopRank, tankIndex)))
	f.talents = map[string]int{"divine_favor": 1}
	f.duration = 3
	result := f.run(t, numericRuns)

	heals := totalsOf(result, holyLightTopRank)
	if heals.hits != 0 || heals.crits == 0 {
		t.Fatalf("the first heal after Divine Favor: %d hits and %d crits, want every one a crit", heals.hits, heals.crits)
	}
	if casts := totalsOf(result, divineFavorSpell).casts; casts != numericRuns {
		t.Errorf("Divine Favor cast %d times in %d fights, want once each", casts, numericRuns)
	}
}

func TestDivineFavorIsSpentByOneHealOnly(t *testing.T) {
	f := numericFight(rotationOf(castItem(divineFavorSpell, tankIndex), castItem(flashOfLightTopRank, tankIndex)))
	f.talents = map[string]int{"divine_favor": 1}
	f.duration = 30
	heals := totalsOf(f.run(t, numericRuns), flashOfLightTopRank)

	// Divine Favor's two minute cooldown allows it once, and it crits one
	// heal: every other heal in the fight is at the natural crit rate.
	if crits := float64(heals.crits) / numericRuns; crits < 1 || crits > 3 {
		t.Errorf("%.2f crits a fight, want the Divine Favor crit plus a few natural ones", crits)
	}
}

func TestIlluminationReturnsHalfTheBaseCostOnACrit(t *testing.T) {
	// Divine Favor forces the crit; rank 5 Illumination always procs.
	f := numericFight(rotationOf(castItem(divineFavorSpell, tankIndex), castItem(holyLightTopRank, tankIndex)))
	f.talents = map[string]int{"divine_favor": 1, "illumination": maxIlluminationRanks}
	f.duration = 3
	result := f.run(t, numericRuns)

	const holyLightRank9Cost = 660
	want := float64(numericRuns) * holyLightRank9Cost * illuminationShare
	if got := resourceGain(result, illuminationAction); !within(got, want, 0.001) {
		t.Errorf("Illumination returned %.0f mana over %d crits, want %.0f", got, numericRuns, want)
	}
}

const illuminationShare = 0.5

func TestIlluminationDoesNothingWithoutACrit(t *testing.T) {
	f := numericFight(castLoop(holyLightTopRank))
	f.talents = map[string]int{"illumination": maxIlluminationRanks}
	f.duration = 3
	result := f.run(t, numericRuns)

	heals := totalsOf(result, holyLightTopRank)
	want := float64(heals.crits) * 660 * illuminationShare
	if got := resourceGain(result, illuminationAction); !within(got, want, 0.001) {
		t.Errorf("Illumination returned %.0f mana for %d crits, want %.0f", got, heals.crits, want)
	}
}

func TestInfusionOfLightShortensTheNextHolyLight(t *testing.T) {
	for rank := 1; rank <= 2; rank++ {
		// Divine Favor crits the Flash of Light, which opens the window
		// for the Holy Light that follows it.
		f := numericFight(rotationOf(
			castItem(divineFavorSpell, tankIndex),
			whileBefore("1s", castItem(flashOfLightTopRank, tankIndex)),
			whileBefore("2s", castItem(holyLightTopRank, tankIndex)),
		))
		f.talents = map[string]int{"divine_favor": 1, "infusion_of_light": rank}
		f.duration = 6
		result := f.run(t, numericRuns)

		const holyLightCastMS = 2500.0
		want := holyLightCastMS - 500*float64(rank)
		casts := totalsOf(result, holyLightTopRank).casts
		if got := castTimeMS(result, holyLightTopRank) / float64(casts); got != want {
			t.Errorf("Infusion of Light %d: Holy Light cast time %.0fms, want %.0fms", rank, got, want)
		}
	}
}

func TestInfusionOfLightEndsWithTheHolyLightItShortened(t *testing.T) {
	f := numericFight(rotationOf(
		castItem(divineFavorSpell, tankIndex),
		whileBefore("1s", castItem(flashOfLightTopRank, tankIndex)),
		whileBefore("4s", castItem(holyLightTopRank, tankIndex)),
	))
	f.talents = map[string]int{"divine_favor": 1, "infusion_of_light": 2}
	f.duration = 9
	result := f.run(t, numericRuns)

	// Two Holy Lights start before 4s: the first at 1.5s for 1.5s, the
	// second at 3s for the full 2.5s.
	const want = 1500 + 2500
	if got := castTimeMS(result, holyLightTopRank) / numericRuns; got != want {
		t.Errorf("Holy Light cast time %.0fms a fight, want %dms (the shortened cast spent once)", got, want)
	}
}

func TestBlessingOfLightAddsBonusHealingToHolyLightAndFlashOfLight(t *testing.T) {
	cases := []struct {
		name  string
		heal  int32
		bonus float64
	}{
		{"Holy Light", holyLightTopRank, 400},
		{"Flash of Light", flashOfLightTopRank, 115},
	}
	// The blessing counts as bonus healing for the spell, so the
	// spell's coefficient scales it.
	for _, c := range cases {
		f := numericFight(rotationOf(
			whileBefore("1s", castItem(blessingOfLightTop, tankIndex)),
			castItem(c.heal, tankIndex),
		))
		totals := totalsOf(f.run(t, numericRuns), c.heal)
		want := healFor(t, c.heal) + clientSpell(t, c.heal).Effects[0].ResolvedSPCoefficient*c.bonus
		if got := totals.normalHeal(); !within(got, want, averageTolerance) {
			t.Errorf("%s under Blessing of Light: heals %.1f, want %.1f", c.name, got, want)
		}
	}
}

func TestBlessingOfLightOnAnotherMemberDoesNotHelpTheTank(t *testing.T) {
	f := numericFight(rotationOf(
		whileBefore("1s", castItem(blessingOfLightTop, 1)),
		castItem(holyLightTopRank, tankIndex),
	))
	totals := totalsOf(f.run(t, numericRuns), holyLightTopRank)
	if got, want := totals.normalHeal(), healFor(t, holyLightTopRank); !within(got, want, averageTolerance) {
		t.Errorf("Holy Light on the tank heals %.1f, want %.1f with no blessing on it", got, want)
	}
}

func TestLightsVigilTurnsTheNextHolyShockIntoAPartyHeal(t *testing.T) {
	member := int(healsim.MemberIndices[0])
	f := numericFight(rotationOf(
		whileBefore("1s", castItem(lightsVigilTopCast, member)),
		castItem(holyShockTopCast, member),
	))
	f.talents = map[string]int{"holy_shock": 1, "lights_vigil": 1}
	f.duration = 4.6
	result := f.run(t, numericRuns)

	// The target's party is the healer and the four members: five heals.
	const partySize = 5
	vigil := totalsOf(result, lightsVigilTopHeal)
	if got := float64(vigil.landed()) / numericRuns; got != partySize {
		t.Errorf("Light's Vigil healed %.2f units a fight, want the party of %d, once", got, partySize)
	}
	// And it cost that Holy Shock no cooldown: a second goes out one global
	// later, where a ten second cooldown would have allowed only one.
	if got := float64(totalsOf(result, holyShockTopCast).casts) / numericRuns; got != 2 {
		t.Errorf("%.2f Holy Shocks a fight, want 2 (the first left no cooldown)", got)
	}
}

func TestLightsVigilOnTheTankHealsOnlyTheTanksParty(t *testing.T) {
	f := numericFight(rotationOf(
		whileBefore("1s", castItem(lightsVigilTopCast, tankIndex)),
		castItem(holyShockTopCast, tankIndex),
	))
	f.talents = map[string]int{"holy_shock": 1, "lights_vigil": 1}
	f.duration = 3.1
	vigil := totalsOf(f.run(t, numericRuns), lightsVigilTopHeal)
	if got := float64(vigil.landed()) / numericRuns; got != 1 {
		t.Errorf("Light's Vigil healed %.2f units a fight, want only the tank", got)
	}
}

func TestLightsVigilHealsTheClientAmount(t *testing.T) {
	member := int(healsim.MemberIndices[0])
	f := numericFight(rotationOf(
		whileBefore("1s", castItem(lightsVigilTopCast, member)),
		castItem(holyShockTopCast, member),
	))
	f.talents = map[string]int{"holy_shock": 1, "lights_vigil": 1}
	f.duration = 3.1
	totals := totalsOf(f.run(t, numericRuns), lightsVigilTopHeal)
	if got, want := totals.normalHeal(), healFor(t, lightsVigilTopHeal); !within(got, want, averageTolerance) {
		t.Errorf("Light's Vigil heals %.1f, want %.1f", got, want)
	}
}

// unitsOf builds the healer's environment without running it.
func unitsOf(t *testing.T, f fight) (*core.Character, *core.Environment) {
	t.Helper()
	req := healsim.Request(f.player(t), idleRaid(), defaultSeconds, 1)
	env, _, _ := core.NewEnvironment(req.Raid, req.Encounter, true)
	return env.Raid.Parties[0].Players[healsim.HealerIndex].GetCharacter(), env
}

func TestSpiritualFocusKeepsTheCastTimeOnTheNamedHeals(t *testing.T) {
	healer, _ := unitsOf(t, fight{talents: map[string]int{"spiritual_focus": 2, "lights_vigil": 1}})
	const want = 0.70
	for name, id := range map[string]int32{"Holy Light": holyLightTopRank, "Flash of Light": flashOfLightTopRank, "Light's Vigil": lightsVigilTopCast} {
		if got := healer.GetSpell(core.ActionID{SpellID: id}).PushbackReduction; got != want {
			t.Errorf("%s keeps its cast time %.2f of the time, want %.2f", name, got, want)
		}
	}
}

func TestHolyPowerAddsCritToHolyShockAndHolyStrikeMoreThanToTheRest(t *testing.T) {
	healer, _ := unitsOf(t, fight{talents: map[string]int{"holy_power": 5, "holy_shock": 1}})
	critPerPercent := float64(core.CritRatingPerCritChance)
	cases := map[int32]float64{
		holyLightTopRank:    5,
		flashOfLightTopRank: 5,
		holyShockTopCast:    15,
		holyShockTopHeal:    15,
		10333:               15, // Holy Strike
	}
	for id, wantPercent := range cases {
		if got := healer.GetSpell(core.ActionID{SpellID: id}).BonusCritRating / critPerPercent; got < wantPercent-1e-9 || got > wantPercent+1e-9 {
			t.Errorf("spell %d: %.2f%% bonus crit, want %.0f%%", id, got, wantPercent)
		}
	}
}

func TestChampionOfTheLightGivesHealingAndDamageTheSameShareOfIntellect(t *testing.T) {
	base, _ := unitsOf(t, fight{})
	talented, _ := unitsOf(t, fight{talents: map[string]int{"champion_of_the_light": 3}})
	const share = 0.60
	intellect := talented.GetStat(stats.Intellect)

	if got := talented.GetStat(stats.HealingPower) - base.GetStat(stats.HealingPower); got < share*intellect-1e-6 || got > share*intellect+1e-6 {
		t.Errorf("healing power rose by %.3f, want %.3f (60%% of %.1f Intellect)", got, share*intellect, intellect)
	}
	damage := func(c *core.Character) float64 {
		return c.GetStat(stats.SpellPower) + c.GetStat(stats.SpellDamage)
	}
	if got := damage(talented) - damage(base); got < share*intellect-1e-6 || got > share*intellect+1e-6 {
		t.Errorf("spell damage rose by %.3f, want %.3f (the same 60%% of Intellect)", got, share*intellect)
	}
}
