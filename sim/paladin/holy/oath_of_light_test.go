package holy

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/healsim"
)

const oathOfLightSpell int32 = 1328548

// Oath of Light (talent 1328548, a spell the 70707 hotfix push adds):
// "Swear an Oath to protect target ally for 5 min, increasing the healing
// of your Holy Light on them by up to $s1, your Flash of Light by up to $s2,
// and your Holy Shock by up to $s3." The push's effects scale with the
// caster's level only (6.6667, 1.9167 and 3.8333 a level: 400, 115 and 230
// at 60). The whole amount lands on a rank near the caster's level, with
// no cast-time coefficient, as Blessing of Light's "up to" amounts did on
// Blizzard's server: Flash of Light rank 1 took the full +60 in Classic
// Era (us.forums.blizzard.com/en/wow/t/blessing-of-light-bug/297989,
// 2019) and warcraft.wiki.gg/wiki/Blessing_of_Light says the same.
func TestOathOfLightRaisesTheSwornAllysHeals(t *testing.T) {
	cases := []struct {
		name         string
		cast, healed int32
		bonus        float64
	}{
		{"Holy Light", holyLightTopRank, holyLightTopRank, 400},
		{"Flash of Light", flashOfLightTopRank, flashOfLightTopRank, 115},
		{"Holy Shock", holyShockTopCast, holyShockTopHeal, 230},
	}
	for _, c := range cases {
		f := numericFight(rotationOf(whileBefore("1s", castItem(oathOfLightSpell, tankIndex)), castItem(c.cast, tankIndex)))
		f.talents = map[string]int{"oath_of_light": 1, "holy_shock": 1}
		want := healFor(t, c.healed) + c.bonus
		if got := totalsOf(f.run(t, numericRuns), c.healed).normalHeal(); !within(got, want, averageTolerance) {
			t.Errorf("%s on the sworn tank heals %.1f, want %.1f", c.name, got, want)
		}
	}
}

// healAt is the normal heal of a cast rank at the given level: the client
// roll's middle plus the coefficient the rank keeps at that level times the
// test's healing power (core.SpellLevelPenalty, as assertHealNumbers).
func healAt(t *testing.T, castID int32, level int32) float64 {
	t.Helper()
	spell := clientSpell(t, castID)
	low, high, _ := spell.DamageRange(0, int(level))
	share := core.SpellLevelPenalty(castID, spell.SpellLevel, int(level))
	return (low+high)/2 + spell.Effects[0].ResolvedSPCoefficient*share*healingPower
}

// The Oath's amount follows the rank's own level rule: a rank learned far
// below the caster loses it as it loses healing power. That is how
// Blizzard's server treated Blessing of Light's amount: Holy Light rank 1
// took 60 of its 210 in Classic Era, the level-1 rank's sub-20 share
// (0.2875 x 210; the thread above), and patch 2.3.0's note "Lower ranks of
// Flash of Light and Holy Light are now properly penalized when used with
// this Blessing" is the same code. Forever's rule is MaxLevel + 12, then
// 5% a level (core.SpellLevelPenalty).
func TestOathOfLightFollowsTheRanksLevelRule(t *testing.T) {
	const level = 50
	cases := []struct {
		name  string
		cast  int32
		share float64
	}{
		{"Holy Light rank 1 (MaxLevel 5)", 635, 0},
		{"Holy Light rank 5 (level 30)", 1042, 0.85},
		{"Holy Light rank 7 (level 46)", 10328, 1},
	}
	for _, c := range cases {
		f := numericFight(rotationOf(whileBefore("1s", castItem(oathOfLightSpell, tankIndex)), castItem(c.cast, tankIndex)))
		f.level = level
		f.talents = map[string]int{"oath_of_light": 1}
		want := healAt(t, c.cast, level) + 6.6667*level*c.share
		if got := totalsOf(f.run(t, numericRuns), c.cast).normalHeal(); !within(got, want, averageTolerance) {
			t.Errorf("%s on the sworn tank at %d heals %.1f, want %.1f", c.name, level, got, want)
		}
	}
}

// Only the sworn ally gains: a party member's Flash of Light is unchanged.
func TestOathOfLightLeavesOtherAlliesAlone(t *testing.T) {
	member := int(healsim.MemberIndices[0])
	f := numericFight(rotationOf(whileBefore("1s", castItem(oathOfLightSpell, tankIndex)), castItem(flashOfLightTopRank, member)))
	f.talents = map[string]int{"oath_of_light": 1}
	if got, want := totalsOf(f.run(t, numericRuns), flashOfLightTopRank).normalHeal(), healFor(t, flashOfLightTopRank); !within(got, want, averageTolerance) {
		t.Errorf("Flash of Light on an unsworn member heals %.1f, want %.1f", got, want)
	}
}

// Without the talent there is no Oath to swear.
func TestOathOfLightNeedsItsTalent(t *testing.T) {
	f := numericFight(rotationOf(whileBefore("1s", castItem(oathOfLightSpell, tankIndex)), castItem(flashOfLightTopRank, tankIndex)))
	if casts := totalsOf(f.run(t, numericRuns), oathOfLightSpell).casts; casts != 0 {
		t.Errorf("Oath of Light was cast %d times without its talent", casts)
	}
}
