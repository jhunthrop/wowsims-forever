package priest

import (
	"time"

	"github.com/wowsims/classic/sim/common/clientsetbonus"
	"github.com/wowsims/classic/sim/core"
)

// The Tier 2.5 and Tier 3 priest sets the client rows carry. Their flat
// bonuses are applied from the rows; the rest is modelled below or
// declared NoSim with a reason.
const (
	garmentsOfTheOracleSetID int32 = 507
	oracleHealSelfBonus      int32 = 26169
	oracleRenewPieces        int32 = 5

	vestmentsOfFaithSetID   int32 = 525
	faithRenewCostPieces    int32 = 2
	faithGreaterHealBonus   int32 = 28809
	faithHealingThreatBonus int32 = 28808
	faithEpiphanyBonus      int32 = 28802
	faithEpiphanyBuff       int32 = 28804

	oracleHealSelfNote     = "heals the wearer for a share of a heal; the sim measures no healing taken"
	faithAbsorbNote        = "shields the healed ally; the sim measures no damage taken by allies"
	faithHealingThreatNote = "lowers the threat of healing spells; the sim measures no healing threat"

	percentDivisor = 100.0

	// The proc's label differs from the buff's: both spells are named Epiphany.
	epiphanyTriggerSuffix = " (spellcast)"

	// The bit of the client family mask that is Renew (the bonus rows'
	// SpellClassMask word 0).
	renewClientFamilyBit = 1 << 6
)

// setBonusClassMasks says which engine spell the client family of the
// Renew bonuses (its duration and its mana cost) is.
var setBonusClassMasks = core.ClassMaskTable{
	{Client: core.ClientClassMask{renewClientFamilyBit}, Engine: PriestSpellMaskRenew},
}

var ItemSetGarmentsOfTheOracle = core.NewClientItemSet(core.ClientSetModel{
	ID: garmentsOfTheOracleSetID,
	Effects: map[int32]core.ApplyEffect{
		// Increases the duration of your Renew spell by 3 sec.
		clientsetbonus.SpellAt(garmentsOfTheOracleSetID, oracleRenewPieces): clientsetbonus.StaticMod(garmentsOfTheOracleSetID, oracleRenewPieces, setBonusClassMasks),
	},
	NoSim: map[int32]string{oracleHealSelfBonus: oracleHealSelfNote},
})

var ItemSetVestmentsOfFaith = core.NewClientItemSet(core.ClientSetModel{
	ID: vestmentsOfFaithSetID,
	Effects: map[int32]core.ApplyEffect{
		// Reduces the mana cost of your Renew spell by 12%.
		clientsetbonus.SpellAt(vestmentsOfFaithSetID, faithRenewCostPieces): clientsetbonus.StaticMod(vestmentsOfFaithSetID, faithRenewCostPieces, setBonusClassMasks),
		// Each spell you cast can trigger an Epiphany, increasing your
		// mana regeneration for 30 sec.
		faithEpiphanyBonus: applyEpiphany,
	},
	NoSim: map[int32]string{
		faithGreaterHealBonus:   faithAbsorbNote,
		faithHealingThreatBonus: faithHealingThreatNote,
	},
})

// applyEpiphany is a chance on every spellcast, damage or healing, to raise
// the wearer's mana regeneration for the buff spell's duration.
func applyEpiphany(agent core.Agent) {
	character := agent.GetCharacter()
	bonus := core.MustClientSpellRow(faithEpiphanyBonus)
	buff := core.MustClientSpellRow(faithEpiphanyBuff)
	flat, ok := core.DecodeClientFlatBonus(buff)
	if !ok {
		panic("priest: the Epiphany buff is not a flat stat bonus")
	}
	epiphany := character.NewTemporaryStatsAura(buff.Name, core.ActionID{SpellID: faithEpiphanyBuff}, flat.Stats,
		time.Duration(buff.DurationMS)*time.Millisecond)
	core.MakeProcTriggerAura(&character.Unit, core.ProcTrigger{
		Name:       bonus.Name + epiphanyTriggerSuffix,
		ActionID:   core.ActionID{SpellID: faithEpiphanyBonus},
		Callback:   core.CallbackOnCastComplete,
		ProcMask:   core.ProcMaskSpellDamage | core.ProcMaskSpellHealing,
		ProcChance: float64(bonus.ProcChance) / percentDivisor,
		Handler: func(sim *core.Simulation, _ *core.Spell, _ *core.SpellResult) {
			epiphany.Activate(sim)
		},
	})
}
