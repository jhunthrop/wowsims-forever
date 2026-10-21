package restoration

import (
	"math"
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/stats"
	"github.com/wowsims/classic/sim/healsim"
)

func TestImprovedHealingWaveShortensOnlyHealingWave(t *testing.T) {
	_, healer := newHealer(t, 60, restoTalents(map[int]int{posImprovedHealingWave: 5}))

	if got, want := healer.HealingWave[10].CastTime(), 2500*time.Millisecond; got != want {
		t.Errorf("Healing Wave casts in %v, want %v (3s less 0.1s a rank)", got, want)
	}
	if got, want := healer.LesserHealingWave[6].CastTime(), 1500*time.Millisecond; got != want {
		t.Errorf("Lesser Healing Wave casts in %v, want it untouched at %v", got, want)
	}
	if got, want := healer.ChainHeal[3].CastTime(), 2500*time.Millisecond; got != want {
		t.Errorf("Chain Heal casts in %v, want it untouched at %v", got, want)
	}
}

func TestTidalFocusCutsHealingManaCostAndAddsHit(t *testing.T) {
	_, base := newHealer(t, 60, "")
	_, healer := newHealer(t, 60, restoTalents(map[int]int{posTidalFocus: 5}))

	for name, pair := range map[string][2]*core.Spell{
		"Healing Wave":        {base.HealingWave[10], healer.HealingWave[10]},
		"Lesser Healing Wave": {base.LesserHealingWave[6], healer.LesserHealingWave[6]},
		"Chain Heal":          {base.ChainHeal[3], healer.ChainHeal[3]},
	} {
		want := pair[0].Cost.GetCurrentCost() * 0.95
		if got := pair[1].Cost.GetCurrentCost(); math.Abs(got-want) > 1e-6 {
			t.Errorf("%s costs %.2f, want %.2f (5 percent less)", name, got, want)
		}
	}
	if got, want := healer.HealingStreamTotem[5].Cost.GetCurrentCost(), base.HealingStreamTotem[5].Cost.GetCurrentCost(); got != want {
		t.Errorf("a totem costs %v, want it untouched at %v", got, want)
	}
	if got, want := healer.GetStat(stats.Hit)-base.GetStat(stats.Hit), 5*float64(core.HitRatingPerHitChance); math.Abs(got-want) > 1e-6 {
		t.Errorf("hit rose by %v, want %v", got, want)
	}
}

func TestTidalMasteryAddsCritToHealsNotLightning(t *testing.T) {
	_, healer := newHealer(t, 60, restoTalents(map[int]int{posTidalMastery: 5}))

	want := 5 * float64(core.CritRatingPerCritChance)
	for _, spell := range []*core.Spell{healer.HealingWave[10], healer.LesserHealingWave[6], healer.ChainHeal[3]} {
		if spell.BonusCritRating != want {
			t.Errorf("%s has %v bonus crit, want %v", spell.ActionID, spell.BonusCritRating, want)
		}
	}
	if got := healer.LightningBolt[10].BonusCritRating; got != 0 {
		t.Errorf("Lightning Bolt has %v bonus crit; the talent says healing spells only", got)
	}
}

func TestPurificationAndHealingWayAddTheirPercentages(t *testing.T) {
	sim, healer := newHealer(t, 60, restoTalents(map[int]int{posPurification: 5, posHealingWay: 3}))
	tank := raidMember(sim, healsim.TankIndex)

	// Healing Way is Healing Wave alone, Purification every healing spell,
	// and the two add (10 percent and 25 percent make 35).
	for name, want := range map[string]struct {
		spell      *core.Spell
		multiplier float64
	}{
		"Healing Wave":        {healer.HealingWave[10], 1.35},
		"Lesser Healing Wave": {healer.LesserHealingWave[6], 1.10},
		"Chain Heal":          {healer.ChainHeal[3], 1.10},
	} {
		healing, crit := healOf(sim, want.spell, tank)
		assertRoll(t, name, want.spell, healing, crit, want.multiplier)
	}
}

