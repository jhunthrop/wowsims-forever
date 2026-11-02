package hunter

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
)

const scopeTestGun = 2098 // Double-barreled Shotgun

func buildScopedHunter(t *testing.T, enchant int32) *Hunter {
	t.Helper()
	if _, ok := core.EnchantsByEffectID[enchant]; !ok && enchant != 0 {
		// Forever's scopes are in the site's simdb only; the row carries no
		// stats (pipeline/simdb/enchants.py), the engine applies the bonus.
		core.EnchantsByEffectID[enchant] = core.Enchant{EffectID: enchant}
	}
	items := make([]*proto.ItemSpec, proto.ItemSlot_ItemSlotRanged+1)
	for i := range items {
		items[i] = &proto.ItemSpec{}
	}
	items[proto.ItemSlot_ItemSlotRanged] = &proto.ItemSpec{Id: scopeTestGun, Enchant: enchant}
	player := core.WithSpec(
		&proto.Player{
			Class:     proto.Class_ClassHunter,
			Race:      proto.Race_RaceOrc,
			Level:     60,
			Equipment: &proto.EquipmentSpec{Items: items},
			Buffs:     &proto.IndividualBuffs{},
		},
		&proto.Player_Hunter{Hunter: &proto.Hunter{Options: &proto.Hunter_Options{PetType: proto.Hunter_Options_PetNone}}},
	)
	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 60, Targets: []*proto.Target{core.DefaultTargetProtoLvl60}},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	return sim.Raid.Parties[0].Players[0].(HunterAgent).GetHunter()
}

// Forever's scopes put their bonus on the scoped weapon: SAF-T Ultra
// Precision Scope's enchant 8720 is 1310308 "Ranged Crit Bonus +02" (aura
// 290, 2) and Poultry Precision Scope's 8057 is 22780 "Ranged Hit Bonus +3"
// (aura 54, 3), each with SpellEquippedItems class 2, subclass mask 262156
// (bow, gun, crossbow). Ranged attacks gain it; melee attacks and the stat
// sheet do not.
func TestForeverScopesReachRangedAttacksOnly(t *testing.T) {
	base := buildScopedHunter(t, 0)
	for _, tc := range []struct {
		enchant   int32
		crit, hit float64
	}{
		{8720, 2 * core.CritRatingPerCritChance, 0},
		{8057, 0, 3 * core.HitRatingPerHitChance},
	} {
		h := buildScopedHunter(t, tc.enchant)
		ranged := h.AutoAttacks.RangedAuto()
		baseRanged := base.AutoAttacks.RangedAuto()
		if got := ranged.BonusCritRating - baseRanged.BonusCritRating; got != tc.crit {
			t.Errorf("enchant %d: Auto Shot crit bonus %v, want %v", tc.enchant, got, tc.crit)
		}
		if got := ranged.BonusHitRating - baseRanged.BonusHitRating; got != tc.hit {
			t.Errorf("enchant %d: Auto Shot hit bonus %v, want %v", tc.enchant, got, tc.hit)
		}
		if got := h.AutoAttacks.MHAuto().BonusCritRating - base.AutoAttacks.MHAuto().BonusCritRating; got != 0 {
			t.Errorf("enchant %d: melee swing crit bonus %v, want 0", tc.enchant, got)
		}
		for _, s := range []stats.Stat{stats.Crit, stats.MeleeCrit, stats.SpellCrit, stats.Hit} {
			if got := h.GetStat(s) - base.GetStat(s); got != 0 {
				t.Errorf("enchant %d: %v on the sheet +%v, want 0", tc.enchant, s, got)
			}
		}
	}
}
