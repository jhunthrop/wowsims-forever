package item_effects_test

import (
	"testing"

	engine "github.com/wowsims/classic/sim"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
)

const (
	blackfathomStoneEnchant int32 = 7098
	stoneTestStaff          int32 = 873 // Staff of Jordan
)

func stonedWarrior(t *testing.T, enchant int32) *core.Character {
	t.Helper()
	engine.RegisterAll()
	if _, ok := core.EnchantsByEffectID[enchant]; !ok && enchant != 0 {
		// The stone is in the site's simdb only; its row carries no stats
		// (pipeline/simdb/enchants.py) and the engine applies the hit.
		core.EnchantsByEffectID[enchant] = core.Enchant{EffectID: enchant}
	}
	items := make([]*proto.ItemSpec, proto.ItemSlot_ItemSlotRanged+1)
	for i := range items {
		items[i] = &proto.ItemSpec{}
	}
	items[proto.ItemSlot_ItemSlotMainHand] = &proto.ItemSpec{Id: stoneTestStaff, Enchant: enchant}
	player := &proto.Player{
		Name: "Stone test", Race: proto.Race_RaceOrc, Class: proto.Class_ClassWarrior, Level: 60,
		Equipment: &proto.EquipmentSpec{Items: items},
		Rotation:  &proto.APLRotation{},
		Spec:      &proto.Player_Warrior{Warrior: &proto.Warrior{Options: &proto.Warrior_Options{}}},
	}
	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 60, Targets: []*proto.Target{bareTarget}},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	return sim.Raid.Parties[0].Players[0].GetCharacter()
}

// Blackfathom Sharpening Stone's temporary enchant 7098 "Hit +2%" is
// 430391 "Hit Weapon +2" (aura 54, 2), restricted by SpellEquippedItems to
// melee weapons (class 2, subclass mask 42483). Aura 54 is the melee and
// ranged hit chance, as the stones' aura 52 is their crit: weapon attacks
// gain 2% hit, spells and the stat sheet do not.
func TestBlackfathomSharpeningStoneHitReachesWeaponAttacksOnly(t *testing.T) {
	base, stoned := stonedWarrior(t, 0), stonedWarrior(t, blackfathomStoneEnchant)
	want := float64(2 * core.HitRatingPerHitChance)
	if got := stoned.AutoAttacks.MHAuto().BonusHitRating - base.AutoAttacks.MHAuto().BonusHitRating; got != want {
		t.Errorf("melee swing hit bonus %v, want %v", got, want)
	}
	if got := stoned.GetStat(stats.Hit) - base.GetStat(stats.Hit); got != 0 {
		t.Errorf("hit on the sheet +%v, want 0 (it would reach spells)", got)
	}
	for _, spell := range stoned.Spellbook {
		if spell.CastType == proto.CastType_CastTypeUnknown && spell.BonusHitRating != 0 {
			t.Errorf("spell %v gained %v hit", spell.ActionID, spell.BonusHitRating)
		}
	}
}