func TestNaturalGraceReducesSpellThreat(t *testing.T) {
	_, healer := newHealer(t, 60, restoTalents(map[int]int{posNaturalGrace: 3}))

	for _, spell := range []*core.Spell{healer.HealingWave[10], healer.LightningBolt[10]} {
		if got := spell.ThreatMultiplier; math.Abs(got-0.85) > 1e-9 {
			t.Errorf("%s threat multiplier = %v, want 0.85", spell.ActionID, got)
		}
	}
}

func TestMindfulnessKeepsPartOfTheRegenWhileCasting(t *testing.T) {
	for rank, want := range map[int]float64{1: 0.17, 2: 0.33, 3: 0.50} {
		_, healer := newHealer(t, 60, restoTalents(map[int]int{posMindfulness: rank}))

		if got := healer.PseudoStats.SpiritRegenRateCasting; got != want {
			t.Errorf("rank %d keeps %v of spirit regen while casting, want %v", rank, got, want)
		}
	}
}

func TestNaturesSwiftnessMakesTheNextHealInstant(t *testing.T) {
	sim, healer := newHealer(t, 60, restoTalents(map[int]int{posNaturesSwiftness: 1}))
	tank := raidMember(sim, healsim.TankIndex)
	swiftness := healer.GetSpell(core.ActionID{SpellID: 16188})
	heal := healer.HealingWave[10]

	if heal.CastTime() == 0 {
		t.Fatal("Healing Wave should take time before Nature's Swiftness")
	}
	swiftness.Cast(sim, &healer.Unit)
	if got := heal.CastTime(); got != 0 {
		t.Errorf("Healing Wave casts in %v under Nature's Swiftness, want instant", got)
	}
	heal.Cast(sim, tank)
	if heal.CastTime() == 0 {
		t.Error("Nature's Swiftness should be spent by the heal")
	}
	if swiftness.IsReady(sim) {
		t.Error("Nature's Swiftness should be on its 3 minute cooldown")
	}
}

func TestNaturesSwiftnessIsLeftToTheHealersRotation(t *testing.T) {
	_, healer := newHealer(t, 60, restoTalents(map[int]int{posNaturesSwiftness: 1}))

	for _, mcd := range healer.GetMajorCooldowns() {
		if mcd.Spell.ActionID.SpellID == 16188 && mcd.ShouldActivate(nil, &healer.Character) {
			t.Error("autocast would burn a healer's Nature's Swiftness; it should be manual")
		}
	}
}

func TestWaterShieldReturnsManaOnAHealingCrit(t *testing.T) {
	sim, healer := newHealer(t, 60, restoTalents(map[int]int{posWaterShield: 1}))
	tank := raidMember(sim, healsim.TankIndex)
	heal := healer.LesserHealingWave[6]

	heal.BonusCritRating += 100 * float64(core.CritRatingPerCritChance)
	healer.WaterShield.Cast(sim, &healer.Unit)
	healer.SpendMana(sim, healer.MaxMana()/2, healer.NewManaMetrics(core.ActionID{SpellID: 1}))

	before := healer.CurrentMana()
	heal.ApplyEffects(sim, tank, heal)
	if got, want := healer.CurrentMana()-before, healer.MaxMana()*0.02; math.Abs(got-want) > 1e-6 {
		t.Errorf("a critical heal returned %.2f mana, want 2%% of the pool (%.2f)", got, want)
	}
}

func TestMindfulnessAndPurificationDoNotLeakIntoOtherTrees(t *testing.T) {
	_, healer := newHealer(t, 60, "")

	if got := healer.PseudoStats.SpiritRegenRateCasting; got != 0 {
		t.Errorf("an untalented shaman keeps %v of its regen while casting", got)
	}
	if got := healer.HealingWave[10].DamageMultiplierAdditive; got != 1 {
		t.Errorf("an untalented Healing Wave has additive multiplier %v", got)
	}
}
