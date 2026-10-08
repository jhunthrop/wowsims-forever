package restoration

import (
	"math"
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/spellconst"
	"github.com/wowsims/classic/sim/core/stats"
	"github.com/wowsims/classic/sim/druid"
)

const (
	clientDruidSpellconst = "../../core/testdata/conformance/client/druid.json"
	// castsPerRank is how many heals each rank rolls: enough to see both ends
	// of the roll and a few critical strikes.
	castsPerRank = 600
	// tolerance covers the client's float32 coefficients (0.20000000298 for
	// 0.2) times a few hundred healing power.
	tolerance = 1e-3
)

// resto60 is a level 60 restoration druid with the standard build.
func resto60(t *testing.T) (*RestorationDruid, *core.Simulation) {
	t.Helper()
	return newDruid(t, newPlayer(60, StandardTalents, healerBonusStats, nil))
}

// healed is one heal as the spell's metrics saw it.
type healed struct {
	amount float64
	crit   bool
}

// castHeal runs the spell's effects once at the target and reports what the
// heal landed for.
func castHeal(sim *core.Simulation, spell *druid.DruidSpell, target *core.Unit) healed {
	metrics := &spell.SpellMetrics[target.UnitIndex]
	before, critsBefore := metrics.TotalHealing, metrics.Crits
	spell.ApplyEffects(sim, target, spell.Spell)
	return healed{amount: metrics.TotalHealing - before, crit: metrics.Crits > critsBefore}
}

// critMultiplier is how much a critical heal of the spell is worth.
func critMultiplier(spell *druid.DruidSpell, target *core.Unit) float64 {
	return spell.CritMultiplier(spell.Unit.AttackTables[target.UnitIndex][spell.CastType])
}

func clientSpell(t *testing.T, id int32) spellconst.Spell {
	t.Helper()
	class, err := spellconst.Load(clientDruidSpellconst)
	if err != nil {
		t.Fatal(err)
	}
	spell, ok := class.ByID(id)
	if !ok {
		t.Fatalf("spell %d is not in the client table", id)
	}
	return spell
}

// assertDirectHeals checks every landed heal of the spell is the client's
// roll for the effect plus coefficient times healing power (times the crit
// multiplier when it crit), and that the draws cover the roll's width.
func assertDirectHeals(t *testing.T, sim *core.Simulation, spell *druid.DruidSpell, target *core.Unit, effectIndex int) {
	t.Helper()
	client := clientSpell(t, spell.SpellID)
	low, high, ok := client.DamageRange(effectIndex, int(spell.Unit.Level))
	if !ok {
		t.Fatalf("%s rank %d: the client has no effect %d", spell.ActionID, spell.Rank, effectIndex)
	}
	coefficient := client.Effects[effectIndex].ResolvedSPCoefficient
	power := coefficient * spell.HealingPower(target)
	crit := critMultiplier(spell, target)

	seenLow, seenHigh, crits := math.Inf(1), math.Inf(-1), 0
	for i := 0; i < castsPerRank; i++ {
		heal := castHeal(sim, spell, target)
		base := heal.amount
		if heal.crit {
			base /= crit
			crits++
		}
		if base < (low+power)-tolerance || base > (high+power)+tolerance {
			t.Fatalf("%s rank %d healed %v (crit %v), want within %v-%v", spell.ActionID, spell.Rank, heal.amount, heal.crit, (low + power), (high + power))
		}
		seenLow, seenHigh = math.Min(seenLow, base), math.Max(seenHigh, base)
	}
	if width := high - low; width > 1 && (seenHigh-seenLow) < width/2 {
		t.Errorf("%s rank %d drew only %v-%v of the roll %v-%v", spell.ActionID, spell.Rank, seenLow, seenHigh, (low + power), (high + power))
	}
	if crits == 0 || crits == castsPerRank {
		t.Errorf("%s rank %d crit %d of %d times", spell.ActionID, spell.Rank, crits, castsPerRank)
	}
}

func TestHealingTouchRanksHealForTheClientRoll(t *testing.T) {
	resto, sim := newDruid(t, newPlayer(60, "", healerBonusStats, nil))
	if got, want := len(resto.HealingTouch)-1, core.MaxTrainerRank(11); got != want {
		t.Fatalf("Healing Touch has %d ranks, want %d (rank 11 is the Ahn'Qiraj book)", got, want)
	}
	for rank := 1; rank < len(resto.HealingTouch); rank++ {
		assertDirectHeals(t, sim, resto.HealingTouch[rank], &resto.Unit, 0)
	}
}

func TestRegrowthRanksHealForTheClientRoll(t *testing.T) {
	resto, sim := newDruid(t, newPlayer(60, "", healerBonusStats, nil))
	if got, want := len(resto.Regrowth)-1, 9; got != want {
		t.Fatalf("Regrowth has %d ranks, want %d", got, want)
	}
	for rank := 1; rank < len(resto.Regrowth); rank++ {
		assertDirectHeals(t, sim, resto.Regrowth[rank], &resto.Unit, 0)
	}
}

