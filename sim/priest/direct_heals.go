package priest

import (
	"time"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

// desperatePrayerCooldown is the client's category cooldown for every rank
// of Desperate Prayer.
const desperatePrayerCooldown = 10 * time.Minute

// singleTargetHeal is a ranked direct heal that lands on its target and
// does nothing else: Lesser Heal, Heal, Flash Heal and Greater Heal.
type singleTargetHeal struct {
	code  int32
	mask  uint64
	table []healRank
}

func (priest *Priest) registerSingleTargetHeal(heal singleTargetHeal) []*core.Spell {
	return priest.registerHealRanks(heal.table, func(rank int, entry healRank) core.SpellConfig {
		config := priest.healSpellConfig(entry, rank, heal.code, heal.mask)
		config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			priest.healTarget(sim, spell, target, priest.roll(sim, entry))
		}
		return config
	})
}

func (priest *Priest) registerLesserHeal() {
	priest.LesserHeal = priest.registerSingleTargetHeal(singleTargetHeal{
		code: SpellCode_PriestLesserHeal, mask: PriestSpellMaskLesserHeal, table: lesserHealRanks,
	})
}

func (priest *Priest) registerHeal() {
	priest.Heal = priest.registerSingleTargetHeal(singleTargetHeal{
		code: SpellCode_PriestHeal, mask: PriestSpellMaskHeal, table: healRanks,
	})
}

func (priest *Priest) registerFlashHeal() {
	priest.FlashHeal = priest.registerSingleTargetHeal(singleTargetHeal{
		code: SpellCode_PriestFlashHeal, mask: PriestSpellMaskFlashHeal, table: flashHealRanks,
	})
}

func (priest *Priest) registerGreaterHeal() {
	priest.GreaterHeal = priest.registerSingleTargetHeal(singleTargetHeal{
		code: SpellCode_PriestGreaterHeal, mask: PriestSpellMaskGreaterHeal, table: greaterHealRanks,
	})
}

// registerPrayerOfHealing heals the whole party of the target. Each member
// draws its own roll and its own crit, as a single-target heal would.
func (priest *Priest) registerPrayerOfHealing() {
	priest.PrayerOfHealing = priest.registerHealRanks(prayerOfHealingRanks, func(rank int, entry healRank) core.SpellConfig {
		config := priest.healSpellConfig(entry, rank, SpellCode_PriestPrayerOfHealing, PriestSpellMaskPrayerOfHealing)
		config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			for _, member := range partyOf(target) {
				priest.healTarget(sim, spell, member, priest.roll(sim, entry))
			}
		}
		return config
	})
}

// registerDesperatePrayer is the priest's own emergency heal. It is cast
// on the priest whatever target the caller names, and every rank shares
// one cooldown. Only a dwarf learns it: its learn rows (13908, 19243,
// skill line 56) carry RaceMasks 4.
func (priest *Priest) registerDesperatePrayer() {
	if priest.Race != proto.Race_RaceDwarf {
		return
	}
	cooldown := core.Cooldown{Timer: priest.NewTimer(), Duration: desperatePrayerCooldown}
	priest.DesperatePrayer = priest.registerHealRanks(desperatePrayerRanks, func(rank int, entry healRank) core.SpellConfig {
		config := priest.healSpellConfig(entry, rank, SpellCode_PriestDesperatePrayer, PriestSpellMaskDesperatePrayer)
		config.Cast.CD = cooldown
		config.ApplyEffects = func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
			priest.healTarget(sim, spell, &priest.Unit, priest.roll(sim, entry))
		}
		return config
	})
}
