package retribution

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// Infernal Trickster Leggings (17754) carry 1294769 "Increased Hit Chance -
// Ranged - 01", restricted by the client to bows, guns, crossbows and
// thrown weapons (SpellEquippedItems subclass mask 262156). A paladin
// wears no ranged weapon, so the hit never reaches its ranged attacks
// (Hammer of Wrath, and since the 11 November 2026 hotfix every
// Judgement); the engine gave it to every ranged cast of the wearer.
func TestRangedHitItemsNeedARangedWeapon(t *testing.T) {
	const infernalTricksterLeggings = 17754
	player := utilitySealPaladin(60, `{"type": "TypeAPL"}`)
	player.Equipment = &proto.EquipmentSpec{Items: []*proto.ItemSpec{
		{}, {}, {}, {}, {}, {}, {}, {}, {Id: infernalTricksterLeggings},
	}}
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 60, Targets: []*proto.Target{core.NewDefaultTarget()}},
		SimOptions: &proto.SimOptions{Iterations: 1, RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	character := sim.Raid.Parties[0].Players[0].GetCharacter()
	if got := character.Equipment[proto.ItemSlot_ItemSlotLegs].ID; got != infernalTricksterLeggings {
		t.Fatalf("legs are item %d, want %d: the item is not in the database", got, infernalTricksterLeggings)
	}
	for _, spellID := range []int32{24239, 20286} {
		spell := character.GetSpell(core.ActionID{SpellID: spellID})
		if spell == nil {
			t.Fatalf("spell %d is not registered at 60", spellID)
		}
		if spell.BonusHitRating != 0 {
			t.Errorf("spell %d carries %v bonus hit from the leggings with no ranged weapon worn, want 0", spellID, spell.BonusHitRating)
		}
	}
}
