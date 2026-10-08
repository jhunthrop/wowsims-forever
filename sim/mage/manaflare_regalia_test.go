package mage

import (
	"testing"
	"time"

	_ "github.com/wowsims/classic/sim/common" // imported to get item effects included.
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
)

// manaflarePieces is the Manaflare Regalia in slot order (client
// ItemSet 2098): crown, mantle, robes, gloves, pants, boots.
var manaflarePieces = []struct {
	slot proto.ItemSlot
	id   int32
}{
	{proto.ItemSlot_ItemSlotHead, 280455},
	{proto.ItemSlot_ItemSlotShoulder, 280454},
	{proto.ItemSlot_ItemSlotChest, 280450},
	{proto.ItemSlot_ItemSlotHands, 280451},
	{proto.ItemSlot_ItemSlotLegs, 280453},
	{proto.ItemSlot_ItemSlotFeet, 280452},
}

func newManaflareMage(t *testing.T, pieces int, talents string) (*core.Simulation, *Mage) {
	t.Helper()
	if !core.WITH_DB {
		t.Skip("needs the item database (--tags=with_db)")
	}
	items := make([]*proto.ItemSpec, 17)
	for _, piece := range manaflarePieces[:pieces] {
		items[piece.slot] = &proto.ItemSpec{Id: piece.id}
	}
	for i := range items {
		if items[i] == nil {
			items[i] = &proto.ItemSpec{}
		}
	}
	player := core.WithSpec(
		&proto.Player{
			Class:              proto.Class_ClassMage,
			Race:               proto.Race_RaceTroll,
			Level:              60,
			Equipment:          &proto.EquipmentSpec{Items: items},
			Buffs:              core.FullBuffs.Player,
			TalentsString:      talents,
			DistanceFromTarget: 5,
		},
		PlayerOptions,
	)
	raid := core.SinglePlayerRaidProto(player, core.FullBuffs.Party, core.FullBuffs.Raid, core.FullBuffs.Debuffs)
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 60, Targets: []*proto.Target{bossTarget()}},
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

// 2P (1300947): "Improves your chance to hit by 1%".
func TestManaflareTwoPieceAddsOnePercentHit(t *testing.T) {
	_, bare := newManaflareMage(t, 0, ForeverFrostTalents)
	_, two := newManaflareMage(t, 2, ForeverFrostTalents)
	_, one := newManaflareMage(t, 1, ForeverFrostTalents)
	if got := two.GetStat(stats.Hit) - bare.GetStat(stats.Hit); got != core.HitRatingPerHitChance {
		t.Errorf("two pieces add %v hit rating, want %v", got, core.HitRatingPerHitChance)
	}
	if one.GetStat(stats.Hit) != bare.GetStat(stats.Hit) {
		t.Error("one piece changed hit")
	}
}

// 3P (1301013): "Reduces the cooldown on your Counterspell spell by 5 sec".
func TestManaflareThreePieceShortensCounterspell(t *testing.T) {
	_, two := newManaflareMage(t, 2, ForeverFrostTalents)
	_, three := newManaflareMage(t, 3, ForeverFrostTalents)
	if got := two.Counterspell.CD.Duration - three.Counterspell.CD.Duration; got != 5*time.Second {
		t.Errorf("three pieces shorten Counterspell by %v, want 5s", got)
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
