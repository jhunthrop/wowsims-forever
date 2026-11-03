package healing

import (
	"testing"
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// Both healing priest rotations open with Lightwell 1.5 s cast at -2.5 s,
// and the priest buffs Inner Fire at -1 s. So Lightwell's cast ends on the
// moment Inner Fire is due.
//
// Two things went wrong there. With no haste, Lightwell's end sat on the
// global, whose action completes the cast; Inner Fire, cast first at the
// same moment, moved the global 1.5 s on and Lightwell completed (and
// spent its mana) 1.5 s late. With any slowing of casts Lightwell was
// still casting at -1 s and Inner Fire was skipped for the fight, which
// made a step of haste below zero look like a large healing gain (the
// weights tool's negative haste for Holy and Discipline).
func lightwellThenInnerFireSim(t *testing.T, haste float64) (*core.Simulation, *HealingPriest) {
	t.Helper()
	rotation := &proto.APLRotation{
		Type: proto.APLRotation_TypeAPL,
		PrepullActions: []*proto.APLPrepullAction{{
			Action:    &proto.APLAction{Action: &proto.APLAction_CastSpell{CastSpell: &proto.APLActionCastSpell{SpellId: &proto.ActionID{RawId: &proto.ActionID_SpellId{SpellId: 27871}}}}},
			DoAtValue: &proto.APLValue{Value: &proto.APLValue_Const{Const: &proto.APLValueConst{Val: "-2.5s"}}},
		}},
	}
	player := healer(60, HolyTalents, &proto.HealingPriest_Options{UseInnerFire: true}, rotation)
	bonus := raidHealerStats
	bonus[stats.SpellHaste] = haste
	player.BonusStats = &proto.UnitStats{Stats: bonus.ToFloatArray()}
	return agentSim(t, player)
}

func stepTo(sim *core.Simulation, at time.Duration) {
	for sim.CurrentTime < at && !sim.Step() {
	}
}

func TestLightwellCompletesOnTimeWhenInnerFireIsCastAtItsEnd(t *testing.T) {
	sim, priest := lightwellThenInnerFireSim(t, 0)
	stepTo(sim, 0)
	lightwell := priest.GetSpell(core.ActionID{SpellID: 27871})
	if lightwell == nil {
		t.Fatal("no Lightwell rank 3")
	}
	if got := lightwell.LastCastAt; got != -time.Second {
		t.Errorf("Lightwell completed at %v, want -1s (its 1.5 s cast from -2.5 s)", got)
	}
}

func TestInnerFireIsCastWhenAPrepullCastRunsPastItsMoment(t *testing.T) {
	for _, haste := range []float64{0, -1, 1} {
		sim, priest := lightwellThenInnerFireSim(t, haste)
		stepTo(sim, time.Second)
		if !priest.GetAuraByID(core.ActionID{SpellID: 10952}).IsActive() {
			t.Errorf("haste %+.0f%%: Inner Fire is not up at the pull", haste)
		}
	}
}
