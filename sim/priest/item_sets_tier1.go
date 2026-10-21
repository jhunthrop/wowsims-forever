package priest

import (
	"github.com/wowsims/classic/sim/common/clientsetbonus"
	"github.com/wowsims/classic/sim/core"
)

// Forever's Tier 1 priest sets (client ItemSets 2104 and 2105, build
// 1.60.1.70009). The 2- and 4-piece bonuses are flat and applied from the
// client's rows; the 5-piece cooldown reductions are spell mods.
const (
	vestmentsOfConvictionSetID int32 = 2104
	raimentsOfConvictionSetID  int32 = 2105

	vestmentsFearWardBonus int32 = 1301041
	raimentsShackleBonus   int32 = 1301045
	fearWardNoSimNote            = "reduces the cooldown of Fear Ward, a utility ability the sim never casts"
	shackleUndeadNoSimNote       = "lengthens Shackle Undead, a crowd control the sim never casts"
)

// tier1ClassMasks says which engine spells the client spell families of
// the Tier 1 5-piece bonuses are. The first row's two families are
// Penance's and Prayer of Mending's, which one bonus reaches together.
var tier1ClassMasks = core.ClassMaskTable{
	{Client: core.ClientClassMask{0, 1<<5 | 1<<23}, Engine: PriestSpellMaskPenance | PriestSpellMaskPrayerOfMending},
	{Client: core.ClientClassMask{0, 0, 0, 1 << 4}, Engine: PriestSpellMaskDevouringPlague},
}

var ItemSetVestmentsOfConviction = core.NewClientItemSet(core.ClientSetModel{
	ID: vestmentsOfConvictionSetID,
	// Reduces the cooldown on your Penance and Prayer of Mending spells by 1 sec.
	Effects: clientsetbonus.FivePieceMod(vestmentsOfConvictionSetID, tier1ClassMasks),
	NoSim:   map[int32]string{clientsetbonus.SpellAt(vestmentsOfConvictionSetID, clientsetbonus.ThreePieces): fearWardNoSimNote},
})

var ItemSetRaimentsOfConviction = core.NewClientItemSet(core.ClientSetModel{
	ID: raimentsOfConvictionSetID,
	// Reduces the cooldown on your Devouring Plague spell by 60 sec.
	Effects: clientsetbonus.FivePieceMod(raimentsOfConvictionSetID, tier1ClassMasks),
	NoSim:   map[int32]string{clientsetbonus.SpellAt(raimentsOfConvictionSetID, clientsetbonus.ThreePieces): shackleUndeadNoSimNote},
})
