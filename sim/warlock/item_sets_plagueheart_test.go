package warlock

import (
	"testing"

	"github.com/wowsims/classic/sim/common/clientsetbonus/clientsetbonustest"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

const (
	plagueheartCorruptionPieces = 4
	plagueheartThreatPieces     = 6
	plagueheartLifeTapPieces    = 8
)

func plagueheartWarlock(t *testing.T, pieces int) (*core.Simulation, *Warlock) {
	t.Helper()
	player := core.WithSpec(
		&proto.Player{
			Class: proto.Class_ClassWarlock,
			Race:  proto.Race_RaceOrc,
			Level: 60,
			Buffs: core.FullBuffs.Player,
		},
		&proto.Player_Warlock{
			Warlock: &proto.Warlock{
				Options: &proto.WarlockOptions{
					Armor:       proto.WarlockOptions_NoArmor,
					Summon:      proto.WarlockOptions_NoSummon,
					WeaponImbue: proto.WarlockOptions_NoWeaponImbue,
				},
			},
		},
	)
	sim := clientsetbonustest.PrePulledSim(t, player, plagueheartSetID, pieces)
	agent, ok := sim.Raid.Parties[0].Players[0].(WarlockAgent)
	if !ok {
		t.Fatal("the raid's first player is not a warlock agent")
	}
	return sim, agent.GetWarlock()
}

func TestPlagueheartFourPieceRaisesCorruptionDamageByTheRowPercent(t *testing.T) {
	_, three := plagueheartWarlock(t, plagueheartCorruptionPieces-1)
	_, four := plagueheartWarlock(t, plagueheartCorruptionPieces)
	want := core.MustClientSpellRow(plagueheartCorruptionBonus).Effects[0].Points / 100
	if len(four.Corruption) == 0 {
		t.Fatal("the warlock has no Corruption")
	}
	for i, spell := range four.Corruption {
		if got := spell.DamageMultiplierAdditive - three.Corruption[i].DamageMultiplierAdditive; !floatsNearlyEqual(got, want) {
			t.Errorf("Corruption rank %d gains %v damage, want the client's %v", i+1, got, want)
		}
	}
}

func TestPlagueheartSixPieceCutsThreatOfTheNamedDots(t *testing.T) {
	_, five := plagueheartWarlock(t, plagueheartThreatPieces-1)
	_, six := plagueheartWarlock(t, plagueheartThreatPieces)
	want := 1 + core.MustClientSpellRow(plagueheartThreatBonus).Effects[1].Points/100
	named := map[string][2][]*core.Spell{
		"Corruption":     {five.Corruption, six.Corruption},
		"Immolate":       {five.Immolate, six.Immolate},
		"Curse of Agony": {five.CurseOfAgony, six.CurseOfAgony},
	}
	for name, pair := range named {
		if len(pair[1]) == 0 {
			t.Fatalf("the warlock has no %s", name)
		}
		for i := range pair[1] {
			if got := pair[1][i].ThreatMultiplier / pair[0][i].ThreatMultiplier; !floatsNearlyEqual(got, want) {
				t.Errorf("%s rank %d threat x%v, want the client's x%v", name, i+1, got, want)
			}
		}
	}
	shadowBolt := six.ShadowBolt[len(six.ShadowBolt)-1].ThreatMultiplier / five.ShadowBolt[len(five.ShadowBolt)-1].ThreatMultiplier
	if !floatsNearlyEqual(shadowBolt, 1) {
		t.Errorf("Shadow Bolt threat changed by x%v, the bonus does not name it", shadowBolt)
	}
}

// healthPaidByOneLifeTap is the damage one top-rank Life Tap deals to a
// warlock the boss is attacking.
func healthPaidByOneLifeTap(sim *core.Simulation, warlock *Warlock) float64 {
	boss := sim.Encounter.TargetUnits[0]
	boss.CurrentTarget = &warlock.Unit
	tap := warlock.LifeTap[len(warlock.LifeTap)-1]
	tap.ApplyEffects(sim, boss, tap)
	return tap.SpellMetrics[warlock.UnitIndex].TotalDamage
}

func TestPlagueheartEightPieceReducesLifeTapHealthCost(t *testing.T) {
	simSeven, seven := plagueheartWarlock(t, plagueheartLifeTapPieces-1)
	simEight, eight := plagueheartWarlock(t, plagueheartLifeTapPieces)
	discount := -core.MustClientSpellRow(plagueheartLifeTapBonus).Effects[0].Points / 100

	base := healthPaidByOneLifeTap(simSeven, seven)
	got := healthPaidByOneLifeTap(simEight, eight)
	if base <= 0 {
		t.Fatal("a tanking warlock pays no Health for Life Tap")
	}
	if want := base * (1 - discount); !floatsNearlyEqual(got, want) {
		t.Errorf("eight pieces pay %v Health, want %v (seven pieces: %v, -%v)", got, want, base, discount)
	}
}
