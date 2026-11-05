package shaman

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
)

func TestMaelstromWeaponChance(t *testing.T) {
	melee := &core.Spell{ProcMask: core.ProcMaskMeleeMHAuto}
	bolt := &core.Spell{ProcMask: core.ProcMaskSpellDamage, SpellCode: SpellCode_ShamanLightningBolt}
	shock := &core.Spell{ProcMask: core.ProcMaskSpellDamage, SpellCode: SpellCode_ShamanEarthShock}

	plain := &Shaman{}
	if got := plain.maelstromWeaponChance(melee); got != maelstromWeaponProcChance {
		t.Errorf("melee hit chance = %v, want %v", got, maelstromWeaponProcChance)
	}
	if got := plain.maelstromWeaponChance(bolt); got != 0 {
		t.Errorf("Lightning Bolt without Totem of the Storm = %v, want 0", got)
	}

	totem := &Shaman{LightningBoltMaelstromChance: maelstromWeaponProcChance * (1 - totemOfTheStormMaelstromReduction)}
	if got, want := totem.maelstromWeaponChance(bolt), maelstromWeaponProcChance*0.5; got != want {
		t.Errorf("Lightning Bolt with Totem of the Storm = %v, want half a melee hit's (%v)", got, want)
	}
	if got := totem.maelstromWeaponChance(melee); got != maelstromWeaponProcChance {
		t.Errorf("melee hit with the totem = %v, want it unchanged at %v", got, maelstromWeaponProcChance)
	}
	if got := totem.maelstromWeaponChance(shock); got != 0 {
		t.Errorf("Earth Shock with the totem = %v, want 0", got)
	}
}

// Blizzard's Forever notes of 3 November: "Maelstrom Weapon: Chance to
// trigger doubled." Before that push a two-hander gained a stack on about
// half its hits (beta report 2748: 51 stacks over 94 hits on a 3.2 s weapon;
// a 15 October forum post: five stacks "every 9 swings or so"), the dummy's
// 50. The client row still carries 50: the doubling is server side.
func TestMaelstromWeaponChanceIsDoubledByTheThirdNovemberPush(t *testing.T) {
	melee := &core.Spell{ProcMask: core.ProcMaskMeleeMHSpecial}
	bolt := &core.Spell{ProcMask: core.ProcMaskSpellDamage, SpellCode: SpellCode_ShamanLightningBolt}
	if got := (&Shaman{}).maelstromWeaponChance(melee); got != 1.0 {
		t.Errorf("melee hit chance = %v, want 1.0 (the client's 50, doubled)", got)
	}
	totem := &Shaman{LightningBoltMaelstromChance: maelstromWeaponProcChance * (1 - totemOfTheStormMaelstromReduction)}
	if got := totem.maelstromWeaponChance(bolt); got != 0.5 {
		t.Errorf("Lightning Bolt with Totem of the Storm = %v, want 0.5", got)
	}
}
