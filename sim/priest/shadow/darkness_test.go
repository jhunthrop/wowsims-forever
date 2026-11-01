package shadow

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
	"github.com/wowsims/classic/sim/priest"
)

// Forever's Darkness (talent spell 15259, client 1.60.1.70707) is one
// SPELL_AURA_MOD_DAMAGE_PERCENT_DONE effect (aura 79) on the Shadow school
// (misc 32): "Increases your Shadow damage done by 2%" a rank. It reaches
// all of the priest's Shadow damage, Shadow Word: Death and the spell-power
// part of Mind Blast included, unlike Era's three class-masked spell mods.
func TestDarknessRaisesAllShadowDamageDone(t *testing.T) {
	_, without, _ := newShadowPriestSimWithTalents(t, nil)
	_, with, _ := newShadowPriestSimWithTalents(t, map[string]int{"darkness": 5})

	base := without.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexShadow]
	got := with.PseudoStats.SchoolDamageDealtMultiplier[stats.SchoolIndexShadow]
	if want := base * 1.10; math.Abs(got-want) > 1e-9 {
		t.Fatalf("Darkness 5/5 Shadow damage multiplier = %v, want %v (+10%% of all Shadow damage)", got, want)
	}

	// The school multiplier carries it; no spell keeps a second copy.
	for _, pair := range [][2]float64{
		{with.MindBlast[priest.MindBlastRanks].BaseDamageMultiplierAdditive, without.MindBlast[priest.MindBlastRanks].BaseDamageMultiplierAdditive},
		{with.ShadowWordPain[priest.ShadowWordPainRanks].DamageMultiplierAdditive, without.ShadowWordPain[priest.ShadowWordPainRanks].DamageMultiplierAdditive},
	} {
		if pair[0] != pair[1] {
			t.Errorf("a spell carries its own Darkness multiplier (%v against %v without the talent)", pair[0], pair[1])
		}
	}
}

// Shadow Word: Death is a Shadow damage spell, so a landed cast adds a
// Shadow Weaving stack like every other one ("Your Shadow damage spells
// have a 100% chance to increase the Shadow damage the target takes from
// you by 2%", 70707 talent text).
func TestShadowWordDeathAddsShadowWeaving(t *testing.T) {
	sim, built, target := newShadowPriestSimWithTalents(t, map[string]int{"shadow_weaving": 3})
	spell := built.ShadowWordDeath[priest.ShadowWordDeathRanks]
	for i := 0; i < 20 && built.ShadowWeavingAura.GetStacks() == 0; i++ {
		spell.ApplyEffects(sim, target, spell)
	}
	if built.ShadowWeavingAura.GetStacks() == 0 {
		t.Error("twenty Shadow Word: Death casts added no Shadow Weaving stack")
	}
}

// Dark Sacrifice is the Undead priest's racial: every rank's
// SkillLineAbility row (1277324 to 1277328, 1.60.1.70707) carries
// RaceMasks_0 16, the Undead bit, so no other race learns it.
func TestDarkSacrificeIsUndeadOnly(t *testing.T) {
	for _, race := range []proto.Race{proto.Race_RaceTroll, proto.Race_RaceHuman, proto.Race_RaceGnome} {
		built := newShadowPriestOfRace(t, race)
		for rank, spell := range built.DarkSacrifice {
			if spell != nil {
				t.Errorf("%v priest has Dark Sacrifice rank %d registered", race, rank)
			}
		}
	}
	undead := newShadowPriestOfRace(t, proto.Race_RaceUndead)
	if undead.DarkSacrifice[priest.DarkSacrificeRanks] == nil {
		t.Error("an Undead priest has no Dark Sacrifice at level 60")
	}
}
