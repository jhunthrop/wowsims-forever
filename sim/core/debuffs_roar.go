package core

import (
	"math"
	"strconv"
	"time"
)

// DemoralizingRoarAuraOfStrength is a druid's own Demoralizing Roar, whose
// attack power reduction comes from the rank and level the client states
// (spellconst/druid.json), not from the Era figure DemoralizingRoarAura
// scales. It shares the "APReduction" exclusive effect, so the larger of any
// two reductions on a target is the one that applies.
func DemoralizingRoarAuraOfStrength(target *Unit, spellID int32, apReduction float64) *Aura {
	aura := target.GetOrRegisterAura(Aura{
		Label:    "DemoralizingRoar-" + strconv.Itoa(int(spellID)),
		ActionID: ActionID{SpellID: spellID},
		Duration: time.Second * 30,
	})
	apReductionEffect(aura, math.Floor(apReduction))
	return aura
}
