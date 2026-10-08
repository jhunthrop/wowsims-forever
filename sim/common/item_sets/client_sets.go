package item_sets

import "github.com/wowsims/classic/sim/core"

// clientSet declares how one client item set is modelled: the bonus spells
// that are hand-written, and the ones the sim has no use for. Every other
// bonus spell of the set is a flat stat read from its row.
type clientSet struct {
	id     int32
	models map[int32]bonusModel
	noSim  map[int32]string
}

// registerClientSets registers each set through core.NewClientItemSet,
// building every hand-written bonus from its own bonus spell id.
func registerClientSets(sets ...clientSet) {
	for _, set := range sets {
		effects := make(map[int32]core.ApplyEffect, len(set.models))
		for bonusID, model := range set.models {
			effects[bonusID] = model(bonusID)
		}
		core.NewClientItemSet(core.ClientSetModel{ID: set.id, Effects: effects, NoSim: set.noSim})
	}
}

// Why a bonus is not modelled.
const (
	noSimStruckUtility = "proc on being struck that is a utility or survival effect (heal, shield, root/snare removal, disarm, silence, freeze, flee) with no damage, threat or mana"
	noSimMovement      = "movement, regeneration out of combat, or defence against effects the sim does not produce"
	noSimSelfHeal      = "heals the wearer on a spellcast; the sim measures no healing taken"
	noSimSpellReflect  = "reflects spells cast at the wearer; a sim boss casts none"
	noSimStun          = "stuns the target; a sim boss is immune to stun"
	noSimCosmetic      = "cosmetic disguise aura"
	noSimUnreachable   = "not reachable in Phase 1 (too few pieces obtainable)"
)
