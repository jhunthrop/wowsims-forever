package druid

import "github.com/wowsims/classic/sim/core"

// Poised for the druid (Forever 1.60.1.70753): SkillLineAbility 59889
// teaches the passive 1328829 to every druid (ClassMask 1024, skill line
// 134, trainer) at level 32. Its buff 1328827 takes 10 Energy off the next
// Cat Form ability in the families below. core.RegisterPoised has the
// mechanism (shared with the rogue's 1328826).
const (
	poisedLearnLevel            = 32
	poisedBuffSpellID     int32 = 1328827
	poisedEnergyReduction int64 = -10 // SpellEffect 1368040, EffectBasePointsF
)

// poisedFamilies is the buff's EffectSpellClassMask_0..3 (SpellEffect
// 1368040): Claw and Rake (0x1000), Shred (0x8000), Ravage (0x10000),
// Pounce (0x20000), Rip and Ferocious Bite (0x800000), and Cower (word 1,
// 0x20000000), which the engine does not model.
var poisedFamilies = core.ClientClassMask{8622080, 536870912, 0, 0}

// catEnergyClassMasks is each Energy ability's own SpellClassOptions mask
// (the top ranks: Claw 9850, Rake 9904, Shred 9830, Ravage 9867, Pounce
// 9827, Rip 9896, Ferocious Bite 31018; every rank of each carries the
// same family bit).
var catEnergyClassMasks = core.ClassMaskTable{
	{Client: core.ClientClassMask{1 << 12}, Engine: DruidSpellMaskClaw},
	{Client: core.ClientClassMask{1 << 12, 0, 0, 1 << 30}, Engine: DruidSpellMaskRake},
	{Client: core.ClientClassMask{1 << 15}, Engine: DruidSpellMaskShred},
	{Client: core.ClientClassMask{1 << 16}, Engine: DruidSpellMaskRavage},
	{Client: core.ClientClassMask{1 << 17}, Engine: DruidSpellMaskPounce},
	{Client: core.ClientClassMask{1 << 23, 0, 1 << 21}, Engine: DruidSpellMaskRip},
	{Client: core.ClientClassMask{1 << 23, 0, 0, 1 << 12}, Engine: DruidSpellMaskFerociousBite},
}

// PoisedEngineMask is the engine spells Poised reaches.
func PoisedEngineMask() uint64 {
	return catEnergyClassMasks.Resolve(poisedFamilies)
}

func (druid *Druid) registerPoised() {
	if int(druid.Level) < poisedLearnLevel {
		return
	}
	druid.PoisedAura = druid.RegisterPoised(core.PoisedConfig{
		BuffSpellID:   poisedBuffSpellID,
		CostReduction: poisedEnergyReduction,
		ClassMask:     PoisedEngineMask(),
	})
}
