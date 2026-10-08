package warlock

import (
	"testing"

	"github.com/wowsims/classic/sim/common/clientdamage/clientdamagetest"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
)

// Life Tap's amount is the client's effect 0 (a dummy effect, parked
// flat) and, per the spell text "Converts ${($m1+$SPI*1)*(1+$18182m1/100)}
// Health into the same Mana ... Spirit increases the amount converted",
// one point of Spirit on top of it.
func TestLifeTapAmountTableMatchesTheClient(t *testing.T) {
	class := clientdamagetest.Load(t, clientWarlockSpellconst)
	clientdamagetest.AssertTable(t, class, clientdamagetest.Dummy, "Life Tap",
		LifeTapSpellId[:], LifeTapAmount[:], nil)
}

// lifeTapMana is the mana one cast of the top rank restores.
func lifeTapMana(t *testing.T, talents string, extraSpellPower, extraSpirit float64) (float64, float64) {
	t.Helper()
	sim, built, target := newWarlockForDamageTest(t, talents)
	built.AddStatDynamic(sim, stats.SpellPower, extraSpellPower)
	built.AddStatDynamic(sim, stats.Spirit, extraSpirit)

	spendMetrics := built.NewManaMetrics(core.ActionID{SpellID: 1})
	built.SpendMana(sim, built.CurrentMana(), spendMetrics)
	before := built.CurrentMana()

	tap := built.LifeTap[len(built.LifeTap)-1]
	tap.ApplyEffects(sim, target, tap)
	return built.CurrentMana() - before, built.GetStat(stats.Spirit)
}

func TestLifeTapRestoresTheRankAmountPlusSpirit(t *testing.T) {
	got, spirit := lifeTapMana(t, "", 0, 0)
	// Rank 6 at level 60: 420 at the spell's level 56, plus 1 a level.
	const rankSixAtLevelSixty = 424.0
	if want := rankSixAtLevelSixty + spirit; got != want {
		t.Errorf("Life Tap restored %v mana, want %v (424 + %v Spirit)", got, want, spirit)
	}
}

func TestLifeTapScalesWithSpiritNotSpellPower(t *testing.T) {
	base, baseSpirit := lifeTapMana(t, "", 0, 0)
	moreSpellPower, _ := lifeTapMana(t, "", 500, 0)
	if moreSpellPower != base {
		t.Errorf("500 more spell power changed Life Tap from %v to %v mana", base, moreSpellPower)
	}
	// Added Spirit passes through the character's Spirit multipliers, so
	// the expected change is the change in the Spirit stat itself.
	moreSpirit, spirit := lifeTapMana(t, "", 0, 100)
	if got, want := moreSpirit-base, spirit-baseSpirit; !floatsNearlyEqual(got, want) {
		t.Errorf("%v more Spirit changed Life Tap by %v mana, want %v", want, got, want)
	}
}

// Improved Life Tap: "Increases the amount of Mana awarded ... by
// 10%/20%".
func TestImprovedLifeTapRaisesTheWholeAmount(t *testing.T) {
	base, _ := lifeTapMana(t, "", 0, 0)
	for rank, want := range map[string]float64{"1": 1.10, "2": 1.20} {
		got, _ := lifeTapMana(t, rank, 0, 0)
		if ratio := got / base; ratio < want-1e-6 || ratio > want+1e-6 {
			t.Errorf("Improved Life Tap %s: x%.4f, want x%.2f", rank, ratio, want)
		}
	}
}
