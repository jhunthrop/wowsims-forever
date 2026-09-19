package dpswarrior

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/warrior"
)

// The five SpellMod_PowerCost_Flat talents are the riskiest part of the
// declarative rewrite: the ability files now carry the UNDISCOUNTED cost
// and the discount is a mod registered in talents.go, so a mod that
// binds to nothing leaves the cost silently too high and a mod applied
// twice leaves it too low. Neither shows up as anything but a golden
// moving by a few DPS.
//
// This asserts the arithmetic end to end - through buildMod,
// OnSpellRegistered, Cost.FlatModifier and ApplyCostModifiers - by
// reading GetCurrentCost off the registered spell under a build that
// takes the talent and under one that takes nothing.
func TestTheRageDiscountTalentsLandOnTheirSpells(t *testing.T) {
	type spellCost struct {
		name string
		// get is the spell the talent discounts, off a built warrior.
		get func(*warrior.Warrior) *warrior.WarriorSpell
		// undiscounted is the cost the ability file registers, which is
		// the client's own cost; discounted is what the talent leaves.
		undiscounted float64
		fury         float64
		protection   float64
	}

	// ForeverFuryTalents takes Improved Heroic Strike 3, Improved Cleave
	// 3 and Improved Execute 1; ForeverProtectionTalents takes Improved
	// Thunder Clap 3 and Improved Sunder Armor 3. Between them the two
	// reference builds exercise all five discounts.
	cases := []spellCost{
		// Both reference builds take Improved Heroic Strike 3.
		{"Heroic Strike", func(w *warrior.Warrior) *warrior.WarriorSpell { return w.HeroicStrike },
			15, 12, 12},
		{"Cleave", func(w *warrior.Warrior) *warrior.WarriorSpell { return w.Cleave },
			20, 17, 20},
		{"Execute", func(w *warrior.Warrior) *warrior.WarriorSpell { return w.Execute },
			15, 12, 15},
		{"Thunder Clap", func(w *warrior.Warrior) *warrior.WarriorSpell { return w.ThunderClap },
			20, 20, 14},
		{"Sunder Armor", func(w *warrior.Warrior) *warrior.WarriorSpell { return w.SunderArmor },
			15, 15, 12},
	}

	for _, build := range []struct {
		label   string
		talents string
		want    func(spellCost) float64
	}{
		{"no talents", emptyWarriorTalents, func(c spellCost) float64 { return c.undiscounted }},
		{"ForeverFuryTalents", warrior.ForeverFuryTalents, func(c spellCost) float64 { return c.fury }},
		{"ForeverProtectionTalents", warrior.ForeverProtectionTalents, func(c spellCost) float64 { return c.protection }},
	} {
		t.Run(build.label, func(t *testing.T) {
			war := buildWarriorForCostTest(t, build.talents)
			for _, c := range cases {
				spell := c.get(war)
				if spell == nil {
					t.Errorf("%s is not registered, so its discount cannot be checked", c.name)
					continue
				}
				if got, want := spell.Cost.GetCurrentCost(), build.want(c); got != want {
					t.Errorf("%s costs %v rage, want %v", c.name, got, want)
				}
			}
		})
	}
}

// emptyWarriorTalents is a legal, empty talent string of the client's
// own widths, built from TalentTreeSizes rather than typed so it cannot
// fall out of step with the generated tree.
var emptyWarriorTalents = emptyTalentString()

func emptyTalentString() string {
	var out []byte
	for i, size := range warrior.TalentTreeSizes {
		if i > 0 {
			out = append(out, '-')
		}
		for j := 0; j < size; j++ {
			out = append(out, '0')
		}
	}
	return string(out)
}

// buildWarriorForCostTest stands up one warrior through the shipping
// agent factory, so the spells under test are registered exactly as a
// sim registers them.
func buildWarriorForCostTest(t *testing.T, talents string) *warrior.Warrior {
	t.Helper()

	sim := core.NewSim(&proto.RaidSimRequest{
		SimOptions: &proto.SimOptions{RandomSeed: 1},
		Raid: &proto.Raid{
			Parties: []*proto.Party{{
				Players: []*proto.Player{{
					Name:          "Warrior",
					Class:         proto.Class_ClassWarrior,
					Race:          proto.Race_RaceOrc,
					TalentsString: talents,
					Consumes:      &proto.Consumes{},
					Buffs:         &proto.IndividualBuffs{},
					Spec:          PlayerOptionsFury,
					Equipment:     &proto.EquipmentSpec{},
				}},
				Buffs: &proto.PartyBuffs{},
			}},
		},
		Encounter: &proto.Encounter{
			Targets:  []*proto.Target{{Name: "target", Level: 63, MobType: proto.MobType_MobTypeDemon}},
			Duration: 60,
		},
	}, simsignals.CreateSignals())
	sim.Reset()

	agent, ok := sim.Raid.Parties[0].Players[0].(warrior.WarriorAgent)
	if !ok {
		t.Fatalf("the raid's first player is not a warrior agent")
	}
	return agent.GetWarrior()
}