// assertHealOverTime checks the spell's periodic heal against the client's
// periodic-heal effect: the tick count and period from its duration, and the
// tick (the roll plus coefficient times healing power) every tick lands for.
func assertHealOverTime(t *testing.T, sim *core.Simulation, spell *druid.DruidSpell, target *core.Unit, effectIndex int) {
	t.Helper()
	client := clientSpell(t, spell.SpellID)
	effect := client.Effects[effectIndex]
	spell.ApplyEffects(sim, target, spell.Spell)
	dot := spell.Hot(target)
	if !dot.IsActive() {
		t.Fatalf("%s rank %d left no heal over time", spell.ActionID, spell.Rank)
	}

	period := time.Duration(effect.PeriodMS) * time.Millisecond
	duration := time.Duration(client.DurationMS) * time.Millisecond
	if dot.TickLength != period || time.Duration(dot.NumberOfTicks)*period != duration {
		t.Errorf("%s rank %d ticks %d x %v, want %v in all at %v", spell.ActionID, spell.Rank, dot.NumberOfTicks, dot.TickLength, duration, period)
	}

	low, _, _ := client.DamageRange(effectIndex, int(spell.Unit.Level))
	want := low + effect.ResolvedSPCoefficient*spell.HealingPower(target)
	if got := dot.SnapshotBaseDamage * dot.SnapshotAttackerMultiplier; math.Abs(got-want) > tolerance {
		t.Errorf("%s rank %d ticks for %v, want %v", spell.ActionID, spell.Rank, got, want)
	}
	dot.Cancel(sim)
}

func TestRejuvenationRanksTickForTheClientAmount(t *testing.T) {
	resto, sim := newDruid(t, newPlayer(60, "", healerBonusStats, nil))
	if got, want := len(resto.Rejuvenation)-1, core.MaxTrainerRank(11); got != want {
		t.Fatalf("Rejuvenation has %d ranks, want %d", got, want)
	}
	for rank := 1; rank < len(resto.Rejuvenation); rank++ {
		assertHealOverTime(t, sim, resto.Rejuvenation[rank], &resto.Unit, 0)
	}
}

func TestRegrowthHealOverTimeTicksForTheClientAmount(t *testing.T) {
	resto, sim := newDruid(t, newPlayer(60, "", healerBonusStats, nil))
	for rank := 1; rank < len(resto.Regrowth); rank++ {
		assertHealOverTime(t, sim, resto.Regrowth[rank], &resto.Unit, 1)
	}
}

func TestTranquilityAndWildGrowthTickForTheClientAmount(t *testing.T) {
	resto, sim := newDruid(t, newPlayer(60, talentsString(t, map[string]int{"wild_growth": 1}), healerBonusStats, nil))
	for rank := 1; rank < len(resto.Tranquility); rank++ {
		assertTranquilityTicks(t, resto, sim, resto.Tranquility[rank])
	}
	if got := len(resto.WildGrowth) - 1; got != 3 {
		t.Fatalf("Wild Growth has %d ranks, want 3", got)
	}
	for rank := 1; rank < len(resto.WildGrowth); rank++ {
		assertHealOverTime(t, sim, resto.WildGrowth[rank], &resto.Unit, 0)
	}
}

func assertTranquilityTicks(t *testing.T, resto *RestorationDruid, sim *core.Simulation, spell *druid.DruidSpell) {
	t.Helper()
	assertHealOverTime(t, sim, spell, &resto.Unit, 0)
	if !spell.Flags.Matches(core.SpellFlagChanneled) {
		t.Errorf("%s rank %d is not a channel", spell.ActionID, spell.Rank)
	}
}

// TestLevelAwareRanks: a character only registers the ranks it has learned
// and the talent spells only with their talents.
func TestLevelAwareRanks(t *testing.T) {
	talents := talentsString(t, map[string]int{"swiftmend": 1, "wild_growth": 1})
	cases := []struct {
		level                                int32
		healingTouch, regrowth, rejuvenation int
		tranquility, wildGrowth              int
		wantSwiftmend                        bool
	}{
		{level: 1, healingTouch: 1, wantSwiftmend: true},
		{level: 8, healingTouch: 2, rejuvenation: 1, wantSwiftmend: true},
		{level: 30, healingTouch: 5, regrowth: 4, rejuvenation: 5, tranquility: 1, wantSwiftmend: true},
		{level: 40, healingTouch: 7, regrowth: 5, rejuvenation: 7, tranquility: 2, wildGrowth: 1, wantSwiftmend: true},
		{level: 59, healingTouch: 10, regrowth: 8, rejuvenation: 10, tranquility: 3, wildGrowth: 2, wantSwiftmend: true},
		{level: 60, healingTouch: 10, regrowth: 9, rejuvenation: 10, tranquility: 4, wildGrowth: 3, wantSwiftmend: true},
	}
	for _, c := range cases {
		resto, _ := newDruid(t, newPlayer(c.level, talents, stats.Stats{}, nil))
		learned := func(ranks []*druid.DruidSpell) int {
			count := 0
			for _, spell := range ranks {
				if spell != nil {
					count++
				}
			}
			return count
		}
		got := map[string]int{
			"Healing Touch": learned(resto.HealingTouch), "Regrowth": learned(resto.Regrowth),
			"Rejuvenation": learned(resto.Rejuvenation), "Tranquility": learned(resto.Tranquility),
			"Wild Growth": learned(resto.WildGrowth),
		}
		want := map[string]int{
			"Healing Touch": c.healingTouch, "Regrowth": c.regrowth,
			"Rejuvenation": c.rejuvenation, "Tranquility": c.tranquility,
			"Wild Growth": c.wildGrowth,
		}
		for name, count := range want {
			if got[name] != count {
				t.Errorf("level %d: %s has %d ranks, want %d", c.level, name, got[name], count)
			}
		}
		if (resto.Swiftmend != nil) != c.wantSwiftmend {
			t.Errorf("level %d: Swiftmend registered = %v, want %v", c.level, resto.Swiftmend != nil, c.wantSwiftmend)
		}
	}
}

