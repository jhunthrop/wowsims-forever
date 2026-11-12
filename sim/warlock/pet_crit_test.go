package warlock

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
)

// Forever demons crit at their own, measured rates (forever-bugs #331 and the
// foreverlogs.gg beta logs), not at a warrior's or mage's attribute rates.
// The sheets' Agility and Intellect differ by level, so a crit that scaled
// with either would not hold at every band.
//   - melee: 12 of 1,718 Succubus swings (0.70%) by a level-30 warlock on
//     level-30 Cave Yetis (#331, Fierywind, 2026-10-12), and Blizzard's reply
//     there, "Their melee crit is extremely low"; Agility does not raise it;
//   - Lash of Pain: 13 of 205 in the same test and 7 of 105 in the public logs
//     of warlocks without Malevolence, 6.4%; Blizzard: "their base spell crit
//     is around normal levels";
//   - Firebolt: 207 of 3,501 (5.9%) by 13 live Imps (foreverlogs.gg 2816 to
//     2853, 5 to 12 Nov, owners at 12 to 40, each Imp 3.6 to 7.7%); the beta
//     logs read 2.2% (53 of 2,383), and the sim-accuracy 2026-11-12 warlock
//     logs lane moved it to the live rate.
func TestDemonCritMatchesTheForeverLogs(t *testing.T) {
	for _, level := range []int32{20, 30, 40, 60} {
		warlock := newWarlockAtLevel(t, level)
		for _, pet := range warlock.BasePets {
			s := pet.GetStats()
			melee := (s[stats.Crit] + s[stats.MeleeCrit]) / core.CritRatingPerCritChance
			spell := (s[stats.Crit] + s[stats.SpellCrit]) / core.CritRatingPerCritChance
			if pet != warlock.Imp && math.Abs(melee-0.7) > 1e-9 {
				t.Errorf("level %d %s melee crit = %.2f%%, want 0.70%%", level, pet.Name, melee)
			}
			wantSpell := map[*WarlockPet]float64{warlock.Imp: 5.9, warlock.Succubus: 6.4}[pet]
			if wantSpell != 0 && math.Abs(spell-wantSpell) > 1e-9 {
				t.Errorf("level %d %s spell crit = %.2f%%, want %.2f%%", level, pet.Name, spell, wantSpell)
			}
		}
	}
}

// A Forever demon takes the critical strike its warlock wears and nothing
// else of the warlock's crit. Blizzard on forever-bugs #331 (AidanZMoon,
// 2026-10-14): "Warlock pets currently only scale with their master's +%
// critical strike gear and have erroneous base crit"; the full mirror of the
// warlock's spell crit waits until that base crit is removed. #371 (crit
// racials missing from the pet sheet) was closed as its duplicate on
// 2026-10-16, and no note since enables the mirror. So a Human warlock with
// 6% crit on his gear, under both druid crit auras, gives his demons the 6%:
// not Ruthless's 1%, not the raid's 3% (the auras also cannot target a demon,
// SpellMisc Attributes_3 0x100).
func TestDemonTakesOnlyTheWarlocksGearCrit(t *testing.T) {
	const gearCrit = 6.0
	gear := stats.Stats{stats.Crit: gearCrit * core.CritRatingPerCritChance}
	for _, summon := range []proto.WarlockOptions_Summon{proto.WarlockOptions_Succubus, proto.WarlockOptions_Imp} {
		player := core.WithSpec(
			&proto.Player{
				Class:      proto.Class_ClassWarlock,
				Race:       proto.Race_RaceHuman,
				Level:      60,
				Equipment:  &proto.EquipmentSpec{},
				BonusStats: &proto.UnitStats{Stats: gear[:]},
			},
			&proto.Player_Warlock{Warlock: &proto.Warlock{Options: &proto.WarlockOptions{
				Armor:  proto.WarlockOptions_NoArmor,
				Summon: summon,
			}}},
		)
		raidBuffs := &proto.RaidBuffs{LeaderOfThePack: true, MoonkinAura: true}
		raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, raidBuffs, &proto.Debuffs{})
		sim := core.NewSim(&proto.RaidSimRequest{
			Raid:       raid,
			Encounter:  &proto.Encounter{Duration: 60, Targets: []*proto.Target{core.DefaultTargetProtoLvl60}},
			SimOptions: &proto.SimOptions{Iterations: 1, RandomSeed: 1, IsTest: true},
		}, simsignals.CreateSignals())
		sim.Reset()
		sim.PrePull()
		for i := 0; i < 10 && sim.CurrentTime <= 0; i++ {
			sim.Step()
		}
		warlock := sim.Raid.Parties[0].Players[0].(WarlockAgent).GetWarlock()
		pet := warlock.ActivePet
		if pet == nil || !pet.IsEnabled() {
			t.Fatalf("%v: no demon summoned", summon)
		}
		s := pet.GetStats()
		melee := (s[stats.Crit] + s[stats.MeleeCrit]) / core.CritRatingPerCritChance
		spell := (s[stats.Crit] + s[stats.SpellCrit]) / core.CritRatingPerCritChance
		wantMelee, wantSpell := demonMeleeCritPercent+gearCrit, demonSpellCritPercent+gearCrit
		if pet == warlock.Imp {
			wantMelee, wantSpell = gearCrit, impSpellCritPercent+gearCrit
		}
		if math.Abs(melee-wantMelee) > 1e-9 || math.Abs(spell-wantSpell) > 1e-9 {
			t.Errorf("%s crit melee/spell = %.2f/%.2f%%, want %.2f/%.2f%% (own rate plus the warlock's 6%% gear crit only)",
				pet.Name, melee, spell, wantMelee, wantSpell)
		}
	}
}
