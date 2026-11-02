package restoration

import (
	"math"
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
	"github.com/wowsims/classic/sim/druid"
)

// naturesSwiftnessID is the talent spell, which is also its aura.
const naturesSwiftnessID = 17116

// healer builds a level 60 restoration druid with the given talent ranks
// (proto field names), Swiftmend and Wild Growth included so every healing
// spell is there to inspect.
func healer(t *testing.T, ranks map[string]int) (*RestorationDruid, *core.Simulation) {
	t.Helper()
	all := map[string]int{"swiftmend": 1, "wild_growth": 1}
	for name, rank := range ranks {
		all[name] = rank
	}
	return newDruid(t, newPlayer(60, talentsString(t, all), healerBonusStats, nil))
}

// healingSpells is the top rank of every healing spell, by name.
func healingSpells(resto *RestorationDruid) map[string]*druid.DruidSpell {
	return map[string]*druid.DruidSpell{
		"Healing Touch": resto.HealingTouch[core.MaxTrainerRank(11)],
		"Regrowth":      resto.Regrowth[9],
		"Rejuvenation":  resto.Rejuvenation[core.MaxTrainerRank(11)],
		"Swiftmend":     resto.Swiftmend,
		"Tranquility":   resto.Tranquility[4],
		"Wild Growth":   resto.WildGrowth[3],
	}
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestGiftOfNatureRaisesEveryHealingSpell(t *testing.T) {
	resto, _ := healer(t, map[string]int{"gift_of_nature": 5})
	for name, spell := range healingSpells(resto) {
		if !near(spell.DamageMultiplierAdditive, 1.10) {
			t.Errorf("%s multiplier = %v with 5/5 Gift of Nature, want 1.10", name, spell.DamageMultiplierAdditive)
		}
	}
}

func TestGiftOfNatureShowsInTheHeal(t *testing.T) {
	plain, plainSim := healer(t, nil)
	gifted, giftedSim := healer(t, map[string]int{"gift_of_nature": 5})
	var plainTotal, giftedTotal float64
	for i := 0; i < castsPerRank; i++ {
		plainTotal += castHeal(plainSim, plain.HealingTouch[10], &plain.Unit).amount
		giftedTotal += castHeal(giftedSim, gifted.HealingTouch[10], &gifted.Unit).amount
	}
	if ratio := giftedTotal / plainTotal; math.Abs(ratio-1.10) > 0.05 {
		t.Errorf("5/5 Gift of Nature healed %.3f times as much, want about 1.10", ratio)
	}
}

func TestImprovedRejuvenationAddsToGiftOfNatureOnRejuvenationOnly(t *testing.T) {
	resto, _ := healer(t, map[string]int{"improved_rejuvenation": 3, "gift_of_nature": 5})
	spells := healingSpells(resto)
	if got := spells["Rejuvenation"].DamageMultiplierAdditive; !near(got, 1.25) {
		t.Errorf("Rejuvenation multiplier = %v, want 1 + 0.10 + 0.15", got)
	}
	if got := spells["Regrowth"].DamageMultiplierAdditive; !near(got, 1.10) {
		t.Errorf("Regrowth multiplier = %v, want 1.10", got)
	}
}

func TestTranquilSpiritCutsHealingTouchAndTranquilityMana(t *testing.T) {
	plain, _ := healer(t, nil)
	talented, _ := healer(t, map[string]int{"tranquil_spirit": 5})
	baseSpells, talentedSpells := healingSpells(plain), healingSpells(talented)
	for name, spell := range talentedSpells {
		want := baseSpells[name].Cost.GetCurrentCost()
		if name == "Healing Touch" || name == "Tranquility" {
			want *= 0.90
		}
		if got := spell.Cost.GetCurrentCost(); math.Abs(got-want) > 1e-6 {
			t.Errorf("%s costs %v with 5/5 Tranquil Spirit, want %v", name, got, want)
		}
	}
}

func TestImprovedTranquilityCutsItsCooldown(t *testing.T) {
	for rank, want := range map[int]time.Duration{0: 5 * time.Minute, 1: 210 * time.Second, 2: 120 * time.Second} {
		resto, _ := healer(t, map[string]int{"improved_tranquility": rank})
		if got := resto.Tranquility[4].CD.Duration; got != want {
			t.Errorf("Tranquility cooldown with %d/2 Improved Tranquility = %v, want %v", rank, got, want)
		}
	}
}

func TestImprovedRegrowthAddsCritToRegrowthOnly(t *testing.T) {
	plain, _ := healer(t, nil)
	talented, _ := healer(t, map[string]int{"improved_regrowth": 5})
	baseSpells, talentedSpells := healingSpells(plain), healingSpells(talented)
	for name, spell := range talentedSpells {
		want := baseSpells[name].BonusCritRating
		if name == "Regrowth" {
			want += 50 * core.CritRatingPerCritChance
		}
		if !near(spell.BonusCritRating, want) {
			t.Errorf("%s bonus crit = %v with 5/5 Improved Regrowth, want %v", name, spell.BonusCritRating, want)
		}
	}
}

func TestNaturalistQuickensHealingTouchAndRaisesDamage(t *testing.T) {
	plain, _ := healer(t, nil)
	talented, _ := healer(t, map[string]int{"naturalist": 5})
	for rank := 1; rank < len(plain.HealingTouch); rank++ {
		want := plain.HealingTouch[rank].DefaultCast.CastTime - 500*time.Millisecond
		if got := talented.HealingTouch[rank].DefaultCast.CastTime; got != want {
			t.Errorf("Healing Touch rank %d casts in %v with 5/5 Naturalist, want %v", rank, got, want)
		}
	}
	if got := talented.Regrowth[9].DefaultCast.CastTime; got != 2*time.Second {
		t.Errorf("Regrowth casts in %v with Naturalist, want 2s", got)
	}
	// Client 1.60.1.70707: Wild Growth has a 3 s cast and Naturalist cuts it.
	if got := plain.WildGrowth[3].DefaultCast.CastTime; got != 3*time.Second {
		t.Errorf("Wild Growth casts in %v, want 3s", got)
	}
	if got := talented.WildGrowth[3].DefaultCast.CastTime; got != 2500*time.Millisecond {
		t.Errorf("Wild Growth casts in %v with 5/5 Naturalist, want 2.5s", got)
	}
	if got := talented.PseudoStats.DamageDealtMultiplier; !near(got, 1.10) {
		t.Errorf("damage multiplier = %v with 5/5 Naturalist, want 1.10", got)
	}
}

// Genesis (1223081) effect 0 is aura 108, misc 22 (periodic healing and
// damage), SpellClassMask_0 0x220252 in 1.60.1.70707: Moonfire,
// Rejuvenation, Regrowth, Pounce and Insect Swarm, with Wild Growth and
// Lacerate in mask_1 and Rip in mask_2. Tranquility (0x80, SpellClassOptions
// 9863) is not in it.
func TestGenesisAddsToTheHealsOverTimeItsMaskNames(t *testing.T) {
	resto, _ := healer(t, map[string]int{"genesis": 5})
	for name, spell := range healingSpells(resto) {
		want := 1.0
		if name != "Healing Touch" && name != "Swiftmend" && name != "Tranquility" {
			want = 1.10
		}
		if !near(spell.PeriodicDamageMultiplierAdditive, want) {
			t.Errorf("%s periodic multiplier = %v with 5/5 Genesis, want %v", name, spell.PeriodicDamageMultiplierAdditive, want)
		}
	}
}

func TestGenesisShowsInTheTicks(t *testing.T) {
	resto, sim := healer(t, map[string]int{"genesis": 5})
	rejuvenation := resto.Rejuvenation[10]
	rejuvenation.ApplyEffects(sim, &resto.Unit, rejuvenation.Spell)
	dot := rejuvenation.Hot(&resto.Unit)
	want := (161 + 0.2*healerBonusStats[stats.HealingPower]) * 1.10
	if got := dot.SnapshotBaseDamage * dot.SnapshotAttackerMultiplier; math.Abs(got-want) > tolerance {
		t.Errorf("Rejuvenation ticks for %v with 5/5 Genesis, want %v", got, want)
	}
}

func TestNaturesSplendorLengthensRejuvenationAndRegrowth(t *testing.T) {
	plain, _ := healer(t, nil)
	talented, _ := healer(t, map[string]int{"natures_splendor": 1})
	cases := []struct {
		name       string
		plain      *druid.DruidSpell
		spell      *druid.DruidSpell
		extraTicks int32
	}{
		{"Rejuvenation", plain.Rejuvenation[10], talented.Rejuvenation[10], 1},
		{"Regrowth", plain.Regrowth[9], talented.Regrowth[9], 2},
	}
	for _, c := range cases {
		base, got := c.plain.Hot(&plain.Unit), c.spell.Hot(&talented.Unit)
		if got.NumberOfTicks != base.NumberOfTicks+c.extraTicks {
			t.Errorf("%s ticks = %d with Nature's Splendor, want %d", c.name, got.NumberOfTicks, base.NumberOfTicks+c.extraTicks)
		}
		if want := time.Duration(got.NumberOfTicks) * 3 * time.Second; got.Duration != want {
			t.Errorf("%s lasts %v, want %v", c.name, got.Duration, want)
		}
	}
}

func TestReflectionKeepsAShareOfRegenerationWhileCasting(t *testing.T) {
	for rank, want := range map[int]float64{0: 0, 1: 1.0 / 6, 2: 1.0 / 3, 3: 0.5} {
		resto, _ := healer(t, map[string]int{"reflection": rank})
		if got := resto.PseudoStats.SpiritRegenRateCasting; !near(got, want) {
			t.Errorf("casting regeneration with %d/3 Reflection = %v, want %v", rank, got, want)
		}
	}
}

func TestLivingSpiritRaisesSpirit(t *testing.T) {
	plain, _ := healer(t, nil)
	talented, _ := healer(t, map[string]int{"living_spirit": 3})
	if got, want := talented.GetStat(stats.Spirit), plain.GetStat(stats.Spirit)*1.15; math.Abs(got-want) > 1e-6 {
		t.Errorf("spirit = %v with 3/3 Living Spirit, want %v", got, want)
	}
}

func TestNaturesSwiftnessMakesTheNextHealInstantOnce(t *testing.T) {
	resto, sim := healer(t, map[string]int{"natures_swiftness": 1})
	healingTouch, regrowth, rejuvenation := resto.HealingTouch[10], resto.Regrowth[9], resto.Rejuvenation[10]
	slowTouch, slowRegrowth := healingTouch.CastTime(), regrowth.CastTime()
	if slowTouch == 0 || slowRegrowth == 0 {
		t.Fatalf("Healing Touch (%v) and Regrowth (%v) should have cast times", slowTouch, slowRegrowth)
	}

	resto.NaturesSwiftness.ApplyEffects(sim, &resto.Unit, resto.NaturesSwiftness.Spell)
	if !resto.HasActiveAura("Natures Swiftness") {
		t.Fatal("Nature's Swiftness did not raise its aura")
	}
	if got := healingTouch.CastTime(); got != 0 {
		t.Errorf("Healing Touch casts in %v under Nature's Swiftness, want instant", got)
	}
	if got := regrowth.CastTime(); got != 0 {
		t.Errorf("Regrowth casts in %v under Nature's Swiftness, want instant", got)
	}

	resto.OnCastComplete(sim, rejuvenation.Spell)
	if !resto.HasActiveAura("Natures Swiftness") {
		t.Error("an instant Rejuvenation used up Nature's Swiftness")
	}

	resto.OnCastComplete(sim, healingTouch.Spell)
	if resto.HasActiveAura("Natures Swiftness") {
		t.Error("Nature's Swiftness survived the Healing Touch it was for")
	}
	if got := healingTouch.CastTime(); got != slowTouch {
		t.Errorf("Healing Touch casts in %v after Nature's Swiftness, want %v", got, slowTouch)
	}
	if got := resto.NaturesSwiftness.CD.Duration; got != 3*time.Minute {
		t.Errorf("Nature's Swiftness cooldown = %v, want 3 minutes", got)
	}
}

func TestNaturesSwiftnessWaitsForTheRotation(t *testing.T) {
	resto, sim := healer(t, map[string]int{"natures_swiftness": 1})
	mcd := resto.GetMajorCooldown(core.ActionID{SpellID: naturesSwiftnessID})
	if mcd == nil {
		t.Fatal("Nature's Swiftness is not a major cooldown")
	}
	if mcd.ShouldActivate(sim, &resto.Character) {
		t.Error("the generic cooldown pass would fire Nature's Swiftness for a healer")
	}
}

// Nature's Focus (node 104957, five ranks): "Gives your Arcane and Nature
// spells a 14/28/42/56/70% chance to not lose casting time when you take
// damage." The modifier's client mask names Healing Touch, Regrowth and
// Tranquility among the heals, not Wild Growth, Rejuvenation or Swiftmend.
func TestNaturesFocusKeepsNatureCastTimeUnderDamage(t *testing.T) {
	inMask := map[string]bool{"Healing Touch": true, "Regrowth": true, "Tranquility": true}
	for rank := 0; rank <= 5; rank++ {
		resto, _ := healer(t, map[string]int{"natures_focus": rank})
		for name, spell := range healingSpells(resto) {
			want := 0.0
			if inMask[name] {
				want = 0.14 * float64(rank)
			}
			if got := spell.PushbackReduction; !near(got, want) {
				t.Errorf("Nature's Focus %d/5: %s keeps its cast time %.2f of the time, want %.2f", rank, name, got, want)
			}
		}
	}
}

// Nature's Grace (16880) procs on "All non-periodic spell criticals": its
// ProcTypeMask 0x14000 takes helpful magic spells as well as harmful ones,
// so a critical Healing Touch graces the druid and a critical
// Rejuvenation tick does not. Its global cooldown cut (16886 effect 1,
// mask 0x7407f7/0x4000000) names the heals too.
func TestNaturesGraceProcsOnACriticalHealNotATick(t *testing.T) {
	resto, sim := healer(t, map[string]int{"natures_grace": 1})
	self := &resto.Unit
	touch, rejuvenation := resto.HealingTouch[10], resto.Rejuvenation[10]

	rejuvenation.BonusCritRating += 100 * core.CritRatingPerCritChance
	rejuvenation.ApplyEffects(sim, self, rejuvenation.Spell)
	runUntil(sim, 4*time.Second)
	if resto.NaturesGraceProcAura.IsActive() {
		t.Fatal("a critical Rejuvenation tick graced the druid")
	}

	touch.BonusCritRating += 100 * core.CritRatingPerCritChance
	touch.ApplyEffects(sim, self, touch.Spell)
	if !resto.NaturesGraceProcAura.IsActive() {
		t.Error("a critical Healing Touch did not grace the druid")
	}
	if got, want := touch.DefaultCast.GCD, core.GCDDefault-150*time.Millisecond; got != want {
		t.Errorf("Healing Touch's global cooldown under Nature's Grace = %v, want %v", got, want)
	}
}
