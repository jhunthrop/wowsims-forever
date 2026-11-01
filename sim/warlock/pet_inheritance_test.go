package warlock

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
)

// petInheritanceDemoTalents sets Unholy Power 5/5 (Demonology field 3),
// Improved Sayaad 3/3 (field 8) and Demonic Knowledge 3/3 (field 16).
// Demonic Knowledge gives the warlock 60 spell power at level 60, which the
// summoned demon then inherits from.
const petInheritanceDemoTalents = "-00050000300000003"

func newPetInheritanceTestWarlock(t *testing.T, summon proto.WarlockOptions_Summon) (*core.Simulation, *Warlock) {
	t.Helper()
	player := core.WithSpec(
		&proto.Player{
			Class:         proto.Class_ClassWarlock,
			Race:          proto.Race_RaceOrc,
			Level:         60,
			Equipment:     &proto.EquipmentSpec{},
			TalentsString: petInheritanceDemoTalents,
		},
		&proto.Player_Warlock{Warlock: &proto.Warlock{Options: &proto.WarlockOptions{
			Armor:       proto.WarlockOptions_NoArmor,
			Summon:      summon,
			WeaponImbue: proto.WarlockOptions_NoWeaponImbue,
		}}},
	)
	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid: raid,
		Encounter: &proto.Encounter{
			Duration: 60,
			Targets:  []*proto.Target{core.DefaultTargetProtoLvl60},
		},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()
	for i := 0; i < 10 && sim.CurrentTime <= 0; i++ {
		sim.Step()
	}
	agent, ok := sim.Raid.Parties[0].Players[0].(WarlockAgent)
	if !ok {
		t.Fatal("player 0 did not build as a warlock")
	}
	return sim, agent.GetWarlock()
}

// Warlock Pet Scaling (client 416189) gives a demon channels for Stamina,
// Intellect, armor and the five resistances, attack power, spell damage and
// hit, with server-computed amounts. The amounts are Classic's, as wowsims'
// Season of Discovery engine carries them for the same client spell and
// warcraft.wiki.gg's Warlock minion page states them: 75% Stamina, 30%
// Intellect, 35% armor, 40% resistances (TBC's figure; the client has the
// channel), 56.5% of the warlock's best spell power as attack power and 15%
// of it as spell damage, and the warlock's hit.
func TestDemonInheritsTheClassicShares(t *testing.T) {
	_, warlock := newPetInheritanceTestWarlock(t, proto.WarlockOptions_Succubus)
	warlock.petSpellPowerSchool = stats.ShadowPower
	owner := stats.Stats{
		stats.Stamina:          400,
		stats.Intellect:        300,
		stats.Armor:            2000,
		stats.FireResistance:   100,
		stats.ShadowResistance: 50,
		stats.SpellPower:       300,
		stats.ShadowPower:      100,
		stats.FirePower:        40,
		stats.Hit:              6 * core.HitRatingPerHitChance,
		stats.Crit:             12 * core.CritRatingPerCritChance,
	}
	got := warlock.makeStatInheritance()(owner)
	want := stats.Stats{
		stats.Stamina:          300,
		stats.Intellect:        90,
		stats.Armor:            700,
		stats.FireResistance:   40,
		stats.ShadowResistance: 20,
		stats.AttackPower:      0.565 * 400,
		stats.SpellDamage:      0.15 * 400,
		stats.Hit:              6 * core.HitRatingPerHitChance,
	}
	for s := stats.Stat(0); s < stats.Len; s++ {
		if math.Abs(got[s]-want[s]) > 1e-9 {
			t.Errorf("inherited %s = %.3f, want %.3f", s.StatName(), got[s], want[s])
		}
	}
}

// The inheritance is linear, so the owner's mid-fight stat changes (a spell
// power trinket gained and lost) move the demon and then return it exactly.
func TestDemonInheritanceIsLinearForOwnerStatChanges(t *testing.T) {
	_, warlock := newPetInheritanceTestWarlock(t, proto.WarlockOptions_Succubus)
	warlock.petSpellPowerSchool = stats.ShadowPower
	inherit := warlock.makeStatInheritance()
	gain := inherit(stats.Stats{stats.ShadowPower: 80})
	loss := inherit(stats.Stats{stats.ShadowPower: -80})
	if gain[stats.SpellDamage] != 12 || loss[stats.SpellDamage] != -12 {
		t.Errorf("shadow power +80/-80 moves the demon's spell damage by %.2f/%.2f, want +12/-12", gain[stats.SpellDamage], loss[stats.SpellDamage])
	}
}

// A summoned Succubus follows the warlock's sheet: Demonic Knowledge's 60
// spell power gives it 0.565 x 60 = 33.9 attack power on its own 129 x 2 -
// 20 = 238, and three quarters of the warlock's Stamina.
func TestSummonedSuccubusFollowsTheWarlocksSheet(t *testing.T) {
	_, warlock := newPetInheritanceTestWarlock(t, proto.WarlockOptions_Succubus)
	succubus := warlock.Succubus
	if !succubus.IsEnabled() {
		t.Fatal("the Succubus is not summoned")
	}
	owner := warlock.GetStats()
	if math.Abs(owner[stats.SpellPower]-60) > 1e-6 {
		t.Fatalf("warlock spell power = %.1f, want Demonic Knowledge's 60", owner[stats.SpellPower])
	}
	got := succubus.GetStats()
	if want := 238 + 0.565*60; math.Abs(got[stats.AttackPower]-want) > 1e-6 {
		t.Errorf("Succubus attack power = %.2f, want %.2f", got[stats.AttackPower], want)
	}
	if want := 234 + 0.75*owner[stats.Stamina]; math.Abs(got[stats.Stamina]-want) > 1e-6 {
		t.Errorf("Succubus Stamina = %.2f, want 234 + 75%% of %.1f = %.2f", got[stats.Stamina], owner[stats.Stamina], want)
	}
	if want := 0.15 * 60; math.Abs(got[stats.SpellDamage]-want) > 1e-6 {
		t.Errorf("Succubus spell damage = %.2f, want %.2f", got[stats.SpellDamage], want)
	}
}
