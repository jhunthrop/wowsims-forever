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
// summoned demon then inherits from. Improved Shadow Bolt 5/5 (Destruction
// field 2) is there for the Shadow Vulnerability test.
const petInheritanceDemoTalents = "-00050000300000003-05"

func newPetInheritanceTestWarlock(t *testing.T, summon proto.WarlockOptions_Summon) (*core.Simulation, *Warlock) {
	t.Helper()
	return newPetTestWarlockWithTalents(t, summon, petInheritanceDemoTalents)
}

func newPetTestWarlockWithTalents(t *testing.T, summon proto.WarlockOptions_Summon, talents string) (*core.Simulation, *Warlock) {
	t.Helper()
	player := core.WithSpec(
		&proto.Player{
			Class:         proto.Class_ClassWarlock,
			Race:          proto.Race_RaceOrc,
			Level:         60,
			Equipment:     &proto.EquipmentSpec{},
			TalentsString: talents,
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
// hit, with server-computed amounts. Forever's own character sheet states
// the spell-damage, armor and resistance shares: WARLOCK_PET_BONUS in the
// 1.60.1 client's Camelot PaperDollFrame.lua (PET_BONUS_SPELLDMG_TO_AP 0.17,
// PET_BONUS_SPELLDMG_TO_SPELLDMG 0.1, PET_BONUS_ARMOR 0.35, PET_BONUS_RES
// 1.00), and wowsims/forever read the same tenth off beta pet sheets and a
// logged Sayaad (c3d9ea951). Stamina and Intellect keep Classic's shares.
func TestDemonInheritsForeversShares(t *testing.T) {
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
		stats.Stamina:          80, // 2 health a point: Blizzard, 3 Nov notes
		stats.Intellect:        90,
		stats.Armor:            700,
		stats.FireResistance:   100,
		stats.ShadowResistance: 50,
		stats.AttackPower:      0.17 * 400,
		stats.SpellDamage:      0.10 * 400,
		stats.Hit:              6 * 8.0 / 17 * core.HitRatingPerHitChance,
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
	if math.Abs(gain[stats.SpellDamage]-8) > 1e-9 || math.Abs(loss[stats.SpellDamage]+8) > 1e-9 {
		t.Errorf("shadow power +80/-80 moves the demon's spell damage by %.2f/%.2f, want +8/-8", gain[stats.SpellDamage], loss[stats.SpellDamage])
	}
}

// A summoned Succubus follows the warlock's sheet: Demonic Knowledge's 60
// spell power gives it 0.17 x 60 = 10.2 attack power on its own 129 x 2 -
// 20 = 238, and a fifth of the warlock's Stamina: Blizzard's 3 Nov notes,
// "Pets now gain 2 Health per point of Master's Stamina (was 7.5 Health per
// point of Master's Stamina)", at 10 health a point of the demon's Stamina.
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
	if want := 238 + 0.17*60; math.Abs(got[stats.AttackPower]-want) > 1e-6 {
		t.Errorf("Succubus attack power = %.2f, want %.2f", got[stats.AttackPower], want)
	}
	if want := 234 + 0.2*owner[stats.Stamina]; math.Abs(got[stats.Stamina]-want) > 1e-6 {
		t.Errorf("Succubus Stamina = %.2f, want 234 + 20%% of %.1f = %.2f", got[stats.Stamina], owner[stats.Stamina], want)
	}
	if want := 0.10 * 60; math.Abs(got[stats.SpellDamage]-want) > 1e-6 {
		t.Errorf("Succubus spell damage = %.2f, want %.2f", got[stats.SpellDamage], want)
	}
}

// Hit (forever-bugs #394, AidanZMoon for Blizzard, 2026-10-20): "Warlocks
// will give spell hit equal to their spell hit, and melee hit equal to
// 8/17ths of their spell hit." The demon's melee carries 8/17 of the
// warlock's hit; its spells top that up to the whole of it.
func TestDemonSpellsTakeTheWarlocksWholeHitAndMeleeEightSeventeenths(t *testing.T) {
	sim, warlock := newPetInheritanceTestWarlock(t, proto.WarlockOptions_Succubus)
	succubus := warlock.Succubus
	ownerHit := 5 * float64(core.HitRatingPerHitChance)
	warlock.AddStatDynamic(sim, stats.Hit, ownerHit-warlock.GetStat(stats.Hit))

	if got, want := succubus.GetStat(stats.Hit), ownerHit*8/17; math.Abs(got-want) > 1e-6 {
		t.Errorf("Succubus melee hit rating = %.3f, want 8/17 of the warlock's %.3f = %.3f", got, ownerHit, want)
	}
	lash := succubus.primaryAbility
	succubus.applyOwnerSpellHit(lash)
	if got, want := lash.SpellHitChance(sim.Encounter.TargetUnits[0]), 0.05; math.Abs(got-want) > 1e-9 {
		t.Errorf("Lash of Pain hit chance = %.4f, want the warlock's whole %.4f", got, want)
	}
}

// Malevolence (1310949) is aura 107 crit chance, 5% at 5/5, on a class mask
// that carries Lash of Pain's bit (0x2000) beside the warlock's own Shadow
// spells, as Improved Sayaad's mask carries it: the Succubus's Lash takes it.
func TestLashOfPainTakesMalevolence(t *testing.T) {
	_, without := newPetTestWarlockWithTalents(t, proto.WarlockOptions_Succubus, "")
	_, with := newPetTestWarlockWithTalents(t, proto.WarlockOptions_Succubus, "00000000005")
	if with.Talents.Malevolence != 5 {
		t.Fatalf("Malevolence = %d, want 5", with.Talents.Malevolence)
	}
	got := with.Succubus.primaryAbility.BonusCritRating - without.Succubus.primaryAbility.BonusCritRating
	if want := 5 * float64(core.CritRatingPerCritChance); math.Abs(got-want) > 1e-9 {
		t.Errorf("Lash of Pain bonus crit rating rose by %.3f under Malevolence 5/5, want %.3f", got, want)
	}
}
