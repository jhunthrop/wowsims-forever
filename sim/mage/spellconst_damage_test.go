package mage

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/common/clientdamage/clientdamagetest"
	"github.com/wowsims/classic/sim/core/spellconst"
)

var damageCheckLevels = []int{10, 20, 30, 38, 40, 50, 60}

// damageLadder is one spell's per-rank tables, as the abilities read
// them: the generated ones for most spells, the hand-written ones keyed
// to the engine's own ids where the generator would pick a different id
// for a rank (Fire Blast, Arcane Blast, Flamestrike; see the headers of
// their files).
type damageLadder struct {
	name     string
	first    int // first rank to check
	ranks    int
	spellID  []int32
	level    []int
	mana     []float64 // nil: the spell has no flat cost to compare
	coeff    []float64
	baseDmg  [][]float64
	perLevel []float64
	maxLevel []int
}

func damageLadders() []damageLadder {
	return []damageLadder{
		{"Scorch", 1, ScorchRanks, ScorchSpellId[:], ScorchLevel[:], ScorchManaCost[:], ScorchSpellCoeff[:], ScorchBaseDamage[:], ScorchPointsPerLevel[:], ScorchMaxLevel[:]},
		{"Fireball", 1, FireballRanks, FireballSpellId[:], FireballLevel[:], FireballManaCost[:], FireballSpellCoeff[:], FireballBaseDamage[:], FireballPointsPerLevel[:], FireballMaxLevel[:]},
		{"Frostbolt", 1, FrostboltRanks, FrostboltSpellId[:], FrostboltLevel[:], FrostboltManaCost[:], FrostboltSpellCoeff[:], FrostboltBaseDamage[:], FrostboltPointsPerLevel[:], FrostboltMaxLevel[:]},
		// Frostfire Bolt's generated rank 0 is the level-1 "Gain the
		// ability" teaching spell, not a castable rank, so the ladder
		// starts at rank 1 (401502, level 40).
		{"Frostfire Bolt", 1, FrostfireBoltRanks, FrostfireBoltSpellId[:], FrostfireBoltLevel[:], FrostfireBoltManaCost[:], FrostfireBoltSpellCoeff[:], FrostfireBoltBaseDamage[:], FrostfireBoltPointsPerLevel[:], FrostfireBoltMaxLevel[:]},
		{"Pyroblast", 1, PyroblastRanks, PyroblastSpellId[:], PyroblastLevel[:], PyroblastManaCost[:], PyroblastSpellCoeff[:], PyroblastBaseDamage[:], PyroblastPointsPerLevel[:], PyroblastMaxLevel[:]},
		{"Arcane Explosion", 1, ArcaneExplosionRanks, ArcaneExplosionSpellId[:], ArcaneExplosionLevel[:], ArcaneExplosionManaCost[:], ArcaneExplosionSpellCoeff[:], ArcaneExplosionBaseDamage[:], ArcaneExplosionPointsPerLevel[:], ArcaneExplosionMaxLevel[:]},
		{"Blast Wave", 1, BlastWaveRanks, BlastWaveSpellId[:], BlastWaveLevel[:], BlastWaveManaCost[:], BlastWaveSpellCoeff[:], BlastWaveBaseDamage[:], BlastWavePointsPerLevel[:], BlastWaveMaxLevel[:]},
		{"Frost Nova", 1, FrostNovaRanks, FrostNovaSpellId[:], FrostNovaLevel[:], FrostNovaManaCost[:], FrostNovaSpellCoeff[:], FrostNovaBaseDamage[:], FrostNovaPointsPerLevel[:], FrostNovaMaxLevel[:]},
		{"Cone of Cold", 1, ConeOfColdRanks, ConeOfColdSpellId[:], ConeOfColdLevel[:], ConeOfColdManaCost[:], ConeOfColdSpellCoeff[:], ConeOfColdBaseDamage[:], ConeOfColdPointsPerLevel[:], ConeOfColdMaxLevel[:]},
		{"Ice Lance", 1, IceLanceRanks, IceLanceSpellId[:], IceLanceLevel[:], IceLanceManaCost[:], IceLanceSpellCoeff[:], IceLanceBaseDamage[:], IceLancePointsPerLevel[:], IceLanceMaxLevel[:]},
		{"Fire Blast", 1, FireBlastRanks, FireBlastSpellId[:], FireBlastLevel[:], FireBlastManaCost[:], FireBlastSpellCoeff[:], FireBlastBaseDamage[:], FireBlastPointsPerLevel[:], FireBlastMaxLevel[:]},
		{"Arcane Blast", 1, ArcaneBlastRanks, ArcaneBlastSpellId[:], ArcaneBlastLevel[:], nil, ArcaneBlastSpellCoeff[:], ArcaneBlastBaseDamage[:], ArcaneBlastPointsPerLevel[:], ArcaneBlastMaxLevel[:]},
		{"Flamestrike", 1, FlamestrikeRanks, FlamestrikeSpellId[:], FlamestrikeLevel[:], FlamestrikeManaCost[:], FlamestrikeSpellCoeff[:], FlamestrikeBaseDamage[:], FlamestrikePointsPerLevel[:], FlamestrikeMaxLevel[:]},
		// Arcane Missiles: the generated arrays are the missile's (the
		// channel's trigger spell), so there is no cost to compare.
		{"Arcane Missiles (missile)", 1, ArcaneMissilesRanks, ArcaneMissilesSpellId[:], ArcaneMissilesLevel[:], nil, ArcaneMissilesSpellCoeff[:], ArcaneMissilesBaseDamage[:], ArcaneMissilesPointsPerLevel[:], ArcaneMissilesMaxLevel[:]},
	}
}

