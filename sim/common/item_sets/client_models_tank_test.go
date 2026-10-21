package item_sets_test

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

const (
	enragingLightSet    int32 = 163
	enragingLightBonus  int32 = 1293674
	enragingLightProc         = "Enraging Light (damage taken)"
	enragingLightBuff         = "Enraging Light"
	enragingLightHoly   int32 = 1293675
	holyStrikes               = 4_000
	multiplierTolerance       = 0.25
)

func undeadTarget() *proto.Target {
	return &proto.Target{Level: core.DefaultTargetProtoLvl60.Level, Stats: core.DefaultTargetProtoLvl60.Stats, MobType: proto.MobType_MobTypeUndead}
}

// averageHolyStrike strikes the target with the buff's melee hits and
// returns the mean damage of the holy strikes they trigger.
func averageHolyStrike(w wornSet, buff *core.Aura, target *core.Unit) float64 {
	holy := w.character.GetSpell(core.ActionID{SpellID: enragingLightHoly})
	for i := 0; i < holyStrikes; i++ {
		w.swingAt(buff, core.ProcMaskMeleeMHAuto, target)
	}
	metrics := holy.SpellMetrics[target.UnitIndex]
	if metrics.Casts != holyStrikes {
		panic("every landed melee hit under Enraging Light must trigger a holy strike")
	}
	return metrics.TotalDamage / float64(metrics.Casts)
}

func TestEnragingLightAddsHolyDamageToMeleeHitsAndTriplesItAgainstUndead(t *testing.T) {
	if wear(t, warriorHost, enragingLightSet, 3).character.GetAura(enragingLightProc) != nil {
		t.Fatal("three pieces already carry Enraging Light")
	}
	w := wear(t, warriorHost, enragingLightSet, 4, core.DefaultTargetProtoLvl60, undeadTarget())
	proc := w.character.GetAura(enragingLightProc)
	if proc == nil {
		t.Fatal("four pieces do not carry Enraging Light")
	}
	buff := w.character.GetAura(enragingLightBuff)
	if buff.IsActive() {
		t.Fatal("Enraging Light is up before the wearer is struck")
	}
	fireUntil(t, buff.IsActive, func() { w.struck(proc) })

	humanoid, undead := w.sim.Encounter.AllTargetUnits[0], w.sim.Encounter.AllTargetUnits[1]
	base := averageHolyStrike(w, buff, humanoid)
	if base <= 0 {
		t.Fatal("the holy strike deals no damage")
	}
	multiplier := core.MustClientSpellRow(enragingLightBonus).Effects[0].Points
	if got := averageHolyStrike(w, buff, undead) / base; math.Abs(got-multiplier) > multiplier*multiplierTolerance {
		t.Errorf("holy damage against undead is x%.2f the damage against a humanoid, want the client's x%v", got, multiplier)
	}

}
