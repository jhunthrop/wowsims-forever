package dpswarrior

import (
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"

	"github.com/wowsims/classic/sim/warrior"
)

// Since 1 October a queued Heroic Strike no longer lowers the off hand's
// miss chance: Blizzard removed the "HS queue trick" (change log of that
// date; Blizzard EU forum thread of 9 October). Queueing it must leave the
// dual wield miss penalty in place.
func TestQueuedHeroicStrikeKeepsTheDualWieldPenalty(t *testing.T) {
	built, sim := buildWarriorWithWeaponsAndSim(t, warrior.ForeverFuryTalents, 12784, 12784)
	if !built.AutoAttacks.IsDualWielding {
		t.Fatal("the test warrior is not dual wielding")
	}
	built.AddRage(sim, 100, built.NewRageMetrics(core.ActionID{OtherID: proto.OtherAction_OtherActionAttack}))
	built.HeroicStrikeQueue.Cast(sim, sim.Encounter.TargetUnits[0])
	if built.PseudoStats.DisableDWMissPenalty {
		t.Error("queueing Heroic Strike removed the off hand's dual wield miss penalty")
	}
}
