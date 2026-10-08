package restoration

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/healsim"
	"github.com/wowsims/classic/sim/shaman"
)

// castsPerSpell is how many times a heal is cast to see its whole roll.
const castsPerSpell = 300

// critMultiplier is a healing crit's multiplier: the spell crit 1.5.
const critMultiplier = 1.5

// healOf casts spell on target once and returns the healing it dealt
// (overheal included), and whether that cast crit.
func healOf(sim *core.Simulation, spell *core.Spell, target *core.Unit) (healing float64, crit bool) {
	before := spell.SpellMetrics[target.UnitIndex]
	spell.ApplyEffects(sim, target, spell)
	after := spell.SpellMetrics[target.UnitIndex]
	return after.TotalHealing - before.TotalHealing, after.Crits > before.Crits
}

// assertRoll fails unless healing is a roll of spell at the healer's
// healing power: the client's range, plus coefficient x power, times
// multiplier, doubled-and-a-half on a crit.
func assertRoll(t *testing.T, label string, spell *core.Spell, healing float64, crit bool, multiplier float64) {
	t.Helper()

	power := testHealingPower * spell.BonusCoefficient
	low := (spell.ClientBaseDamage[0] + power) * multiplier
	high := (spell.ClientBaseDamage[1] + power) * multiplier
	if crit {
		low *= critMultiplier
		high *= critMultiplier
	}
	if healing < low-0.01 || healing > high+0.01 {
		t.Errorf("%s healed %.2f, want %.2f to %.2f (crit %v)", label, healing, low, high, crit)
	}
}

// directHeals are the casts that heal one target once, by rank.
func directHeals(healer *shaman.Shaman) map[string][]*core.Spell {
	return map[string][]*core.Spell{
		"Healing Wave":        healer.HealingWave,
		"Lesser Healing Wave": healer.LesserHealingWave,
	}
}

func TestEveryRankHealsWithinTheClientRollPlusItsCoefficient(t *testing.T) {
	sim, healer := newHealer(t, 60, "")
	tank := raidMember(sim, healsim.TankIndex)

	for name, ranks := range directHeals(healer.Shaman) {
		for rank, spell := range ranks {
			if spell == nil {
				continue
			}
			t.Run(fmt.Sprintf("%s rank %d", name, rank), func(t *testing.T) {
				var min, max float64 = math.MaxFloat64, 0
				for i := 0; i < castsPerSpell; i++ {
					healing, crit := healOf(sim, spell, tank)
					assertRoll(t, name, spell, healing, crit, 1)
					if !crit {
						min, max = math.Min(min, healing), math.Max(max, healing)
					}
				}
				if max-min < (spell.ClientBaseDamage[1]-spell.ClientBaseDamage[0])/2 {
					t.Errorf("%d casts only spread %.1f to %.1f; the roll should vary over %v", castsPerSpell, min, max, spell.ClientBaseDamage)
				}
			})
		}
	}
}

func TestHealingWaveRankTenIsTheClientHealAtLevelSixty(t *testing.T) {
	_, healer := newHealer(t, 60, "")

	spell := healer.HealingWave[10]
	if spell.ActionID.SpellID != 25357 {
		t.Fatalf("rank 10 is spell %d, want 25357", spell.ActionID.SpellID)
	}
	// 1615 at the spell's own level 60, spread 13.256 percent wide.
	if got := spell.ClientBaseDamage; math.Abs(got[0]-1507.95) > 0.01 || math.Abs(got[1]-1722.05) > 0.01 {
		t.Errorf("rank 10 rolls %v, want 1507.95 to 1722.05", got)
	}
	if spell.BonusCoefficient != 0.857 {
		t.Errorf("rank 10 coefficient = %v, want 0.857", spell.BonusCoefficient)
	}
	if got := spell.Cost.GetCurrentCost(); got != 620 {
		t.Errorf("rank 10 costs %v, want 620", got)
	}
	if got := spell.DefaultCast.CastTime; got != 3*time.Second {
		t.Errorf("rank 10 casts in %v, want 3s", got)
	}
}

