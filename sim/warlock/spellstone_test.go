package warlock

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

func newWarlockWithImbue(t *testing.T, level int32, imbue proto.WarlockOptions_WeaponImbue) *Warlock {
	t.Helper()
	player := core.WithSpec(
		&proto.Player{Class: proto.Class_ClassWarlock, Race: proto.Race_RaceOrc, Level: level, Equipment: &proto.EquipmentSpec{}},
		&proto.Player_Warlock{Warlock: &proto.Warlock{Options: &proto.WarlockOptions{
			Armor: proto.WarlockOptions_NoArmor, Summon: proto.WarlockOptions_NoSummon, WeaponImbue: imbue,
		}}},
	)
	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})
	env, _, _ := core.NewEnvironment(raid, &proto.Encounter{}, true)
	return env.Raid.Parties[0].Players[0].(WarlockAgent).GetWarlock()
}

// Forever's Spellstones are weapon enchants 8059, 8060 and 8061 (items 5522,
// 13602 and 13603, made at 36, 48 and 60), whose spells 1237162, 1237164 and
// 1237165 are aura 13 Shadow damage 14/17/21 and aura 65 cast speed 1/2/2%:
// no crit.
func TestSpellstoneIsShadowDamageAndCastSpeed(t *testing.T) {
	for _, tc := range []struct {
		level        int32
		shadow, cast float64
	}{{36, 14, 1.01}, {47, 14, 1.01}, {48, 17, 1.02}, {59, 17, 1.02}, {60, 21, 1.02}} {
		bare := newWarlockWithImbue(t, tc.level, proto.WarlockOptions_NoWeaponImbue)
		stone := newWarlockWithImbue(t, tc.level, proto.WarlockOptions_Spellstone)
		if got := stone.GetStat(stats.ShadowPower) - bare.GetStat(stats.ShadowPower); got != tc.shadow {
			t.Errorf("level %d Spellstone Shadow damage = %.0f, want %.0f", tc.level, got, tc.shadow)
		}
		for _, s := range []stats.Stat{stats.Crit, stats.SpellCrit, stats.MeleeCrit} {
			if got := stone.GetStat(s) - bare.GetStat(s); got != 0 {
				t.Errorf("level %d Spellstone adds %s %.1f, want none", tc.level, s.StatName(), got)
			}
		}
		if got := bare.CastSpeed / stone.CastSpeed; math.Abs(got-tc.cast) > 1e-9 {
			t.Errorf("level %d Spellstone cast speed x%.4f, want x%.2f", tc.level, got, tc.cast)
		}
	}
}
