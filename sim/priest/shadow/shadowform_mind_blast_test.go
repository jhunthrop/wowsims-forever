package shadow

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/priest"
)

// The client's Shadowform text: "reducing the Mana cost of all Shadow
// spells by 50%, increasing the critical strike damage bonus of your Shadow
// spells by 100%". Mind Blast is a Shadow spell, but it carried no class
// mask, so the ClassSpellsOnly mods skipped it: full mana cost and a plain
// crit bonus while Mind Flay and Shadow Word: Pain got both (audit P3).
func TestShadowformReachesMindBlast(t *testing.T) {
	sim, built, _ := newShadowPriestSimWithTalents(t, map[string]int{"shadowform": 1})
	mb := built.MindBlast[priest.MindBlastRanks]
	if mb == nil {
		t.Fatal("top Mind Blast rank not registered at level 60")
	}
	if built.ShadowformAura.IsActive() {
		built.ShadowformAura.Deactivate(sim)
	}
	baseCost := mb.Cost.GetCurrentCost()
	baseBonus := mb.CritDamageBonus

	built.ShadowformAura.Activate(sim)
	if got, want := mb.Cost.GetCurrentCost(), baseCost/2; math.Abs(got-want) > 0.5 {
		t.Errorf("Mind Blast costs %v in Shadowform, want %v (half of %v)", got, want, baseCost)
	}
	if got, want := mb.CritDamageBonus-baseBonus, 1.0; math.Abs(got-want) > 1e-9 {
		t.Errorf("Shadowform raised Mind Blast's crit damage bonus by %v, want %v", got, want)
	}
}

// Forever made Devouring Plague a normal Shadow spell rather than an Undead
// racial: the client's SkillLineAbility rows for it (2944 through 19280)
// carry every race, and the trainer teaches it to every priest. The engine
// only registered it for Undead, so a gnome or dwarf shadow priest never
// cast it.
func TestDevouringPlagueIsBaselineForEveryRace(t *testing.T) {
	for _, race := range []proto.Race{proto.Race_RaceGnome, proto.Race_RaceDwarf, proto.Race_RaceHuman, proto.Race_RaceTroll} {
		built := newShadowPriestOfRace(t, race)
		if built.DevouringPlague == nil || built.DevouringPlague[priest.DevouringPlagueRanks] == nil {
			t.Errorf("%v: Devouring Plague is not registered at level 60", race)
		}
	}
}