func TestMageDamageLaddersMatchTheClient(t *testing.T) {
	client := clientdamagetest.LoadClient(t, "..", "mage")
	for _, ladder := range damageLadders() {
		for rank := ladder.first; rank <= ladder.ranks; rank++ {
			id := ladder.spellID[rank]
			spell, ok := client.ByID(id)
			if !ok {
				t.Fatalf("%s rank %d: id %d is not in the client table", ladder.name, rank, id)
			}
			if ladder.level[rank] != spell.SpellLevel {
				t.Errorf("%s rank %d (%d): level %d, client %d", ladder.name, rank, id, ladder.level[rank], spell.SpellLevel)
			}
			if ladder.mana != nil && math.Abs(ladder.mana[rank]-spell.Cost) > 0.5 {
				t.Errorf("%s rank %d (%d): mana %v, client %v", ladder.name, rank, id, ladder.mana[rank], spell.Cost)
			}
			index := damageEffectIndex(spell)
			if effect := spell.Effects[index]; effect.CoefficientSource == "table" && math.Abs(ladder.coeff[rank]-effect.ResolvedSPCoefficient) > 0.002 {
				t.Errorf("%s rank %d (%d): coefficient %v, client %v", ladder.name, rank, id, ladder.coeff[rank], effect.ResolvedSPCoefficient)
			}
			for _, casterLevel := range damageCheckLevels {
				roll := clientdamage.Roll(ladder.baseDmg[rank], ladder.perLevel[rank], ladder.level[rank], ladder.maxLevel[rank], casterLevel)
				clientdamagetest.AssertRoll(t, client, ladder.name, id, index, casterLevel, roll)
			}
		}
	}
}

// damageEffectIndex is the effect a ladder describes: the school-damage
// effect, else the first effect (the generator's primary effect).
func damageEffectIndex(spell spellconst.Spell) int {
	for _, effect := range spell.Effects {
		if effect.Effect == 2 {
			return effect.Index
		}
	}
	return spell.Effects[0].Index
}

