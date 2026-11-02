package healing

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/healsim"
)

// Dormant Heart of the Mountain (249473) is a Forever trinket the fork's
// own item database lacks, so the request carries it, as the site's simdb
// does. Its equip spell 1249118 (aura 42, ProcChance 100, no recovery)
// triggers 1249119 "Eternal Power": aura 135 +40 healing and aura 13 +13
// damage, all schools, for 15 sec. Wowhead Forever's tooltip (read
// 2026-11-02): "Your casts of Greater Heal, Healing Touch, Healing Wave, or
// Holy Light in combat grant up to 40 increased healing and up to 13
// increased damage for 15 sec."
const (
	dormantHeartID      int32 = 249473
	eternalPowerSpellID int32 = 1249119
	greaterHealRank5    int32 = 25314
	flashHealRank7      int32 = 10917
)

var dormantHeartDatabase = &proto.SimDatabase{Items: []*proto.SimItem{{
	Id:            dormantHeartID,
	Name:          "Dormant Heart of the Mountain",
	Type:          proto.ItemType_ItemTypeTrinket,
	RequiredLevel: 50,
}}}

func castOnlyRotation(spellID int32) *proto.APLRotation {
	return &proto.APLRotation{PriorityList: []*proto.APLListItem{{Action: &proto.APLAction{
		Action: &proto.APLAction_CastSpell{CastSpell: &proto.APLActionCastSpell{
			SpellId: &proto.ActionID{RawId: &proto.ActionID_SpellId{SpellId: spellID}},
		}},
	}}}}
}

// eternalPowerUptime runs a holy priest casting only spellID with the
// trinket worn and returns the share of the fight Eternal Power was up.
func eternalPowerUptime(t *testing.T, spellID int32) float64 {
	t.Helper()
	player := healer(60, HolyTalents, &proto.HealingPriest_Options{}, castOnlyRotation(spellID))
	player.Database = dormantHeartDatabase
	items := make([]*proto.ItemSpec, proto.ItemSlot_ItemSlotTrinket1+1)
	for i := range items {
		items[i] = &proto.ItemSpec{}
	}
	items[proto.ItemSlot_ItemSlotTrinket1] = &proto.ItemSpec{Id: dormantHeartID}
	player.Equipment = &proto.EquipmentSpec{Items: items}
	result := core.RunRaidSim(healsim.Request(player, healsim.TestProfile(), rotationSeconds, 5))
	if result.Error != nil {
		t.Fatal(result.Error.Message)
	}
	metrics := result.RaidMetrics.Parties[0].Players[healsim.HealerIndex]
	if healsim.CastsOf(metrics, &proto.ActionID{RawId: &proto.ActionID_SpellId{SpellId: spellID}}) == 0 {
		t.Fatalf("the priest never cast %d", spellID)
	}
	for _, aura := range metrics.Auras {
		if aura.Id.GetSpellId() == eternalPowerSpellID {
			return aura.UptimeSecondsAvg / rotationSeconds
		}
	}
	return 0
}

func TestDormantHeartOfTheMountainGrantsEternalPowerOnGreaterHeal(t *testing.T) {
	if uptime := eternalPowerUptime(t, greaterHealRank5); uptime < 0.9 {
		t.Errorf("Eternal Power was up %.2f of a Greater Heal fight; every cast grants it for 15 sec", uptime)
	}
	if uptime := eternalPowerUptime(t, flashHealRank7); uptime != 0 {
		t.Errorf("Eternal Power was up %.2f of a Flash Heal fight; only the four named heals grant it", uptime)
	}
}
