package restoration

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
	"github.com/wowsims/classic/sim/druid"
)

// runUntil steps the simulation to the given time.
func runUntil(sim *core.Simulation, until time.Duration) {
	for sim.CurrentTime < until {
		if sim.Step() {
			return
		}
	}
}

// bloomHealing is what Bloom has healed the druid for.
func bloomHealing(resto *RestorationDruid) float64 {
	if resto.Bloom == nil {
		return 0
	}
	return resto.Bloom.SpellMetrics[resto.UnitIndex].TotalHealing
}

// Bloom (client 1.60.1.70707, spell 1328621): "When your Rejuvenation or
// Regrowth spell expires due to its duration, the target is instantly healed
// for an amount equal to 100% of your Spirit."
func TestBloomHealsForSpiritWhenRejuvenationRunsOut(t *testing.T) {
	for _, c := range []struct {
		name  string
		ranks map[string]int
		bloom bool
	}{
		{"untalented", nil, false},
		{"Bloom", map[string]int{"bloom": 1}, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			resto, sim := healer(t, c.ranks)
			rejuvenation := resto.Rejuvenation[10]
			rejuvenation.ApplyEffects(sim, &resto.Unit, rejuvenation.Spell)
			spirit := resto.GetStat(stats.Spirit)
			runUntil(sim, 13*time.Second)

			got := bloomHealing(resto)
			if !c.bloom {
				if got != 0 {
					t.Errorf("Bloom healed %v without the talent", got)
				}
				return
			}
			if !near(got, spirit) {
				t.Errorf("Bloom healed %v when Rejuvenation ran out, want the druid's Spirit %v", got, spirit)
			}
		})
	}
}

// Swiftmend leaves the Rejuvenation it reads in place, so the effect still
// runs its full duration and still Blooms.
func TestBloomStillFollowsASwiftmendedRejuvenation(t *testing.T) {
	resto, sim := healer(t, map[string]int{"bloom": 1})
	rejuvenation := resto.Rejuvenation[10]
	rejuvenation.ApplyEffects(sim, &resto.Unit, rejuvenation.Spell)
	spirit := resto.GetStat(stats.Spirit)
	runUntil(sim, 3*time.Second)
	resto.Swiftmend.ApplyEffects(sim, &resto.Unit, resto.Swiftmend.Spell)
	runUntil(sim, 13*time.Second)
	if got := bloomHealing(resto); !near(got, spirit) {
		t.Errorf("Bloom healed %v after a Swiftmend, want the druid's Spirit %v", got, spirit)
	}
}

// Bloom's heal (1328622) is a triggered spell, not a cast: the round 3
// Omen of Clarity rate (4% a spell) was fitted per logged cast, and set
// bonuses such as Vestments of the Devout's "your spellcasts have a 5%
// chance" read a cast. A Bloom must therefore fire neither the cast
// callbacks nor the healing proc mask those procs listen on.
func TestBloomTriggersNoCastProcs(t *testing.T) {
	resto, _ := healer(t, map[string]int{"bloom": 1})
	if !resto.Bloom.Flags.Matches(core.SpellFlagNoOnCastComplete) {
		t.Error("Bloom fires OnCastComplete, so cast procs (Devout's mana) roll on it")
	}
	if resto.Bloom.ProcMask.Matches(core.ProcMaskSpellHealing | core.ProcMaskSpellDamage) {
		t.Errorf("Bloom's proc mask %v rolls spell procs (Omen of Clarity)", resto.Bloom.ProcMask)
	}
}

// Bloom's heal (1328622) can critically strike: its SpellMisc row lacks
// SPELL_ATTR2_CANT_CRIT (Attributes_2 0x20000000), which the client's
// triggered heals that cannot crit carry (Judgement of Light 20267,
// Living Seed 414683).
func TestBloomCanCrit(t *testing.T) {
	resto, sim := healer(t, map[string]int{"bloom": 1})
	resto.Bloom.BonusCritRating += 200 * core.CritRatingPerCritChance
	resto.Bloom.ApplyEffects(sim, &resto.Unit, resto.Bloom.Spell)
	if got := resto.Bloom.SpellMetrics[resto.UnitIndex].TotalCritHealing; got <= 0 {
		t.Error("Bloom at 200% crit chance did not crit")
	}
}

// Blizzard's 1 October 2026 beta note names Omen of Clarity among the class
// abilities a rank "vastly below your current level" activates less: a
// Rejuvenation rank 1 at 60 (level-gap share 0, core.Spell.LevelPenaltyShare)
// never grants Clearcasting, while the top rank still does.
func TestOmenOfClarityNeverProcsFromARankOneRejuvenationAtSixty(t *testing.T) {
	resto, sim := healer(t, nil)
	rankOne := resto.Rejuvenation[1]
	if share := rankOne.LevelPenaltyShare(); share != 0 {
		t.Fatalf("rank 1 Rejuvenation at 60 keeps share %v, want 0", share)
	}
	// Omen rolls a heal over time once a cast (OnCastComplete), so the test
	// casts, a global apart, with the mana topped up.
	manaMetrics := resto.NewManaMetrics(core.ActionID{SpellID: 1})
	cast := func(spell *druid.DruidSpell) {
		// A global apart, with the global, the running Rejuvenation and the
		// mana reset by hand: the fixture's fight is too short to step through
		// hundreds of casts.
		sim.CurrentTime += core.GCDDefault
		resto.GCD.Reset()
		if hot := spell.Hot(&resto.Unit); hot != nil && hot.IsActive() {
			hot.Deactivate(sim)
		}
		resto.AddMana(sim, resto.MaxMana()-resto.CurrentMana(), manaMetrics)
		if !spell.Cast(sim, &resto.Unit) {
			t.Fatalf("%s did not cast at %v", spell.ActionID, sim.CurrentTime)
		}
	}
	for i := 0; i < 300; i++ {
		cast(rankOne)
		if resto.ClearcastingAura.IsActive() {
			t.Fatal("a rank 1 Rejuvenation granted Clearcasting at level 60")
		}
	}
	top := resto.Rejuvenation[core.MaxTrainerRank(11)]
	for i := 0; i < 600 && !resto.ClearcastingAura.IsActive(); i++ {
		cast(top)
	}
	if !resto.ClearcastingAura.IsActive() {
		t.Error("600 top-rank Rejuvenations never granted Clearcasting")
	}
}
