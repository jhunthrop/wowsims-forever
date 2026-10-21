package core

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// The Flametongue Totem is read from the 1.60.1.70009 client
// (data/builds/1.60.1.70009/raw):
//
//	Spell 8227/8249/10526/16387 (ranks 1-4, spell levels 28/38/48/58, 5 min):
//	  effect 28 summon, misc 5950/6012/7423/10557; description "The totem
//	  enhances the melee attacks of all party members within $8250a1 yards.
//	  Each main hand hit causes ${$8248m1/77*$<mult>-1} to
//	  ${$8248M1/25*$<mult>} additional Fire damage, based on the speed of the
//	  weapon."
//	Area aura 8230/8250/10521/15036: effect 35 (apply party aura), aura 42
//	  (proc trigger), ProcChance 100, ProcTypeMask_0 4 (melee auto attack),
//	  radius index 10 (30 yards); its trigger is the proc spell.
//	Proc spell 8253/8248/10523/16389: effect 3, EffectBasePointsF 548 / 781 /
//	  1061 / 1363, no per-level growth, no variance, no coefficient.
//
// The weapon-speed scaling is the description's own range: m1/77 to m1/25
// is the amount per second of weapon speed over 1.3 to 4.0 seconds, so a
// hit deals base points x speed / 100. The Flametongue Weapon proc states
// the same text, and the engine's weapon imbue (rank 6: 2498 + 78 per level
// above 56 = 2810 at 60, 28.1 a second, against its 112 / 4 = 28) agrees.
func TestFlametongueTotemRanksMatchClient(t *testing.T) {
	want := BuffRanks{
		{SpellID: 8227, Level: 28, Amount: 548},
		{SpellID: 8249, Level: 38, Amount: 781},
		{SpellID: 10526, Level: 48, Amount: 1061},
		{SpellID: 16387, Level: 58, Amount: 1363},
	}
	if len(FlametongueTotemRanks) != len(want) {
		t.Fatalf("Flametongue Totem has %d ranks, want %d", len(FlametongueTotemRanks), len(want))
	}
	for i, rank := range want {
		if FlametongueTotemRanks[i] != rank {
			t.Errorf("rank %d = %+v, want %+v", i+1, FlametongueTotemRanks[i], rank)
		}
	}
	wantProc := []int32{8253, 8248, 10523, 16389}
	for i, id := range wantProc {
		if FlametongueTotemProcSpellIDs[i] != id {
			t.Errorf("rank %d proc spell = %d, want %d", i+1, FlametongueTotemProcSpellIDs[i], id)
		}
	}
}

func TestFlametongueTotemHitScalesWithWeaponSpeed(t *testing.T) {
	cases := []struct {
		level int
		speed float64
		want  float64
	}{
		{60, 2.6, 1363 * 2.6 / 100},
		{60, 1.3, 1363 * 1.3 / 100},
		{60, 4.0, 1363 * 4.0 / 100},
		{50, 2.0, 1061 * 2.0 / 100},
		{38, 1.8, 781 * 1.8 / 100},
		{28, 3.0, 548 * 3.0 / 100},
	}
	for _, c := range cases {
		got, ok := FlametongueTotemHitDamage(c.level, c.speed)
		if !ok || math.Abs(got-c.want) > 1e-9 {
			t.Errorf("level %d speed %.1f: damage = %v (%v), want %v", c.level, c.speed, got, ok, c.want)
		}
	}
}

func TestFlametongueTotemIsNotLearnedBeforeLevel28(t *testing.T) {
	if _, ok := FlametongueTotemHitDamage(27, 2.0); ok {
		t.Error("a level 27 character receives Flametongue Totem")
	}
}

// Flametongue Weapon on the main hand "disables any benefit you personally
// receive from Flametongue Totem" (spells 8024 to 16342), and a feral druid
// has no main-hand weapon; everyone else is reached, including a shaman
// whose main hand carries Windfury Weapon.
func TestFlametongueTotemReaches(t *testing.T) {
	cases := []struct {
		name      string
		character Character
		want      bool
	}{
		{"no consumables", Character{}, true},
		{"poison on the main hand", Character{Consumes: &proto.Consumes{MainHandImbue: proto.WeaponImbue_InstantPoison}}, true},
		{"flametongue weapon on the main hand", Character{Consumes: &proto.Consumes{MainHandImbue: proto.WeaponImbue_FlametongueWeapon}}, false},
		{"flametongue weapon on the off hand only", Character{Consumes: &proto.Consumes{OffHandImbue: proto.WeaponImbue_FlametongueWeapon}}, true},
		{"windfury weapon on the main hand", Character{Consumes: &proto.Consumes{MainHandImbue: proto.WeaponImbue_WindfuryWeapon}}, true},
		{"feral", Character{Unit: Unit{PseudoStats: stats.PseudoStats{FeralCombatEnabled: true}}}, false},
	}
	for _, c := range cases {
		if got := flametongueTotemReaches(&c.character); got != c.want {
			t.Errorf("%s: flametongueTotemReaches = %v, want %v", c.name, got, c.want)
		}
	}
}
