package hunter

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
)

// Forever's Devilsaur Eye (19991, required level 50) and Devilsaur Tooth
// (19992) are not Zul'Gurub's. The client's ItemEffect rows keep the old
// use spells but their SpellEffect rows changed:
//
//   - Devilsaur Eye, ItemEffect 101844: use 24352 on a 300000 ms cooldown.
//     24352 "Devilsaur Glare" is a 10 s aura with one dummy effect (aura 4)
//     and ProcTypeMask 4436: "Your next melee or ranged attack within 10 sec
//     will Root the target for 2.5 sec ... Only works against Beasts." No
//     attack power, no hit.
//   - Devilsaur Tooth, ItemEffect 101573: use 24353 on a 600000 ms cooldown.
//     24353 is aura 108 (spell mod, cost) -100% on class mask word 0 bit 24,
//     Revive Pet: "Reduces the Mana cost of your next Revive Pet cast within
//     30 sec by 100%." Nothing for the pet's attacks.
//
// Neither use does damage on a boss, so wearing either must leave the
// hunter's and the pet's numbers where the item's own stats put them.
func TestForeverDevilsaurTrinketsGiveNoCombatBonus(t *testing.T) {
	for _, tc := range []struct {
		name string
		item int32
	}{
		{"Devilsaur Eye", 19991},
		{"Devilsaur Tooth", 19992},
	} {
		sim, h := buildHunterWithCatWearing(t, tc.item)
		before := h.GetStats()
		petSpecialCrit := h.pet.specialAbility.BonusCritRating
		petDumpCrit := h.pet.focusDump.BonusCritRating

		for _, spellID := range []int32{24352, 24353} {
			spell := h.GetSpell(core.ActionID{SpellID: spellID})
			if spell == nil {
				continue
			}
			if spell.CanCast(sim, h.CurrentTarget) {
				spell.Cast(sim, h.CurrentTarget)
			}
		}

		after := h.GetStats()
		for _, s := range []stats.Stat{stats.AttackPower, stats.RangedAttackPower, stats.Hit, stats.Crit, stats.MeleeCrit} {
			if after[s] != before[s] {
				t.Errorf("%s: using it moved stat %v from %v to %v, want no change", tc.name, s, before[s], after[s])
			}
		}
		if got := h.pet.specialAbility.BonusCritRating; got != petSpecialCrit {
			t.Errorf("%s: the pet's Bite crit rating moved from %v to %v, want no change", tc.name, petSpecialCrit, got)
		}
		if got := h.pet.focusDump.BonusCritRating; got != petDumpCrit {
			t.Errorf("%s: the pet's Claw crit rating moved from %v to %v, want no change", tc.name, petDumpCrit, got)
		}
	}
}

func buildHunterWithCatWearing(t *testing.T, trinketID int32) (*core.Simulation, *Hunter) {
	t.Helper()
	items := make([]*proto.ItemSpec, proto.ItemSlot_ItemSlotRanged+1)
	for i := range items {
		items[i] = &proto.ItemSpec{}
	}
	items[proto.ItemSlot_ItemSlotRanged] = &proto.ItemSpec{Id: scopeTestGun}
	items[proto.ItemSlot_ItemSlotTrinket1] = &proto.ItemSpec{Id: trinketID}
	player := core.WithSpec(
		&proto.Player{
			Class: proto.Class_ClassHunter, Race: proto.Race_RaceOrc, Level: 60,
			Equipment: &proto.EquipmentSpec{Items: items},
			Buffs:     &proto.IndividualBuffs{},
		},
		&proto.Player_Hunter{Hunter: &proto.Hunter{Options: &proto.Hunter_Options{
			PetType: proto.Hunter_Options_Cat, PetUptime: 1, PetAttackSpeed: proto.Hunter_Options_OneTwo,
		}}},
	)
	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 60, Targets: []*proto.Target{core.DefaultTargetProtoLvl60}},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	return sim, sim.Raid.Parties[0].Players[0].(HunterAgent).GetHunter()
}
