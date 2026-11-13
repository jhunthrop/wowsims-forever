package shaman

import (
	"math"
	"testing"
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage/clientdamagetest"
	"github.com/wowsims/classic/sim/core/spellconst"
)

// Blizzard's 10 November Enhancement hotfix (t/2403040) as push 113722
// states it in the 1.60.1.70938 data: the engine's Stormstrike, Maelstrom
// Weapon and Mental Dexterity are the vendored client rows.

func clientSpell(t *testing.T, class spellconst.Class, id int32) spellconst.Spell {
	t.Helper()
	spell, ok := class.ByID(id)
	if !ok {
		t.Fatalf("client spell %d is not in %s", id, clientShamanSpellconst)
	}
	return spell
}

func clientEffect(t *testing.T, spell spellconst.Spell, index int) spellconst.Effect {
	t.Helper()
	for _, effect := range spell.Effects {
		if effect.Index == index {
			return effect
		}
	}
	t.Fatalf("client spell %d has no effect %d", spell.ID, index)
	return spellconst.Effect{}
}

func TestStormstrikeIsTheClientRow(t *testing.T) {
	stormstrike := clientSpell(t, clientdamagetest.Load(t, clientShamanSpellconst), 17364)
	if got, want := stormstrikeCooldown(), time.Duration(stormstrike.CooldownMS)*time.Millisecond; got != want {
		t.Errorf("Stormstrike's cooldown = %v, want the client's %v", got, want)
	}
	if got, want := stormstrikeBaseManaCost(), stormstrike.CostPct/100; math.Abs(got-want) > 1e-9 {
		t.Errorf("Stormstrike costs %.3f of base mana, want the client's %.3f", got, want)
	}
	if got, want := stormstrikeWeaponDamagePercent, clientEffect(t, stormstrike, 2).Amount/100; math.Abs(got-want) > 1e-9 {
		t.Errorf("Stormstrike strikes for %.2f weapon damage, want the client's %.2f", got, want)
	}
	if got, want := stormstrikeChargeBonus-1, clientEffect(t, stormstrike, 1).Amount/100; math.Abs(got-want) > 1e-9 {
		t.Errorf("Stormstrike's charge adds %.2f, want the client's %.2f", got, want)
	}
	if got, want := stormstrikeChargeDuration, time.Duration(stormstrike.DurationMS)*time.Millisecond; got != want {
		t.Errorf("Stormstrike's charge lasts %v, want the client's %v", got, want)
	}
}

func TestMaelstromWeaponIsTheClientRow(t *testing.T) {
	class := clientdamagetest.Load(t, clientShamanSpellconst)
	talent := clientSpell(t, class, 408498)
	stack := clientSpell(t, class, 408505)
	if got, want := maelstromWeaponPerStackPercent, int32(clientEffect(t, talent, 0).Amount); got != want {
		t.Errorf("Maelstrom Weapon takes %d%% a stack, want the client's %d%%", got, want)
	}
	for _, index := range []int{0, 1} { // cast time (misc 10) and cost (misc 14)
		if got, want := -maelstromWeaponPerStackPercent, int32(clientEffect(t, stack, index).Amount); got != want {
			t.Errorf("Maelstrom Weapon's stack effect %d = %d, want the client's %d", index, got, want)
		}
	}
	if got, want := maelstromWeaponMaxStacks, int32(clientEffect(t, talent, 2).Amount); got != want {
		t.Errorf("Maelstrom Weapon stacks to %d, want the client's %d", got, want)
	}
	if got, want := maelstromWeaponDuration, time.Duration(stack.DurationMS)*time.Millisecond; got != want {
		t.Errorf("Maelstrom Weapon lasts %v, want the client's %v", got, want)
	}
}

func TestMentalDexterityIsTheClientRow(t *testing.T) {
	talent := clientSpell(t, clientdamagetest.Load(t, clientShamanSpellconst), 415140)
	top := mentalDexterityAttackPowerPercent[len(mentalDexterityAttackPowerPercent)-1]
	if want := clientEffect(t, talent, 0).Amount / 100; math.Abs(top-want) > 1e-9 {
		t.Errorf("Mental Dexterity 3/3 gives %.2f attack power per Intellect, want the client's %.2f", top, want)
	}
}
