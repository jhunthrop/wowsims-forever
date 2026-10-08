package druid

import (
	"github.com/wowsims/classic/sim/common/clientsetbonus"
	"github.com/wowsims/classic/sim/core"
)

// Forever's Tier 1 druid sets (client ItemSets 2112, 2113, 2114 and 2115,
// build 1.60.1.70009). The 2- and 4-piece bonuses are flat and applied from
// the client's rows; the 5-piece bonuses are spell mods and the 3-piece
// bonuses have no effect the sim measures.
const (
	grovekeeperRaimentSetID  int32 = 2112
	grovekeeperRageSetID     int32 = 2113
	grovekeeperEclipseSetID  int32 = 2114
	grovekeeperFerocitySetID int32 = 2115

	barkskinNoSimNote     = "reduces the cooldown of Barkskin, which only the bear kit registers; the restoration druid never casts it"
	rebirthNoSimNote      = "lets Rebirth be cast in Bear Form and shortens its cast time; the sim never resurrects"
	naturesGraspNoSimNote = "reduces the cooldown of Nature's Grasp, a root the sim never casts"
	hibernateNoSimNote    = "lets Hibernate be cast in Cat Form; the sim never casts it"
)

// tier1ClassMasks says which engine spells the client spell families of
// the Tier 1 5-piece bonuses are (the SpellClassOptions masks of
// Swiftmend, Berserk, Tiger's Fury and Insect Swarm).
var tier1ClassMasks = core.ClassMaskTable{
	{Client: core.ClientClassMask{0, 1 << 1}, Engine: DruidSpellMaskSwiftmend},
	{Client: core.ClientClassMask{0, 0, 1 << 6}, Engine: DruidSpellMaskBerserk},
	{Client: core.ClientClassMask{0, 0, 1 << 11}, Engine: DruidSpellMaskTigersFury},
	{Client: core.ClientClassMask{1 << 21}, Engine: DruidSpellMaskInsectSwarm},
}

var ItemSetGrovekeeperRaiment = core.NewClientItemSet(core.ClientSetModel{
	ID: grovekeeperRaimentSetID,
	// Reduces the cooldown on your Swiftmend spell by 3 sec.
	Effects: clientsetbonus.FivePieceMod(grovekeeperRaimentSetID, tier1ClassMasks),
	NoSim:   clientsetbonus.NoSimThreePiece(grovekeeperRaimentSetID, barkskinNoSimNote),
})

var ItemSetGrovekeeperRage = core.NewClientItemSet(core.ClientSetModel{
	ID: grovekeeperRageSetID,
	// Reduces the cooldown on your Berserk ability by 15 sec.
	Effects: clientsetbonus.FivePieceMod(grovekeeperRageSetID, tier1ClassMasks),
	NoSim:   clientsetbonus.NoSimThreePiece(grovekeeperRageSetID, rebirthNoSimNote),
})

var ItemSetGrovekeeperEclipse = core.NewClientItemSet(core.ClientSetModel{
	ID: grovekeeperEclipseSetID,
	Effects: map[int32]core.ApplyEffect{
		// Increases the duration of your Insect Swarm spell by 3 sec.
		clientsetbonus.SpellAt(grovekeeperEclipseSetID, clientsetbonus.FivePieces): insectSwarmExtraTicks(
			clientsetbonus.SpellAt(grovekeeperEclipseSetID, clientsetbonus.FivePieces)),
	},
	NoSim: clientsetbonus.NoSimThreePiece(grovekeeperEclipseSetID, naturesGraspNoSimNote),
})

var ItemSetGrovekeeperFerocity = core.NewClientItemSet(core.ClientSetModel{
	ID: grovekeeperFerocitySetID,
	// Reduces the cooldown on your Tiger's Fury ability by 3 sec.
	Effects: clientsetbonus.FivePieceMod(grovekeeperFerocitySetID, tier1ClassMasks),
	NoSim:   clientsetbonus.NoSimThreePiece(grovekeeperFerocitySetID, hibernateNoSimNote),
})

// insectSwarmExtraTicks lengthens Insect Swarm by the whole ticks that fit
// in the bonus's duration: a damage over time only deals damage on its
// ticks, and the engine's dot duration mod refuses a partial one, so the
// 3 seconds are three halves of a 2 second tick and the half tick (1 sec)
// carries nothing.
func insectSwarmExtraTicks(spellID int32) core.ApplyEffect {
	duration := clientsetbonus.TableMod(spellID, tier1ClassMasks)
	mod := core.SpellModConfig{
		Kind:      core.SpellMod_DotNumberOfTicks_Flat,
		ClassMask: duration.ClassMask,
		IntValue:  int64(duration.TimeValue / InsectSwarmTickLength),
	}
	return func(agent core.Agent) {
		agent.GetCharacter().AddStaticMod(mod)
	}
}
