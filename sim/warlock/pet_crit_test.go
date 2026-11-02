package warlock

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

// Forever demons crit at their own, measured rates (forever-bugs #331 and the
// foreverlogs.gg beta logs), not at a warrior's or mage's attribute rates.
// The sheets' Agility and Intellect differ by level, so a crit that scaled
// with either would not hold at every band.
//   - melee: 12 of 1,718 Succubus swings (0.70%) by a level-30 warlock on
//     level-30 Cave Yetis (#331, Fierywind, 2026-10-12), and Blizzard's reply
//     there, "Their melee crit is extremely low"; Agility does not raise it;
//   - Lash of Pain: 13 of 205 in the same test and 7 of 105 in the public logs
//     of warlocks without Malevolence, 6.4%; Blizzard: "their base spell crit
//     is around normal levels";
//   - Firebolt: 53 of 2,383 (2.2%) in 17 public logs, levels 12 to 30.
func TestDemonCritMatchesTheForeverLogs(t *testing.T) {
	for _, level := range []int32{20, 30, 40, 60} {
		warlock := newWarlockAtLevel(t, level)
		for _, pet := range warlock.BasePets {
			s := pet.GetStats()
			melee := (s[stats.Crit] + s[stats.MeleeCrit]) / core.CritRatingPerCritChance
			spell := (s[stats.Crit] + s[stats.SpellCrit]) / core.CritRatingPerCritChance
			if pet != warlock.Imp && math.Abs(melee-0.7) > 1e-9 {
				t.Errorf("level %d %s melee crit = %.2f%%, want 0.70%%", level, pet.Name, melee)
			}
			wantSpell := map[*WarlockPet]float64{warlock.Imp: 2.2, warlock.Succubus: 6.4}[pet]
			if wantSpell != 0 && math.Abs(spell-wantSpell) > 1e-9 {
				t.Errorf("level %d %s spell crit = %.2f%%, want %.2f%%", level, pet.Name, spell, wantSpell)
			}
		}
	}
}
