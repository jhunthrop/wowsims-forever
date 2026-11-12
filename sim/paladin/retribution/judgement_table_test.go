package retribution

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// Blizzard's 11 November 2026 hotfix (t/2403040): "Judgement changed to now
// be considered a Ranged Attack. This means it no longer counts as a Melee
// attack for the purposes of item and effect procs and uses the Paladin's
// Ranged Hit chance." Server side: client 1.60.1.70888 still files the
// judgements on the melee table (SpellCategories DefenseType 2 and SpellMisc
// Attributes_0 0x200000 since 1.60.1.70707). So the Judgement ability and
// every judgement it resolves to roll the ranged attack table, and the
// damage judgements carry a ranged proc mask that no melee proc (Windfury
// Totem, a weapon enchant, Hand of Justice) reads.
func TestJudgementsAreRangedAttacks(t *testing.T) {
	player := utilitySealPaladin(60, `{"type": "TypeAPL"}`)
	player.TalentsString = talentString(t, map[string]int{"seal_of_command": 1})
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 60, Targets: []*proto.Target{core.NewDefaultTarget()}},
		SimOptions: &proto.SimOptions{Iterations: 1, RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	unit := &sim.Raid.Parties[0].Players[0].GetCharacter().Unit

	judgements := []struct {
		name    string
		spellID int32
		damage  bool
	}{
		{"Judgement", 20271, false},
		{"Judgement of Righteousness rank 8", 20286, true},
		{"Judgement of Command rank 5", 20966, true},
		{"Judgement of Fury rank 7", 20414, true},
		{"Judgement of Light rank 4", 20346, false},
		{"Judgement of Wisdom rank 3", 20355, false},
	}
	for _, j := range judgements {
		spell := unit.GetSpell(core.ActionID{SpellID: j.spellID})
		if spell == nil {
			t.Fatalf("%s (%d) is not registered at 60", j.name, j.spellID)
		}
		if spell.CastType != proto.CastType_CastTypeRanged {
			t.Errorf("%s cast type = %v, want ranged (the ranged attack table)", j.name, spell.CastType)
		}
		if !j.damage {
			continue
		}
		if spell.DefenseType != core.DefenseTypeRanged {
			t.Errorf("%s defense type = %v, want ranged", j.name, spell.DefenseType)
		}
		if spell.ProcMask.Matches(core.ProcMaskMelee) {
			t.Errorf("%s proc mask %v counts as a melee attack; a melee proc would fire on it", j.name, spell.ProcMask)
		}
		if !spell.ProcMask.Matches(core.ProcMaskRanged) {
			t.Errorf("%s proc mask %v is not a ranged attack's", j.name, spell.ProcMask)
		}
	}
}
