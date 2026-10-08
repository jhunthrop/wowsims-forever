package paladin

import (
	"github.com/wowsims/classic/sim/common/clientsetbonus"
	"github.com/wowsims/classic/sim/core"
)

// Forever's Tier 1 paladin sets (client ItemSets 2106, 2107 and 2108,
// build 1.60.1.70009). The 2- and 4-piece bonuses are flat and applied from
// the client's rows; the 5-piece bonuses are a spell mod (Judgement and
// Holy Shock cooldowns) and a shorter Forbearance.
const (
	justiceBattlegearSetID  int32 = 2106
	justiceArmorSetID       int32 = 2107
	justiceBattleplateSetID int32 = 2108

	hammerOfJusticeNoSimNote      = "reduces the cooldown of Hammer of Justice, a stun the sim never casts"
	blessingOfProtectionNoSimNote = "reduces the cooldown of Blessing of Protection, a utility ability the sim never casts"
	turnUndeadNoSimNote           = "shortens the cast time of Turn Undead, a crowd control the sim never casts"
)

// tier1ClassMasks says which engine spells the client spell families of
// the Tier 1 5-piece bonuses are (the SpellClassOptions masks of Judgement
// and Holy Shock). Holy Shock's family covers both the damage and the heal
// half of the engine's two spells.
var tier1ClassMasks = core.ClassMaskTable{
	{Client: core.ClientClassMask{1 << 23}, Engine: PaladinSpellMaskJudgement},
	{Client: core.ClientClassMask{1 << 21}, Engine: PaladinSpellMaskHolyShock | PaladinSpellMaskHolyShockHeal},
}

var ItemSetJusticeBattlegear = core.NewClientItemSet(core.ClientSetModel{
	ID: justiceBattlegearSetID,
	// Reduces the cooldown on your Judgement spell by 0.5 sec.
	Effects: clientsetbonus.FivePieceMod(justiceBattlegearSetID, tier1ClassMasks),
	NoSim:   clientsetbonus.NoSimThreePiece(justiceBattlegearSetID, hammerOfJusticeNoSimNote),
})

var ItemSetJusticeArmor = core.NewClientItemSet(core.ClientSetModel{
	ID: justiceArmorSetID,
	// Reduces the cooldown on your Holy Shock spell by 1 sec.
	Effects: clientsetbonus.FivePieceMod(justiceArmorSetID, tier1ClassMasks),
	NoSim:   clientsetbonus.NoSimThreePiece(justiceArmorSetID, blessingOfProtectionNoSimNote),
})

var battleplateForbearanceBonus = clientsetbonus.SpellAt(justiceBattleplateSetID, clientsetbonus.FivePieces)

var ItemSetJusticeBattleplate = core.NewClientItemSet(core.ClientSetModel{
	ID: justiceBattleplateSetID,
	Effects: map[int32]core.ApplyEffect{
		// Reduces the duration of Forbearance any time you gain it by 10 sec.
		battleplateForbearanceBonus: shortenForbearance(battleplateForbearanceBonus),
	},
	NoSim: clientsetbonus.NoSimThreePiece(justiceBattleplateSetID, turnUndeadNoSimNote),
})

// shortenForbearance reads the bonus row's dummy number (milliseconds,
// negative) and applies it to the length of the Forbearance debuff.
func shortenForbearance(spellID int32) core.ApplyEffect {
	change := clientsetbonus.DummyDuration(spellID)
	return func(agent core.Agent) {
		agent.(PaladinAgent).GetPaladin().forbearanceDurationMod += change
	}
}
