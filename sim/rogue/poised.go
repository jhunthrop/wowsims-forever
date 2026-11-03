package rogue

import "github.com/wowsims/classic/sim/core"

// Poised for the rogue (Forever 1.60.1.70753): SkillLineAbility 59888
// teaches the passive 1328825 to every rogue (ClassMask 8, skill line 39,
// trainer) at level 18. Its buff 1328826 takes 10 Energy off the next
// ability in the family below. core.RegisterPoised has the mechanism.
const (
	poisedBuffSpellID     int32 = 1328826
	poisedEnergyReduction int64 = -10 // SpellEffect 1368039, EffectBasePointsF
)

// poisedFamilies is the buff's EffectSpellClassMask_0..3 (SpellEffect
// 1368039): every rogue ability that costs Energy.
var poisedFamilies = core.ClientClassMask{255727518, 2099456, 67108864, 0}

func (rogue *Rogue) registerPoised() {
	if int(rogue.Level) < PoisedLevel[0] {
		return
	}
	rogue.PoisedAura = rogue.RegisterPoised(core.PoisedConfig{
		BuffSpellID:   poisedBuffSpellID,
		CostReduction: poisedEnergyReduction,
		ClassMask:     rogueClassMasks.Resolve(poisedFamilies),
	})
}