// A periodic or ground-aura effect the generator does not emit carries a
// flat per-tick amount in a hand table; each must equal the client's
// effect at every caster level (the client states no growth for them).
func TestMageTickTablesMatchTheClient(t *testing.T) {
	client := clientdamagetest.LoadClient(t, "..", "mage")
	cases := []struct {
		name        string
		ids         []int32 // client spells holding the tick effect, by rank
		effectIndex int
		ticks       []float64
	}{
		{"Fireball dot", FireballSpellId[:], 1, FireballDotTickDamage[:]},
		{"Pyroblast dot", PyroblastSpellId[:], 1, PyroblastDotTickDamage[:]},
		{"Frostfire Bolt dot", FrostfireBoltSpellId[:], 2, FrostfireBoltDotTickDamage[:]},
		// The ground aura's tick lives on its own client rows.
		{"Flamestrike ground aura", []int32{0, 1279983, 1279985, 1279987, 1279988, 1279989, 1279990}, 0, FlamestrikeDotTickDamage[:]},
	}
	for _, tc := range cases {
		for rank := 1; rank < len(tc.ticks); rank++ {
			for _, casterLevel := range damageCheckLevels {
				tick := tc.ticks[rank]
				clientdamagetest.AssertRoll(t, client, tc.name, tc.ids[rank], tc.effectIndex, casterLevel, [2]float64{tick, tick})
			}
		}
	}
}

func TestFireballDotTicksFollowTheClientDuration(t *testing.T) {
	client := clientdamagetest.LoadClient(t, "..", "mage")
	for rank := 1; rank <= FireballRanks; rank++ {
		spell, _ := client.ByID(FireballSpellId[rank])
		if got := spell.DurationMS / spell.Effects[1].PeriodMS; got != FireballDotTicks[rank] {
			t.Errorf("Fireball rank %d: %d ticks, client duration/period %d", rank, FireballDotTicks[rank], got)
		}
	}
}

func TestBlizzardTickMatchesTheClient(t *testing.T) {
	client := clientdamagetest.LoadClient(t, "..", "mage")
	// Blizzard's engine ids are the channel; the damage is on its own
	// rows (6141's sibling 1279977 and so on), one tick a second.
	tickIDs := []int32{0, 1279976, 1279977, 1279978, 1279979, 1279980, 1279949}
	for rank := 1; rank <= BlizzardRanks; rank++ {
		spell, _ := client.ByID(tickIDs[rank])
		if spell.SpellLevel != BlizzardLevel[rank] {
			t.Errorf("Blizzard rank %d: level %d, client %d", rank, BlizzardLevel[rank], spell.SpellLevel)
		}
		if math.Abs(spell.Effects[0].ResolvedSPCoefficient-BlizzardTickSpellCoeff[rank]) > 0.002 {
			t.Errorf("Blizzard rank %d: coefficient %v, client %v", rank, BlizzardTickSpellCoeff[rank], spell.Effects[0].ResolvedSPCoefficient)
		}
		for _, casterLevel := range damageCheckLevels {
			roll := clientdamage.Roll(BlizzardTickBaseDamage[rank], BlizzardTickPointsPerLevel[rank], BlizzardLevel[rank], BlizzardTickMaxLevel[rank], casterLevel)
			clientdamagetest.AssertRoll(t, client, "Blizzard tick", tickIDs[rank], 0, casterLevel, roll)
		}
	}
}

func TestFrostfireBoltDotTicksFollowTheClientDuration(t *testing.T) {
	client := clientdamagetest.LoadClient(t, "..", "mage")
	for rank := 1; rank <= FrostfireBoltRanks; rank++ {
		spell, _ := client.ByID(FrostfireBoltSpellId[rank])
		if got := spell.DurationMS / spell.Effects[2].PeriodMS; got != FrostfireBoltDotTicks {
			t.Errorf("Frostfire Bolt rank %d: %d ticks, client duration/period %d", rank, FrostfireBoltDotTicks, got)
		}
	}
}
