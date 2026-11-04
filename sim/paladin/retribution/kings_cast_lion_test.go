package retribution

import (
	"fmt"
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
	"github.com/wowsims/classic/sim/core/stats"
)

// A paladin's own cast of Blessing of Kings is the same +10% to every stat as
// Heart of the Lion's area buff (409583), and the two do not stack (the raid
// Kings already shares one category with the Lion, sim/core/buffs.go). So on
// a unit already in the Lion's area, the cast adds nothing; on a unit with no
// Lion it adds 10%.
func TestCastKingsDoesNotStackWithHeartOfTheLion(t *testing.T) {
	for _, c := range []struct {
		lion bool
		want float64
	}{{false, 1.10}, {true, 1.00}} {
		sim := core.NewSim(&proto.RaidSimRequest{
			Raid: core.SinglePlayerRaidProto(core.WithSpec(&proto.Player{
				Class: proto.Class_ClassPaladin, Race: proto.Race_RaceHuman, Level: 60,
				Equipment: &proto.EquipmentSpec{}, TalentsString: Phase45RetTalents,
				Rotation: consecrationRotation,
			}, PlayerOptionsSealofRighteousness), &proto.PartyBuffs{}, &proto.RaidBuffs{HeartOfTheLion: c.lion}, &proto.Debuffs{}),
			Encounter:  &proto.Encounter{Duration: 30, Targets: []*proto.Target{core.NewDefaultTarget()}},
			SimOptions: &proto.SimOptions{RandomSeed: 1, IsTest: true},
		}, simsignals.CreateSignals())
		sim.Reset()
		unit := &sim.Raid.Parties[0].Players[0].GetCharacter().Unit
		aura := unit.GetAura(fmt.Sprintf("Blessing of Kings (paladin %d)", unit.UnitIndex))
		if aura == nil {
			t.Fatalf("lion %v: no cast Blessing of Kings aura on the paladin", c.lion)
		}
		before := unit.GetStats()
		aura.Activate(sim)
		after := unit.GetStats()
		for _, stat := range []stats.Stat{stats.Strength, stats.Agility, stats.Stamina, stats.Intellect, stats.Spirit} {
			if got := after[stat] / before[stat]; math.Abs(got-c.want) > 0.002 {
				t.Errorf("lion %v: cast Kings moves %s by %.4fx, want %.2fx", c.lion, stat.StatName(), got, c.want)
			}
		}
		aura.Deactivate(sim)
		if restored := unit.GetStats(); restored[stats.Strength] != before[stats.Strength] {
			t.Errorf("lion %v: Strength %v after Kings fades, want %v", c.lion, restored[stats.Strength], before[stats.Strength])
		}
	}
}
