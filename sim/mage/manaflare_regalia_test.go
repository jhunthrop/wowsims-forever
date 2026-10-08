package mage

import (
	"testing"
	"time"

	_ "github.com/wowsims/classic/sim/common" // imported to get item effects included.
	"github.com/wowsims/classic/sim/common/clientsetbonus/clientsetbonustest"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

func newManaflareMage(t *testing.T, pieces int, talents string) (*core.Simulation, *Mage) {
	return newManaflareMageAgainst(t, pieces, talents, bossTarget())
}

// newManaflareMageAgainst wears pieces of the set (synthetic, zero-stat
// pieces carrying the client's set id) and fights target.
func newManaflareMageAgainst(t *testing.T, pieces int, talents string, target *proto.Target) (*core.Simulation, *Mage) {
	t.Helper()
	if !core.WITH_DB {
		t.Skip("needs the item database (--tags=with_db)")
	}
	equipment, database := core.ClientSetTestGear(manaflareRegaliaSetID, pieces)
	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassMage,
			Race:               proto.Race_RaceTroll,
			Level:              60,
			Equipment:          equipment,
			Database:           database,
			Buffs:              core.FullBuffs.Player,
			TalentsString:      talents,
			DistanceFromTarget: 5,
		},
		PlayerOptions,
	)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 60, Targets: []*proto.Target{target}},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()
	return sim, sim.Raid.Parties[0].Players[0].(MageAgent).GetMage()
}

func combustionTalents(t *testing.T) string {
	t.Helper()
	return talentStringWithRank(t, ForeverFrostTalents, "combustion", 1)
}

// 2P (1300947) and 4P (1301079) are flat bonuses applied from the client's
// rows: hit, and spell damage against Elementals.
func TestManaflareFlatBonusesMatchTheRows(t *testing.T) {
	_, bare := newManaflareMage(t, 0, ForeverFrostTalents)
	for _, pieces := range []int{2, 4} {
		_, worn := newManaflareMage(t, pieces, ForeverFrostTalents)
		clientsetbonustest.AssertAutomaticTotals(t, manaflareRegaliaSetID, pieces, bare.GetCharacter(), worn.GetCharacter())
	}
}

// 4P (1301079): "Increases damage done by your spells and effects by up to
// 21 when fighting Elementals". Live only against an Elemental target.
func TestManaflareFourPieceAddsSpellDamageOnlyAgainstElementals(t *testing.T) {
	row, _ := core.DecodeClientFlatBonus(core.MustClientSpellRow(1301079))
	want := row.SpellDamageVs[0].Amount
	elemental := &proto.Target{Level: 60, Stats: core.DefaultTargetProtoLvl60.Stats, MobType: proto.MobType_MobTypeElemental}

	cases := []struct {
		name   string
		pieces int
		target *proto.Target
		want   float64
	}{
		{"four pieces against an Elemental", 4, elemental, want},
		{"three pieces against an Elemental", 3, elemental, 0},
		{"four pieces against the default target", 4, bossTarget(), 0},
	}
	for _, tc := range cases {
		sim, mage := newManaflareMageAgainst(t, tc.pieces, ForeverFrostTalents, tc.target)
		at := mage.AttackTables[sim.Encounter.AllTargetUnits[0].UnitIndex][proto.CastType_CastTypeMainHand]
		if at.BonusSpellDamageTaken != tc.want {
			t.Errorf("%s: spell damage bonus %v, want %v", tc.name, at.BonusSpellDamageTaken, tc.want)
		}
	}
}

// 3P (1301013): "Reduces the cooldown on your Counterspell spell by 5 sec".
func TestManaflareThreePieceShortensCounterspell(t *testing.T) {
	_, two := newManaflareMage(t, 2, ForeverFrostTalents)
	_, three := newManaflareMage(t, 3, ForeverFrostTalents)
	want := -time.Duration(core.MustClientSpellRow(manaflareCounterspellBonusSpell).Effects[0].Points) * time.Millisecond
	if got := two.Counterspell.CD.Duration - three.Counterspell.CD.Duration; got != want {
		t.Errorf("three pieces shorten Counterspell by %v, the row says %v", got, want)
	}
}

// 5P (1301488): Frostfire Bolt "has a 10% increased chance to trigger
// Missile Barrage, gains 10% increased critical strike chance while your
// Combustion spell is active, and has a 10% increased chance to trigger
// Fingers of Frost". Read as percentage points added to the chance.
func TestManaflareFivePieceRaisesFrostfireBoltProcChances(t *testing.T) {
	_, four := newManaflareMage(t, 4, ForeverFrostTalents)
	_, five := newManaflareMage(t, 5, ForeverFrostTalents)
	ffb := five.FrostfireBolt[FrostfireBoltRanks]
	bolt := five.Frostbolt[FrostboltRanks-1]

	cases := []struct {
		name string
		got  float64
		want float64
	}{
		{"FoF from Frostfire Bolt", five.fingersOfFrostChance(ffb), 0.25},
		{"FoF from Frostbolt", five.fingersOfFrostChance(bolt), 0.15},
		{"Missile Barrage from Frostfire Bolt", five.missileBarrageChance(ffb), 0.30},
		{"Missile Barrage from Frostbolt", five.missileBarrageChance(bolt), 0.20},
		{"FoF from Frostfire Bolt at four pieces", four.fingersOfFrostChance(four.FrostfireBolt[FrostfireBoltRanks]), 0.15},
		{"Missile Barrage from Frostfire Bolt at four pieces", four.missileBarrageChance(four.FrostfireBolt[FrostfireBoltRanks]), 0.20},
	}
	for _, tc := range cases {
		if tc.got < tc.want-1e-9 || tc.got > tc.want+1e-9 {
			t.Errorf("%s: %v, want %v", tc.name, tc.got, tc.want)
		}
	}
}

func TestManaflareFivePieceAddsFrostfireBoltCritWhileCombustionIsActive(t *testing.T) {
	sim4, four := newManaflareMage(t, 4, combustionTalents(t))
	sim5, five := newManaflareMage(t, 5, combustionTalents(t))

	idle4 := four.FrostfireBolt[FrostfireBoltRanks].BonusCritRating
	idle5 := five.FrostfireBolt[FrostfireBoltRanks].BonusCritRating
	if idle4 != idle5 {
		t.Fatalf("the set changes Frostfire Bolt crit without Combustion: %v vs %v", idle4, idle5)
	}

	four.CombustionAura.Activate(sim4)
	five.CombustionAura.Activate(sim5)
	want := float64(10 * core.CritRatingPerCritChance)
	got := five.FrostfireBolt[FrostfireBoltRanks].BonusCritRating - four.FrostfireBolt[FrostfireBoltRanks].BonusCritRating
	if got < want-1e-9 || got > want+1e-9 {
		t.Errorf("Combustion adds %v extra Frostfire Bolt crit rating at five pieces, want %v", got, want)
	}
	if four.Fireball[FireballRanks-1].BonusCritRating != five.Fireball[FireballRanks-1].BonusCritRating {
		t.Error("the Frostfire Bolt bonus leaked onto Fireball")
	}

	five.CombustionAura.Deactivate(sim5)
	if five.FrostfireBolt[FrostfireBoltRanks].BonusCritRating != idle5 {
		t.Error("the Combustion bonus stayed after Combustion ended")
	}
}
