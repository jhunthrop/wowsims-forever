package paladin

import (
	"time"

	"github.com/wowsims/classic/sim/common/clientdamage"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// Paladin healing is registered for the healing spec only. The damage
// specs never cast a heal, and registering these ranks for them would add
// spells (and golden rows) to specs that cannot use them.

// healRank is one rank of a ranked heal as the client states it: the id
// the player casts, the level it is learned at, its flat mana cost and
// the roll of its heal effect (effect 10 of the client row).
type healRank struct {
	spellID  int32
	level    int
	manaCost float64
	heal     clientdamage.Effect
}

// directHeal is a ranked, hard-cast heal that lands on one target
// (Holy Light, Flash of Light).
type directHeal struct {
	ranks       []healRank
	castTime    time.Duration
	coefficient float64
	spellCode   int32
	classMask   uint64
	// blessing is which of Blessing of Light's two bonuses this heal reads.
	blessing blessingOfLightHeal
}

func (paladin *Paladin) isHealer() bool {
	return paladin.Spec == proto.Spec_SpecHolyPaladin
}

// registerHealing registers what only the healing spec casts. Holy Shock
// and Divine Favor are shared with the damage specs and register their
// healing half themselves.
func (paladin *Paladin) registerHealing() {
	if !paladin.isHealer() {
		return
	}
	paladin.registerHolyLight()
	paladin.registerFlashOfLight()
	paladin.registerLightsVigil()
	paladin.registerBlessingOfLight()
}

func (paladin *Paladin) registerDirectHeal(heal directHeal) {
	for i, rank := range heal.ranks {
		if paladin.Level < int32(rank.level) {
			break
		}
		paladin.registerDirectHealRank(heal, i+1, rank)
	}
}

func (paladin *Paladin) registerDirectHealRank(heal directHeal, rankNumber int, rank healRank) {
	casterLevel := int(paladin.Level)

	spell := paladin.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: rank.spellID},
		SpellSchool: core.SpellSchoolHoly,
		DefenseType: core.DefenseTypeMagic,
		ProcMask:    core.ProcMaskSpellHealing,
		Flags:       core.SpellFlagHelpful | core.SpellFlagAPL,

		RequiredLevel: rank.level,
		Rank:          rankNumber,

		SpellCode:      heal.spellCode,
		ClassSpellMask: heal.classMask,

		ManaCost: core.ManaCostOptions{FlatCost: rank.manaCost},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD:      core.GCDDefault,
				CastTime: heal.castTime,
			},
		},

		DamageMultiplier:  1,
		ThreatMultiplier:  1,
		BonusCoefficient:  heal.coefficient,
		ClientBaseDamage:  rank.heal.Range(casterLevel),
		PushbackReduction: paladin.spiritualFocusPushbackReduction(heal.classMask),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseHealing := rank.heal.Roll(sim, casterLevel) + paladin.blessingOfLightBonus(target, heal.blessing)
			spell.CalcAndDealHealing(sim, target, baseHealing, spell.OutcomeHealingCrit)
		},
	})
	paladin.markIlluminating(spell, rank.manaCost)
}
