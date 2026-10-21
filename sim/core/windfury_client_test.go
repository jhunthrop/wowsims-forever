package core

import (
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
	"testing"
	"time"
)

// The Windfury Totem buff is read from the 1.60.1.70009 client
// (spellconst/shaman.json, spells 8516/10608/10610): attack power 95, 179
// and 246 for 1 second, one extra attack. Vanilla Classic's 122/229/315
// for 1.5 seconds is not this build's.
func TestWindfuryTotemBuffMatchesClient(t *testing.T) {
	want := [WindfuryRanks + 1]float64{0, 95, 179, 246}
	if WindfuryBuffBonusAP != want {
		t.Errorf("Windfury Totem attack power by rank = %v, want %v", WindfuryBuffBonusAP, want)
	}
	if WindfuryBuffDuration != time.Second {
		t.Errorf("Windfury Totem buff duration = %v, want 1s", WindfuryBuffDuration)
	}
}

// The totem's proc aura states a 100 ms ProcCategoryRecovery (spells 8515
// and 10612 in SpellAuraOptions); the engine's 1.5 s was vanilla Classic's.
func TestWindfuryTotemProcCooldownMatchesClient(t *testing.T) {
	if extraAttackProcICD != 100*time.Millisecond {
		t.Errorf("Windfury Totem proc cooldown = %v, want 100ms", extraAttackProcICD)
	}
}

// Windfury Weapon on the main hand "disables any benefit you personally
// receive from Windfury Totem" (spells 8232 to 16362), and a feral druid
// has no main-hand weapon to proc it; everyone else is reached.
func TestWindfuryTotemReaches(t *testing.T) {
	cases := []struct {
		name      string
		character Character
		want      bool
	}{
		{"no consumables", Character{}, true},
		{"poison on the main hand", Character{Consumes: &proto.Consumes{MainHandImbue: proto.WeaponImbue_InstantPoison}}, true},
		{"windfury weapon on the main hand", Character{Consumes: &proto.Consumes{MainHandImbue: proto.WeaponImbue_WindfuryWeapon}}, false},
		{"windfury weapon on the off hand only", Character{Consumes: &proto.Consumes{OffHandImbue: proto.WeaponImbue_WindfuryWeapon}}, true},
		{"feral", Character{Unit: Unit{PseudoStats: stats.PseudoStats{FeralCombatEnabled: true}}}, false},
	}
	for _, c := range cases {
		if got := windfuryTotemReaches(&c.character); got != c.want {
			t.Errorf("%s: windfuryTotemReaches = %v, want %v", c.name, got, c.want)
		}
	}
}
