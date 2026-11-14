package feral

import (
	"testing"

	_ "github.com/wowsims/classic/sim/common"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// Forever's druid deep dive (news.blizzard.com article 24301515, 2026-09-30):
// "All potions and almost all similar consumables are now useable while
// shapeshifted and do not cancel the Druid's shapeshift. Rage potions and
// Thistletea are also now usable by Druids." and "Weapon enchantments and
// procs now all work while shapeshifted, including effects like
// weightstones, Crusader enchant, and Windfury Totem." Blizzard fixed the
// last case that refused a mana potion in Cat Form (forever-bugs #109,
// 2026-10-08: "We can drink mana and health potions in forms now").
func TestACatUsesItsConsumablesWithoutLeavingCatForm(t *testing.T) {
	rotation := core.GetAplRotation("../../../ui/feral_druid/apls", "forever_feral")
	const duration = 180

	player := core.WithSpec(&proto.Player{
		Class:         proto.Class_ClassDruid,
		Race:          proto.Race_RaceTauren,
		Level:         60,
		Equipment:     &proto.EquipmentSpec{},
		TalentsString: P1Talents,
		Rotation:      rotation.Rotation,
		Consumes: &proto.Consumes{
			DefaultPotion:   proto.Potions_MajorFrenzyPotion,
			DefaultConjured: proto.Conjured_ConjuredRogueThistleTea,
			MainHandImbue:   proto.WeaponImbue_ElementalSharpeningStone,
		},
		DistanceFromTarget: 5,
	}, PlayerOptionsMonoCat)

	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{WindfuryTotem: true}, &proto.Debuffs{})
	result := core.RunRaidSim(&proto.RaidSimRequest{
		Raid:      raid,
		Encounter: &proto.Encounter{Duration: duration, Targets: []*proto.Target{core.NewDefaultTarget()}},
		SimOptions: &proto.SimOptions{
			Iterations: 20,
			RandomSeed: 1,
			IsTest:     true,
		},
	})
	if result.Error != nil {
		t.Fatalf("sim error: %s", result.Error.Message)
	}

	metrics := result.RaidMetrics.Parties[0].Players[0]
	casts := map[int32]int32{}
	for _, action := range metrics.Actions {
		for _, target := range action.Targets {
			casts[action.Id.GetItemId()] += target.Casts
		}
	}
	if casts[250943] == 0 {
		t.Error("the cat never drank its Major Frenzy Potion")
	}
	if casts[7676] == 0 {
		t.Error("the cat never drank Thistle Tea")
	}

	uptime := map[int32]float64{}
	for _, aura := range metrics.Auras {
		uptime[aura.Id.GetSpellId()] = aura.UptimeSecondsAvg
	}
	if got := uptime[768]; got < duration-1 {
		t.Errorf("Cat Form was up %.1f s of %d: a consumable broke the form", got, duration)
	}
	var windfuryStrikes int32
	for _, action := range metrics.Actions {
		if action.Id.GetSpellId() == core.WindfuryStrikeSpellId[3] {
			for _, target := range action.Targets {
				windfuryStrikes += target.Hits
			}
		}
	}
	if windfuryStrikes == 0 {
		t.Error("Windfury Totem never struck for the cat")
	}
}
