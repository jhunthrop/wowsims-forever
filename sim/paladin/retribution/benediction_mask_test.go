package retribution

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// Benediction (20101) effect 0 is aura 108, misc 14 (cost), -10 at 5/5,
// with SpellClassMask 0x1be11fbf / 0x26018242 / 0x20212100 / 0x8800 in
// 1.60.1.70707. That mask holds the Seals and also Judgement (0x800000),
// Consecration (0x20), Exorcism (mask_2 0x2000), Holy Strike (mask_3
// 0x8000), Holy Shock (0x200000) and Holy Shield (20925/20927/20928,
// mask_1 0x40); not Hammer of Wrath (mask_1 0x80).
// The text: "all instant cast spells and abilities".
func TestBenedictionDiscountsEveryInstantItsMaskNames(t *testing.T) {
	player := utilitySealPaladin(60, `{"type": "TypeAPL"}`)
	player.TalentsString = talentString(t, map[string]int{"benediction": 5, "holy_shield": 1})
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 60, Targets: []*proto.Target{core.NewDefaultTarget()}},
		SimOptions: &proto.SimOptions{Iterations: 1, RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	unit := &sim.Raid.Parties[0].Players[0].GetCharacter().Unit
	for name, id := range map[string]int32{"Judgement": 20271, "Consecration": 20924, "Holy Strike": 10333, "Exorcism": 10314, "Holy Shield": 20928} {
		spell := unit.GetSpell(core.ActionID{SpellID: id})
		if spell == nil {
			t.Errorf("%s (%d) is not registered at 60", name, id)
			continue
		}
		if got := spell.Cost.Multiplier; got != 90 {
			t.Errorf("%s costs %v%% of its base with Benediction 5/5, want 90", name, got)
		}
	}
	if hammer := unit.GetSpell(core.ActionID{SpellID: 24239}); hammer != nil && hammer.Cost.Multiplier != 100 {
		t.Errorf("Hammer of Wrath costs %v%% with Benediction 5/5, want 100 (not in the mask)", hammer.Cost.Multiplier)
	}
}
