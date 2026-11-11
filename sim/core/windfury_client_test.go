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

// Blizzard's 10 November hotfix (t/2403040, Shaman): "Weapon imbues and
// Weapon totems can now be used at the same time ... the Shaman can benefit
// from Windfury Weapon and Windfury Totem at the same time, granting their
// attacks a chance to activate both." Every character is reached.
func TestWindfuryTotemReaches(t *testing.T) {
	cases := []struct {
		name      string
		character Character
		want      bool
	}{
		{"no consumables", Character{}, true},
		{"poison on the main hand", Character{Consumes: &proto.Consumes{MainHandImbue: proto.WeaponImbue_InstantPoison}}, true},
		{"windfury weapon on the main hand", Character{Consumes: &proto.Consumes{MainHandImbue: proto.WeaponImbue_WindfuryWeapon}}, true},
		{"windfury weapon on the off hand only", Character{Consumes: &proto.Consumes{OffHandImbue: proto.WeaponImbue_WindfuryWeapon}}, true},
		{"feral (Forever: weapon procs work in a form)", Character{Unit: Unit{PseudoStats: stats.PseudoStats{FeralCombatEnabled: true}}}, true},
	}
	for _, c := range cases {
		if got := windfuryTotemReaches(&c.character); got != c.want {
			t.Errorf("%s: windfuryTotemReaches = %v, want %v", c.name, got, c.want)
		}
	}
}
