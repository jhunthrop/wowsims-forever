package priest

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

const priestHealFlags = SpellFlagPriest | core.SpellFlagHelpful | core.SpellFlagAPL

// PriestSpellMaskRenewedHope is every heal Renewed Hope names: Flash Heal,
// Lesser Heal, Heal, Greater Heal and Penance.
const PriestSpellMaskRenewedHope = PriestSpellMaskFlashHeal |
	PriestSpellMaskLesserHeal | PriestSpellMaskHeal | PriestSpellMaskGreaterHeal | PriestSpellMaskPenance

// RegisterHealingSpells registers every healing ability a priest of this
// level can cast. Only the healing specs call it: the Shadow spec never
// heals, and keeping the registration here leaves its spellbook alone.
func (priest *Priest) RegisterHealingSpells() {
	priest.registerWeakenedSoul()
	priest.registerLesserHeal()
	priest.registerHeal()
	priest.registerFlashHeal()
	priest.registerGreaterHeal()
	priest.registerRenew()
	priest.registerPrayerOfHealing()
	priest.registerPowerWordShield()
	priest.registerDesperatePrayer()
	priest.registerPenance()
	priest.registerHolyNova()
	priest.registerPrayerOfMending()
	priest.registerLightwell()
	priest.registerInnerFire()
	priest.registerRacialHeals()
}

// registerHealRanks registers every rank of table the priest's level has
// reached, as a slice indexed by rank (index 0 unused), the layout the
// damage spells use. A rank above the priest's level stays nil.
func (priest *Priest) registerHealRanks(table []healRank, newConfig func(rank int, entry healRank) core.SpellConfig) []*core.Spell {
	spells := make([]*core.Spell, len(table)+1)
	for i, entry := range table {
		if entry.level > int(priest.Level) {
			continue
		}
		spells[i+1] = priest.GetOrRegisterSpell(newConfig(i+1, entry))
	}
	return spells
}

// healSpellConfig is what every priest heal declares the same way: the
// client's cost, cast time, level and coefficient, and the roll
// conformance compares (ClientBaseDamage, here the base healing). The
// caller adds what the spell does.
func (priest *Priest) healSpellConfig(entry healRank, rank int, code int32, mask uint64) core.SpellConfig {
	return core.SpellConfig{
		ActionID:       core.ActionID{SpellID: entry.spellID},
		SpellCode:      code,
		ClassSpellMask: mask,
		SpellSchool:    core.SpellSchoolHoly,
		DefenseType:    core.DefenseTypeMagic,
		ProcMask:       core.ProcMaskSpellHealing,
		Flags:          priestHealFlags,

		RequiredLevel: entry.level,
		Rank:          rank,

		ManaCost: core.ManaCostOptions{
			FlatCost: entry.manaCost,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD:      core.GCDDefault,
				CastTime: time.Duration(entry.castMS) * time.Millisecond,
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: entry.coefficient,
		ClientBaseDamage: entry.effect.Range(int(priest.Level)),
	}
}

// roll is one draw of the rank's amount at the priest's level.
func (priest *Priest) roll(sim *core.Simulation, entry healRank) float64 {
	return entry.effect.Roll(sim, int(priest.Level))
}

// healTarget lands one heal of baseHealing on target. Every direct heal
// goes through here so Renewed Hope, which rides on the cast and not on
// the spell, reaches all the heals it names the same way: the crit bonus
// on the heal, then the cut to the target's Weakened Soul.
func (priest *Priest) healTarget(sim *core.Simulation, spell *core.Spell, target *core.Unit, baseHealing float64) *core.SpellResult {
	result := priest.healPulse(sim, spell, target, baseHealing)
	priest.shortenWeakenedSoul(sim, spell, target)
	return result
}

// healPulse is healTarget without the Weakened Soul cut: Penance's second
// and third pulses. Blizzard counts a Penance as one heal for that cut,
// whatever the talent text reads ("Not A Bug. This is a typo in the deep
// dive.", community bug tracker issue #288, 10 October 2026), so only the
// cast's first pulse takes time off Weakened Soul. The crit bonus is per
// heal and stays on every pulse.
func (priest *Priest) healPulse(sim *core.Simulation, spell *core.Spell, target *core.Unit, baseHealing float64) *core.SpellResult {
	hope := priest.renewedHopeCrit(spell, target)
	spell.BonusCritRating += hope
	result := spell.CalcAndDealHealing(sim, target, baseHealing, spell.OutcomeHealingCrit)
	spell.BonusCritRating -= hope
	return result
}

// partyOf is the units in target's party, target included. A unit that is
// not a raid player (an enemy, a stray pet) is its own party.
func partyOf(target *core.Unit) []*core.Unit {
	agent := target.Env.Raid.GetPlayerFromUnitIndex(target.UnitIndex)
	if agent == nil {
		return []*core.Unit{target}
	}
	players := agent.GetCharacter().Party.Players
	units := make([]*core.Unit, len(players))
	for i, player := range players {
		units[i] = &player.GetCharacter().Unit
	}
	return units
}
