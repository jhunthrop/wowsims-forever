package item_sets

import (
	"fmt"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// The client's "damage done versus creature type" aura and the creature
// type mask bit of the undead.
const (
	clientAuraModDamageVsCreature int32 = 168
	clientCreatureTypeUndead      int32 = 32
)

// undeadSlaying is a bonus that raises the wearer's damage against undead
// by the row's percentage. A set bonus is applied once and means every
// undead in the fight, so it reads the target pool and not the active
// prefix.
func undeadSlaying(bonusID int32) core.ApplyEffect {
	effect := auraEffectOf(bonusID, core.MustClientSpellRow(bonusID), clientAuraModDamageVsCreature)
	if effect.Misc0 != clientCreatureTypeUndead {
		panic(fmt.Sprintf("item_sets: client spell %d raises damage against creature mask %d, not the undead", bonusID, effect.Misc0))
	}
	multiplier := 1 + effect.Points/percent
	return func(agent core.Agent) {
		character := agent.GetCharacter()
		character.Env.RegisterPostFinalizeEffect(func() {
			for _, target := range character.Env.Encounter.AllTargetUnits {
				if target.MobType != proto.MobType_MobTypeUndead {
					continue
				}
				for _, attackTable := range character.AttackTables[target.UnitIndex] {
					attackTable.DamageDealtMultiplier *= multiplier
				}
			}
		})
	}
}
