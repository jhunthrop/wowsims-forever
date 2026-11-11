package retribution

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// Holy Strike's weapon percent (effect 31, 25% to 50% by rank) scales the
// normalized weapon damage only. The flat Holy bonus (effect 121's base
// points, 93 at rank 8) and the spell damage term (0.429 of spell damage)
// are added whole, as the client's own text has it: "An instant strike
// that causes 50% weapon damage plus an additional 81 to 105 as Holy
// damage."
//
// Live Forever logs settle it (sim-accuracy lane sim-paladins, 11 Nov
// 2026, design/reviews/2026-11-11-sim-accuracy/paladins.md): in the 8 Nov
// advanced-logging report a level 30 to 35 Retribution paladin with
// Verigan's Fist landed 78 non-critical Holy Strike (rank 4, 36% weapon
// plus 24 to 25) hits of 106 on average against his own pre-armor white
// swings of 198 and Seal of Command procs of 145 (70% of weapon damage):
// 0.36 x 202 + 25 is 98 before his Judgement of the Crusader, 107 with it
// on a quarter of the hits; the old (202 + 25) x 0.36 is 82. Two paladins
// on foreverlogs.gg agree (Holy Strike 0.73 to 0.75 of a Seal of Command
// proc, against the old reading's 0.60).
//
// The expectation is built from the engine's own weapon and attack power,
// so a gear or stat change cannot move it; the sim's average landed hit
// (crits counted twice, the melee crit) must match it within 2%.
func TestHolyStrikeWeaponPercentLeavesTheFlatAndSpellDamageWhole(t *testing.T) {
	const (
		spellDamage       = 500.0
		holyStrikeRank8   = 10333
		rank8Flat         = 93.0
		rank8WeaponShare  = 0.50
		holyStrikeCoeff   = 0.429
		tolerance         = 0.02
		iterations        = 300
		fightSeconds      = 120
		meleeCritMultiple = 2.0
	)

	bonus := stats.Stats{}
	bonus[stats.SpellDamage] = spellDamage
	player := core.WithSpec(&proto.Player{
		Class:              proto.Class_ClassPaladin,
		Race:               proto.Race_RaceHuman,
		Level:              60,
		Equipment:          &proto.EquipmentSpec{Items: []*proto.ItemSpec{{}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {Id: sorTestTwoHanderID}}},
		BonusStats:         &proto.UnitStats{Stats: bonus[:]},
		Rotation:           core.APLRotationFromJsonString(`{"type": "TypeAPL", "priorityList": [{"action": {"castSpell": {"spellId": {"spellId": 10333}}}}]}`),
		DistanceFromTarget: 5,
	}, PlayerOptionsSealofRighteousness)
	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})
	// A target at the paladin's own level: three levels up, the engine's
	// level-based partial resistance takes about 6% off a Holy hit, which
	// is a separate mechanic from the formula this test is about.
	target := core.NewDefaultTarget()
	target.Level = 60
	encounter := &proto.Encounter{Duration: fightSeconds, Targets: []*proto.Target{target}}

	env, _, _ := core.NewEnvironment(raid, encounter, false)
	character := env.Raid.Parties[0].Players[0].GetCharacter()
	spell := character.GetSpell(core.ActionID{SpellID: holyStrikeRank8})
	if spell == nil {
		t.Fatal("Holy Strike rank 8 is not registered at level 60")
	}
	weapon := character.AutoAttacks.MH()
	attackPower := spell.MeleeAttackPower(env.Encounter.TargetUnits[0])
	normalized := weapon.AverageDamage() + weapon.NormalizedSwingSpeed*attackPower/weapon.AttackPowerPerDPS
	want := rank8WeaponShare*normalized + rank8Flat + holyStrikeCoeff*spellDamage

	result := core.RunRaidSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  encounter,
		SimOptions: &proto.SimOptions{Iterations: iterations, RandomSeed: 1, IsTest: true},
	})
	if result.Error != nil {
		t.Fatalf("sim error: %s", result.Error.Message)
	}
	metrics := findActionMetrics(result, holyStrikeRank8)
	if metrics == nil {
		t.Fatal("Holy Strike never landed")
	}
	var damage, hits, crits float64
	for _, target := range metrics.Targets {
		damage += target.Damage
		hits += float64(target.Hits)
		crits += float64(target.Crits)
	}
	if hits+crits == 0 {
		t.Fatal("Holy Strike never landed")
	}
	got := damage / (hits + meleeCritMultiple*crits)
	if math.Abs(got-want)/want > tolerance {
		t.Errorf("Holy Strike rank 8 average hit %.1f, want %.1f (%.0f%% of %.1f normalized weapon damage + %.0f + %.3f x %.0f spell damage); the old reading, (%.1f + %.0f) x %.2f + %.3f x %.0f x %.2f, is %.1f",
			got, want, rank8WeaponShare*100, normalized, rank8Flat, holyStrikeCoeff, spellDamage,
			normalized, rank8Flat, rank8WeaponShare, holyStrikeCoeff, spellDamage, rank8WeaponShare,
			(normalized+rank8Flat)*rank8WeaponShare+holyStrikeCoeff*spellDamage*rank8WeaponShare)
	}
}
