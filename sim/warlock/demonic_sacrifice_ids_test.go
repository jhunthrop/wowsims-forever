package warlock

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

// demonicSacrificeTalentsString is Demonic Sacrifice alone (Demonology
// field 10 of 19).
const demonicSacrificeTalentsString = "-0000000001"

func newSacrificeTestWarlock(t *testing.T, summon proto.WarlockOptions_Summon) (*core.Simulation, *Warlock) {
	t.Helper()
	player := core.WithSpec(
		&proto.Player{
			Class:         proto.Class_ClassWarlock,
			Race:          proto.Race_RaceOrc,
			Level:         60,
			Equipment:     &proto.EquipmentSpec{},
			TalentsString: demonicSacrificeTalentsString,
		},
		&proto.Player_Warlock{Warlock: &proto.Warlock{Options: &proto.WarlockOptions{
			Armor:  proto.WarlockOptions_NoArmor,
			Summon: summon,
		}}},
	)
	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  &proto.Encounter{Duration: 60, Targets: []*proto.Target{core.DefaultTargetProtoLvl60}},
		SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
	}, simsignals.CreateSignals())
	sim.Reset()
	sim.PrePull()
	for i := 0; i < 10 && sim.CurrentTime <= 0; i++ {
		sim.Step()
	}
	return sim, sim.Raid.Parties[0].Players[0].(WarlockAgent).GetWarlock()
}

// Forever's Demonic Sacrifice text: "Voidwalker: Restores 2% of your total
// Mana every 4 sec ... Felhunter: Restores 3% of your total Health every 4
// sec." The client's two regeneration buffs are 18792 Fel Energy (aura 21,
// power, 2 every 4000 ms) and 18790 Fel Stamina (aura 20, health, 3 every
// 4000 ms), so the Voidwalker's sacrifice is Fel Energy and the
// Felhunter's Fel Stamina. The amounts were right; the ids were crossed.
func TestDemonicSacrificeBuffsCarryTheClientIDs(t *testing.T) {
	for _, tc := range []struct {
		summon proto.WarlockOptions_Summon
		want   int32
	}{
		{proto.WarlockOptions_Voidwalker, 18792},
		{proto.WarlockOptions_Felhunter, 18790},
	} {
		sim, warlock := newSacrificeTestWarlock(t, tc.summon)
		sacrifice := warlock.GetSpell(core.ActionID{SpellID: 18788})
		if sacrifice == nil {
			t.Fatal("no Demonic Sacrifice spell")
		}
		sacrifice.ApplyEffects(sim, warlock.CurrentTarget, sacrifice)

		var active []int32
		for _, id := range []int32{18789, 18790, 18791, 18792} {
			if aura := warlock.GetAuraByID(core.ActionID{SpellID: id}); aura != nil && aura.IsActive() {
				active = append(active, id)
			}
		}
		if len(active) != 1 || active[0] != tc.want {
			t.Errorf("sacrificing %v leaves %v active, want only %d", tc.summon, active, tc.want)
		}
	}
}
