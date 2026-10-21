package paladin

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// blessingOfLightHeal names the two heals Blessing of Light improves.
type blessingOfLightHeal int

const (
	blessingOfLightHolyLight blessingOfLightHeal = iota
	blessingOfLightFlashOfLight
	blessingOfLightHealKinds
)

// blessingOfLightBonus is the bonus healing the blessing gives each heal.
type blessingOfLightBonus [blessingOfLightHealKinds]float64

type blessingOfLightRank struct {
	spellID  int32
	level    int
	manaCost float64
	bonus    blessingOfLightBonus
}

const blessingOfLightDuration = time.Hour

// blessingOfLightRanks are trainables "Blessing of Light" (spellconst
// effects 0 and 1: Holy Light then Flash of Light).
var blessingOfLightRanks = []blessingOfLightRank{
	{spellID: 19977, level: 40, manaCost: 85, bonus: blessingOfLightBonus{210, 60}},
	{spellID: 19978, level: 50, manaCost: 110, bonus: blessingOfLightBonus{300, 85}},
	{spellID: 19979, level: 60, manaCost: 135, bonus: blessingOfLightBonus{400, 115}},
}

// greaterBlessingOfLight is trainables "Greater Blessing of Light": the
// top rank's bonus for the whole raid in one cast.
var greaterBlessingOfLight = blessingOfLightRank{
	spellID: 25890, level: 60, manaCost: 260, bonus: blessingOfLightBonus{400, 115},
}

// registerBlessingOfLight registers the blessing as a buff on the healed
// unit. Assumption: the client states the bonus as a bare amount ("up to
// 400") with no coefficient of its own, which is how it states every bonus
// healing source, so it counts as bonus healing for that spell and is
// scaled by the spell's coefficient (Holy Light 0.714, Flash of Light
// 0.429) like healing power. The client states no level penalty for a
// downranked heal either, so none is modelled.
func (paladin *Paladin) registerBlessingOfLight() {
	paladin.blessingOfLightAuras = paladin.NewRaidAuraArray(func(target *core.Unit) *core.Aura {
		return target.RegisterAura(core.Aura{
			Label:    "Blessing of Light",
			ActionID: core.ActionID{SpellID: blessingOfLightRanks[0].spellID},
			Duration: blessingOfLightDuration,
		})
	})
	paladin.blessingOfLightBonuses = make([]blessingOfLightBonus, len(paladin.Env.AllUnits))

	for i, rank := range blessingOfLightRanks {
		if paladin.Level < int32(rank.level) {
			break
		}
		paladin.registerBlessingOfLightCast(rank, i+1, false)
	}
	if paladin.Level >= int32(greaterBlessingOfLight.level) {
		paladin.registerBlessingOfLightCast(greaterBlessingOfLight, 0, true)
	}
}

func (paladin *Paladin) registerBlessingOfLightCast(rank blessingOfLightRank, rankNumber int, raidWide bool) {
	spell := paladin.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: rank.spellID},
		SpellSchool: core.SpellSchoolHoly,
		ProcMask:    core.ProcMaskEmpty,
		Flags:       core.SpellFlagHelpful | core.SpellFlagAPL | core.SpellFlagNoOnCastComplete,

		RequiredLevel: rank.level,
		Rank:          rankNumber,

		ManaCost: core.ManaCostOptions{FlatCost: rank.manaCost},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{GCD: core.GCDDefault},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, _ *core.Spell) {
			if !raidWide {
				paladin.applyBlessingOfLight(sim, target, rank.bonus)
				return
			}
			for _, unit := range paladin.Env.AllUnits {
				if unit.Type != core.EnemyUnit {
					paladin.applyBlessingOfLight(sim, unit, rank.bonus)
				}
			}
		},
	})
	// The blessing the paladin casts on themselves is the spell's buff.
	spell.RelatedSelfBuff = paladin.blessingOfLightAuras.Get(&paladin.Unit)
}

func (paladin *Paladin) applyBlessingOfLight(sim *core.Simulation, target *core.Unit, bonus blessingOfLightBonus) {
	paladin.blessingOfLightBonuses[target.UnitIndex] = bonus
	paladin.blessingOfLightAuras.Get(target).Activate(sim)
}

// blessingOfLightBonus is the flat healing the target's Blessing of Light
// adds to the given heal; zero for a target without one, and for a
// paladin that never registered the blessing.
func (paladin *Paladin) blessingOfLightBonus(target *core.Unit, heal blessingOfLightHeal) float64 {
	if paladin.blessingOfLightAuras == nil {
		return 0
	}
	aura := paladin.blessingOfLightAuras.Get(target)
	if aura == nil || !aura.IsActive() {
		return 0
	}
	return paladin.blessingOfLightBonuses[target.UnitIndex][heal]
}
