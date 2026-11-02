package tank

import (
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/druid"
)

// Forever: "Rage potions and Thistletea are also now usable by Druids", and
// every potion is drunk in a form without cancelling it (the druid deep dive,
// news.blizzard.com article 24301515). A bear drinks its Mighty Rage Potion
// and stays in Dire Bear Form. The bear's published rotation casts no
// cooldowns of its own, so the test adds the autocast line.
func TestABearDrinksARagePotionInForm(t *testing.T) {
	rotation := bearRotation()
	rotation.PriorityList = append([]*proto.APLListItem{{
		Action: &proto.APLAction{Action: &proto.APLAction_AutocastOtherCooldowns{AutocastOtherCooldowns: &proto.APLActionAutocastOtherCooldowns{}}},
	}}, rotation.PriorityList...)
	player := bearPlayer(60, druid.ForeverBearTalents, loadGear(t), rotation)
	player.Consumes = &proto.Consumes{DefaultPotion: proto.Potions_MightyRagePotion}
	m := runHarness(t, player, 20)

	casts := int32(0)
	for _, action := range m.Actions {
		if action.Id.GetItemId() == 13442 {
			for _, target := range action.Targets {
				casts += target.Casts
			}
		}
	}
	if casts == 0 {
		t.Error("the bear never drank its Mighty Rage Potion")
	}
	for _, aura := range m.Auras {
		if aura.Id.GetSpellId() == 9634 && aura.UptimeSecondsAvg < harnessDuration-1 {
			t.Errorf("Dire Bear Form was up %.1f s of %.0f: the potion broke the form", aura.UptimeSecondsAvg, harnessDuration)
		}
	}
}