func TestSwiftmendAndWildGrowthNeedTheirTalents(t *testing.T) {
	resto, _ := newDruid(t, newPlayer(60, "", healerBonusStats, nil))
	if resto.Swiftmend != nil {
		t.Error("Swiftmend registered without its talent")
	}
	for _, spell := range resto.WildGrowth {
		if spell != nil {
			t.Error("Wild Growth registered without its talent")
		}
	}
}

// TestOnlyTheTrainedIdsRegister: the client carries a second, unlearnable id
// for every Regrowth and Rejuvenation rank; only the trainer's registers.
func TestOnlyTheTrainedIdsRegister(t *testing.T) {
	resto, _ := resto60(t)
	reissued := []int32{436937, 436946, 417057, 417068}
	for _, id := range reissued {
		if resto.GetSpell(core.ActionID{SpellID: id}) != nil {
			t.Errorf("spell %d, a reissue no trainer teaches, is registered", id)
		}
	}
	for _, id := range []int32{8936, 9858, 774, 9841} {
		if resto.GetSpell(core.ActionID{SpellID: id}) == nil {
			t.Errorf("trainer spell %d is not registered", id)
		}
	}
}

func TestSwiftmendNeedsAnEffectAndEatsIt(t *testing.T) {
	resto, sim := resto60(t)
	self := &resto.Unit
	rejuvenation, regrowth := resto.Rejuvenation[10], resto.Regrowth[9]

	if resto.Swiftmend.CanCast(sim, self) {
		t.Fatal("Swiftmend can be cast on a target with no Rejuvenation or Regrowth")
	}

	rejuvenation.ApplyEffects(sim, self, rejuvenation.Spell)
	dot := rejuvenation.Hot(self)
	wantHeal := dot.SnapshotBaseDamage * dot.SnapshotAttackerMultiplier * float64(dot.NumberOfTicks)
	if !resto.Swiftmend.CanCast(sim, self) {
		t.Fatal("Swiftmend cannot be cast on a target with Rejuvenation")
	}

	heal := castHeal(sim, resto.Swiftmend, self)
	if heal.crit {
		wantHeal *= critMultiplier(resto.Swiftmend, self)
	}
	if math.Abs(heal.amount-wantHeal) > tolerance {
		t.Errorf("Swiftmend healed %v, want the whole of the Rejuvenation, %v", heal.amount, wantHeal)
	}
	if dot.IsActive() {
		t.Error("Swiftmend left the Rejuvenation up")
	}

	regrowth.ApplyEffects(sim, self, regrowth.Spell)
	regrowthDot := regrowth.Hot(self)
	castHeal(sim, resto.Swiftmend, self)
	if regrowthDot.IsActive() {
		t.Error("Swiftmend left the Regrowth up")
	}
}

func TestSwiftmendEatsTheEffectWithTheLeastTimeLeft(t *testing.T) {
	resto, sim := resto60(t)
	self := &resto.Unit
	rejuvenation, regrowth := resto.Rejuvenation[10], resto.Regrowth[9]
	regrowth.ApplyEffects(sim, self, regrowth.Spell)
	rejuvenation.ApplyEffects(sim, self, rejuvenation.Spell)

	castHeal(sim, resto.Swiftmend, self)
	if rejuvenation.Hot(self).IsActive() || !regrowth.Hot(self).IsActive() {
		t.Error("Swiftmend ate the longer Regrowth instead of the Rejuvenation about to end")
	}
}

func TestSwiftmendCostsAFifthOfBaseManaAndWaitsFifteenSeconds(t *testing.T) {
	resto, _ := resto60(t)
	if got, want := resto.Swiftmend.Cost.GetCurrentCost(), 0.2*resto.BaseMana; math.Abs(got-want) > tolerance {
		t.Errorf("Swiftmend costs %v, want 20%% of base mana, %v", got, want)
	}
	if got := resto.Swiftmend.CD.Duration; got != 15*time.Second {
		t.Errorf("Swiftmend cooldown = %v, want 15s", got)
	}
}
