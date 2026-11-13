package item_effects_test

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// Gloves of Holy Might (867) since Blizzard's 9 November 2026 hotfix ("Gloves
// of Holy Might now work as intended"), native in client 1.60.1.70938:
// ItemEffect 238916 is TriggerType 1 (on equip) to 1330855, aura 42 with
// ProcChance 15, ProcCategoryRecovery 20000 and ProcTypeMask 69636 (0x11004:
// done melee auto attack, done harmful spell of no class, done harmful magic
// spell), triggering the 1325229 Smite. Before it, 70790's row was a chance
// on any melee hit. So a white swing procs it at 15% with 20 seconds between
// procs, and a melee special (0x10, done melee-class spell) never does.
func TestGlovesOfHolyMightProcsOnWhiteSwingsAtFifteenPercentEveryTwentySeconds(t *testing.T) {
	for _, tc := range []struct {
		name  string
		swing func(*core.Character) *core.Spell
		// step lets the simulation reach the white swing's damage, which
		// lands a spell batch after the swing.
		step     bool
		minProcs int
		maxProcs int
	}{
		// Each step is a spell batch, so the swings span a few seconds:
		// the 20 s recovery allows one proc.
		{"white swing", func(c *core.Character) *core.Spell { return c.AutoAttacks.MHAuto() }, true, 1, 1},
		{"melee special (Wing Clip)", func(c *core.Character) *core.Spell { return c.GetSpell(core.ActionID{SpellID: 14268}) }, false, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sim, character := unbuffedCharacter(t, proto.Class_ClassHunter, proto.ItemSlot_ItemSlotHands, 867)
			smite := character.GetSpell(core.ActionID{SpellID: 1325229})
			if smite == nil {
				t.Fatal("no Gloves of Holy Might Smite registered")
			}
			swing := tc.swing(character)
			if swing == nil {
				t.Fatal("no spell to swing with")
			}
			target := sim.Encounter.TargetUnits[0]
			landed := 0
			for i := 0; i < 400; i++ {
				before := swing.SpellMetrics[target.UnitIndex].Hits + swing.SpellMetrics[target.UnitIndex].Crits
				swing.ApplyEffects(sim, target, swing)
				if tc.step {
					sim.Step()
				}
				if swing.SpellMetrics[target.UnitIndex].Hits+swing.SpellMetrics[target.UnitIndex].Crits > before {
					landed++
				}
			}
			if landed < 100 {
				t.Fatalf("only %d of 400 swings landed", landed)
			}
			if sim.CurrentTime >= 20*time.Second {
				t.Fatalf("the swings took %v, past the 20 s recovery", sim.CurrentTime)
			}
			if procs := int(smite.SpellMetrics[target.UnitIndex].Casts); procs < tc.minProcs || procs > tc.maxProcs {
				t.Errorf("%d Smites from %d landed hits inside %v, want %d to %d", procs, landed, sim.CurrentTime, tc.minProcs, tc.maxProcs)
			}
		})
	}
}
