package warlock

import (
	"testing"

	"github.com/wowsims/classic/sim/common/clientsetbonus/clientsetbonustest"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

const demonheartLifeTapRow int32 = 1301715 // 5P: Life Tap +20% Mana

func demonheartWarlock(t *testing.T, pieces int) (*core.Simulation, *Warlock) {
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
	sim := clientsetbonustest.PrePulledSim(t, player, demonheartRaimentSetID, pieces)
	agent, ok := sim.Raid.Parties[0].Players[0].(WarlockAgent)
	if !ok {
		t.Fatal("the raid's first player is not a warlock agent")
	}
	return sim, agent.GetWarlock()
}

// manaFromOneLifeTap is the Mana one cast of the top rank restores to an
// empty mana bar.
func manaFromOneLifeTap(sim *core.Simulation, warlock *Warlock) float64 {
	spent := warlock.NewManaMetrics(core.ActionID{SpellID: 1})
	warlock.SpendMana(sim, warlock.CurrentMana(), spent)
	tap := warlock.LifeTap[len(warlock.LifeTap)-1]
	tap.ApplyEffects(sim, sim.Encounter.TargetUnits[0], tap)
	return warlock.CurrentMana()
}

// 2P (hit) and 4P (spell damage against Demons) are flat rows.
func TestDemonheartAutomaticBonusesMatchTheRows(t *testing.T) {
	_, bare := demonheartWarlock(t, 0)
	for _, pieces := range []int{2, 4} {
		_, worn := demonheartWarlock(t, pieces)
		clientsetbonustest.AssertAutomaticTotals(t, demonheartRaimentSetID, pieces, bare.GetCharacter(), worn.GetCharacter())
	}
}

// 5P: "Your Life Tap generates 20% more Mana at no additional Health cost".
func TestDemonheartFivePieceRaisesLifeTapMana(t *testing.T) {
	simFour, four := demonheartWarlock(t, 4)
	simFive, five := demonheartWarlock(t, 5)
	bonus := core.MustClientSpellRow(demonheartLifeTapRow).Effects[0].Points / 100

	base := manaFromOneLifeTap(simFour, four)
	got := manaFromOneLifeTap(simFive, five)
	if want := base * (1 + bonus); !floatsNearlyEqual(got, want) {
		t.Errorf("five pieces Life Tap for %v mana, want %v (four pieces: %v, +%v)", got, want, base, bonus)
	}
}
