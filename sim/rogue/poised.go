package rogue

import "github.com/wowsims/classic/sim/core"

// Poised for the rogue (Forever 1.60.1.70755): SkillLineAbility 59888
// teaches the passive 1328825 to every rogue (ClassMask 8, skill line 39,
// trainer) at level 18. Its buff 1328826 takes 10 Energy off the next
// ability in the family below. core.RegisterPoised has the mechanism.
const (
	poisedBuffSpellID     int32 = 1328826
	poisedEnergyReduction int64 = -10 // SpellEffect 1368039, EffectBasePointsF
)

// composureExtraReduction is Composure's rank -> extra Energy off Poised,
// index 0 unused: 1329265 e0 aura 107 misc 3 (effect 1 points) on mask_1
// 16, Poised's buff 1328826. The talent spell's base points are 0; the
// trait curve (TraitDefinitionEffectPoints 25221, curve 128092) is -5/-10.
var composureExtraReduction = [3]int64{0, -5, -10}

// poisedCostReduction is Poised's Energy discount with Composure.
func (rogue *Rogue) poisedCostReduction() int64 {
	return poisedEnergyReduction + composureExtraReduction[rankIndex(rogue.Talents.Composure, composureExtraReduction[:])]
}

// poisedFamilies is the buff's EffectSpellClassMask_0..3 (SpellEffect
// 1368039): every rogue ability that costs Energy.
var poisedFamilies = core.ClientClassMask{255727518, 2099456, 67108864, 0}

func (rogue *Rogue) registerPoised() {
	if int(rogue.Level) < PoisedLevel[0] {
		return
	}
	rogue.PoisedAura = rogue.RegisterPoised(core.PoisedConfig{
		BuffSpellID:   poisedBuffSpellID,
		CostReduction: rogue.poisedCostReduction(),
		ClassMask:     rogueClassMasks.Resolve(poisedFamilies),
	})
}
