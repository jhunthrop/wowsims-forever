package holy

import (
	"testing"

	"github.com/wowsims/classic/sim/healsim"
)

const oathOfLightSpell int32 = 1328548

// Oath of Light (talent 1328548, a spell the 70707 hotfix push adds):
// "Swear an Oath to protect target ally for 5 min, increasing the healing
// of your Holy Light on them by up to $s1, your Flash of Light by up to $s2,
// and your Holy Shock by up to $s3." The push's effects scale with the
// caster's level only (6.6667, 1.9167 and 3.8333 a level: 400, 115 and 230
// at 60). The bonus counts as bonus healing for the one spell, scaled by
// its coefficient, as Blessing of Light's 400 and 115 were.
func TestOathOfLightRaisesTheSwornAllysHeals(t *testing.T) {
	cases := []struct {
		name               string
		cast, healed       int32
		bonus, coefficient float64
	}{
		{"Holy Light", holyLightTopRank, holyLightTopRank, 400, 0.714},
		{"Flash of Light", flashOfLightTopRank, flashOfLightTopRank, 115, 0.429},
		{"Holy Shock", holyShockTopCast, holyShockTopHeal, 230, 0.429},
	}
	for _, c := range cases {
		f := numericFight(rotationOf(whileBefore("1s", castItem(oathOfLightSpell, tankIndex)), castItem(c.cast, tankIndex)))
		f.talents = map[string]int{"oath_of_light": 1, "holy_shock": 1}
		want := healFor(t, c.healed) + c.bonus*c.coefficient
		if got := totalsOf(f.run(t, numericRuns), c.healed).normalHeal(); !within(got, want, averageTolerance) {
			t.Errorf("%s on the sworn tank heals %.1f, want %.1f", c.name, got, want)
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