func TestLesserHealingWaveRankSixIsTheOneAPlayerLearns(t *testing.T) {
	_, healer := newHealer(t, 60, "")

	spell := healer.LesserHealingWave[6]
	if spell.ActionID.SpellID != 10468 {
		t.Errorf("rank 6 is spell %d, want 10468 (27624 has no learn row)", spell.ActionID.SpellID)
	}
	if got := spell.DefaultCast.CastTime; got != 1500*time.Millisecond {
		t.Errorf("Lesser Healing Wave casts in %v, want 1.5s", got)
	}
}

func TestHealingSpellsAreHelpfulNatureSpells(t *testing.T) {
	_, healer := newHealer(t, 60, FullTalents)

	for _, spell := range []*core.Spell{healer.HealingWave[10], healer.LesserHealingWave[6], healer.ChainHeal[3], healer.Riptide[3]} {
		if !spell.Flags.Matches(core.SpellFlagHelpful) {
			t.Errorf("%s should be a helpful spell", spell.ActionID)
		}
		if !spell.SpellSchool.Matches(core.SpellSchoolNature) {
			t.Errorf("%s should be a Nature spell", spell.ActionID)
		}
		if !spell.ProcMask.Matches(core.ProcMaskSpellHealing) {
			t.Errorf("%s should proc as a healing spell", spell.ActionID)
		}
	}
}

// injure takes health off a raid member so a heal has room to land.
func injure(sim *core.Simulation, unit *core.Unit, amount float64) {
	unit.RemoveHealth(sim, amount)
}

func TestHealsLandOnTheTargetAndReportEffectiveHealing(t *testing.T) {
	sim, healer := newHealer(t, 60, "")
	tank := raidMember(sim, healsim.TankIndex)
	injure(sim, tank, 500)

	spell := healer.HealingWave[10]
	healing, _ := healOf(sim, spell, tank)

	metrics := spell.SpellMetrics[tank.UnitIndex]
	wantEffective := math.Min(healing, 500)
	if math.Abs(metrics.TotalEffectiveHealing-wantEffective) > 0.01 {
		t.Errorf("effective healing = %.2f, want %.2f of a %.2f heal on a 500 deficit", metrics.TotalEffectiveHealing, wantEffective, healing)
	}
}

func TestChainHealJumpsToTheTwoMostInjuredMembersAtHalfEach(t *testing.T) {
	sim, healer := newHealer(t, 60, "")
	tank := raidMember(sim, healsim.TankIndex)
	spell := healer.ChainHeal[3]

	// Members 1 to 4 lose 10, 40, 20 and 30 percent: 2 and 4 are the worst.
	for i, deficit := range map[int32]float64{1: 0.10, 2: 0.40, 3: 0.20, 4: 0.30} {
		injure(sim, raidMember(sim, i), 5000*deficit)
	}

	var primary float64
	primary, _ = healOf(sim, spell, tank)
	// The cast above also rolled its jumps; read each target's own total.
	jumps := map[int32]float64{}
	for index := int32(1); index <= 4; index++ {
		jumps[index] = spell.SpellMetrics[raidMember(sim, index).UnitIndex].TotalHealing
	}

	if jumps[1] != 0 || jumps[3] != 0 {
		t.Errorf("members 1 and 3 are not among the most injured but healed %.0f and %.0f", jumps[1], jumps[3])
	}
	if jumps[2] == 0 || jumps[4] == 0 {
		t.Fatalf("members 2 and 4 are the most injured and should be healed, got %.0f and %.0f", jumps[2], jumps[4])
	}
	assertChainRatio(t, "first jump (member 2)", jumps[2], primary, 0.5)
	assertChainRatio(t, "second jump (member 4)", jumps[4], primary, 0.25)
}

// assertChainRatio checks a jump against the primary heal. The two rolls are
// one roll, but each heal crits for itself, so the ratio is the falloff times
// 1, 1.5 or 1/1.5.
func assertChainRatio(t *testing.T, label string, jump, primary, falloff float64) {
	t.Helper()

	for _, factor := range []float64{1, critMultiplier, 1 / critMultiplier} {
		if math.Abs(jump/primary-falloff*factor) < 1e-6 {
			return
		}
	}
	t.Errorf("%s healed %.2f against a primary of %.2f, want %.2f of it (crit aside)", label, jump, primary, falloff)
}
